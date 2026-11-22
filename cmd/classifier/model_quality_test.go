package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/logger"
)

// goldenSet — эталонные предсказания, сгенерированные Python-моделью
// (scripts/export_go_model.py). Проверяет, что Go-инференс (нормализация,
// импутация, обход деревьев) даёт тот же результат, что обучение.
type goldenSet struct {
	FeatureKeys []string `json:"feature_keys"`
	Classes     []string `json:"classes"`
	Cases       []struct {
		Features map[string]float64 `json:"features"`
		Class    int                `json:"class"`
	} `json:"cases"`
}

func defaultGoldenPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	root := cwd
	if base := filepath.Base(cwd); base == "cmd" || base == "classifier" {
		root = filepath.Join(cwd, "..", "..")
	}
	abs, absErr := filepath.Abs(filepath.Join(root, "models", "classifier_golden.json"))
	if absErr != nil {
		return filepath.Join("models", "classifier_golden.json")
	}
	return abs
}

func loadGoldenSet(t *testing.T) goldenSet {
	t.Helper()
	data, err := os.ReadFile(defaultGoldenPath())
	require.NoError(t, err, "classifier_golden.json должен существовать (scripts/export_go_model.py)")
	var golden goldenSet
	require.NoError(t, json.Unmarshal(data, &golden))
	require.NotEmpty(t, golden.Cases)
	return golden
}

func loadRepoClassifier(t *testing.T) *ONNXClassifier {
	t.Helper()
	clf, err := NewONNXClassifier(DefaultONNXWeightsPath())
	require.NoError(t, err, "модель classifier_weights.json должна загружаться")
	return clf
}

// TestTreeModelMatchesGoldenSet — главный артефактный тест: Go должен
// воспроизводить Python-модель на эталонной выборке (допуск 2% на float).
func TestTreeModelMatchesGoldenSet(t *testing.T) {
	model, err := NewTreeModel(defaultTreeModelPath())
	require.NoError(t, err, "classifier_trees.json должен загружаться (scripts/export_go_model.py)")
	require.NotEmpty(t, model.featureMin, "в модели должна быть нормализация")

	golden := loadGoldenSet(t)
	seen := map[int]bool{}
	mismatches := 0

	for i, tc := range golden.Cases {
		got, conf, predErr := model.Predict(tc.Features)
		require.NoError(t, predErr, "case %d", i)
		assert.GreaterOrEqual(t, conf, 0.0)
		assert.LessOrEqual(t, conf, 1.0)
		if got != tc.Class {
			mismatches++
			t.Logf("case %d (%v): ожидался %d, получен %d (conf=%.3f)",
				i, tc.Features, tc.Class, got, conf)
		}
		seen[got] = true
	}

	allowed := int(float64(len(golden.Cases)) * 0.02)
	assert.LessOrEqual(t, mismatches, allowed,
		"допустимо не более 2%% расхождений, получено %d из %d", mismatches, len(golden.Cases))
	assert.GreaterOrEqual(t, len(seen), 5,
		"модель вырождена: распознано только %d классов", len(seen))
}

// TestClassifierNormalizationIsApplied проверяет, что сырые значения
// канонизируются в [0,1] до подачи в веса (без этого модель видела 160 вместо
// 0.75 и всегда выдавала один класс).
func TestClassifierNormalizationIsApplied(t *testing.T) {
	clf := loadRepoClassifier(t)
	require.NotEmpty(t, clf.featureMin)
	require.NotEmpty(t, clf.featureMax)

	low := clf.extractFeatures(map[string]float64{"heart_rate": clf.featureMin[0], "hrv": 50, "spo2": 98, "temperature": 36.6})
	high := clf.extractFeatures(map[string]float64{"heart_rate": clf.featureMax[0], "hrv": 50, "spo2": 98, "temperature": 36.6})

	assert.InDelta(t, 0.0, low[0], 1e-9)
	assert.InDelta(t, 1.0, high[0], 1e-9)
}

// TestHealthReportsActualModelState — /health не должен врать: если модель не
// загружена, model_loaded=false (раньше всегда стояло true).
func TestHealthReportsActualModelState(t *testing.T) {
	t.Run("without model", func(t *testing.T) {
		s := &classifierServer{}
		rec := httptest.NewRecorder()
		s.healthHandler(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
		require.Equal(t, http.StatusOK, rec.Code)

		var resp healthResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.False(t, resp.ModelLoaded)
		assert.False(t, resp.ScalerLoaded)
	})

	t.Run("with model", func(t *testing.T) {
		clf := loadRepoClassifier(t)
		s := &classifierServer{onnxClassifier: clf}
		rec := httptest.NewRecorder()
		s.healthHandler(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

		var resp healthResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.True(t, resp.ModelLoaded)
		assert.True(t, resp.ScalerLoaded)
	})
}

// TestClassifyHandlerUsesMLModel — ML подключается только для unknown:
// правила не могут классифицировать низкий SpO2, и здесь решает модель.
func TestClassifyHandlerUsesMLModel(t *testing.T) {
	tree, err := NewTreeModel(defaultTreeModelPath())
	require.NoError(t, err)
	s := &classifierServer{log: logger.New("classifier"), treeModel: tree}

	body := `{"user_id":"u1","physiological_data":{"heart_rate":120,"heart_rate_variability":40,"spo2":91.5,"temperature":36.8,"sleep_hours":7},"user_profile":{"age":30}}`
	req := httptest.NewRequest(http.MethodPost, "/classify", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.classifyHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp classifyResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, modelML, resp.Model, "для unknown должна использоваться ML-модель")
	assert.NotEmpty(t, resp.State)
	assert.NotEmpty(t, resp.PredictedClass)
}

// TestRuleBasedClassifierHandlesKnownStates — правила обязаны корректно
// классифицировать типовые состояния (ML здесь не участвует).
func TestRuleBasedClassifierHandlesKnownStates(t *testing.T) {
	s := &classifierServer{log: logger.New("classifier")}

	cases := []struct {
		name     string
		body     string
		expected string
	}{
		{"recovery", `{"physiological_data":{"heart_rate":60,"heart_rate_variability":90,"spo2":98,"temperature":36.5,"sleep_hours":8}}`, "recovery"},
		{"endurance_basic", `{"physiological_data":{"heart_rate":135,"heart_rate_variability":50,"spo2":97,"temperature":36.7,"sleep_hours":7}}`, "endurance_basic"},
		{"endurance_threshold", `{"physiological_data":{"heart_rate":160,"heart_rate_variability":30,"spo2":96,"temperature":36.9,"sleep_hours":7}}`, "endurance_threshold"},
		{"power_hiit", `{"physiological_data":{"heart_rate":185,"heart_rate_variability":18,"spo2":95,"temperature":37.2,"sleep_hours":7}}`, "power_hiit"},
		{"overtraining", `{"physiological_data":{"heart_rate":100,"heart_rate_variability":18,"spo2":98,"temperature":36.6,"sleep_hours":4}}`, "overtraining"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/classify", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			s.classifyHandler(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			var resp classifyResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			assert.Equal(t, tc.expected, resp.PredictedClass)
			assert.Equal(t, modelRuleBased, resp.Model)
		})
	}
}
