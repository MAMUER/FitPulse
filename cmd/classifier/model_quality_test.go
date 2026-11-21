package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/logger"
)

// canonicalStates — эталонные физиологические состояния. Модель обучается на
// rule-based разметке тех же состояний, поэтому ML-путь обязан их различать.
// Если тест падает, значит модель вырождена (всегда один класс) или в образ не
// попали веса/нормализация.
var canonicalStates = []struct {
	name     string
	features map[string]float64
	expected int
}{
	{"recovery", map[string]float64{"heart_rate": 60, "hrv": 90, "spo2": 98.5, "temperature": 36.5, "sleep_hours": 8}, 0},
	{"endurance_basic", map[string]float64{"heart_rate": 135, "hrv": 50, "spo2": 97.5, "temperature": 36.7, "sleep_hours": 7}, 1},
	{"endurance_threshold", map[string]float64{"heart_rate": 160, "hrv": 30, "spo2": 96, "temperature": 36.9, "sleep_hours": 7}, 2},
	{"power_hiit", map[string]float64{"heart_rate": 185, "hrv": 18, "spo2": 95, "temperature": 37.2, "sleep_hours": 7}, 3},
	{"overtraining", map[string]float64{"heart_rate": 100, "hrv": 18, "spo2": 98, "temperature": 36.6, "sleep_hours": 4}, 4},
	{"illness", map[string]float64{"heart_rate": 110, "hrv": 18, "spo2": 93.5, "temperature": 38.5, "sleep_hours": 6}, 5},
	{"unknown", map[string]float64{"heart_rate": 120, "hrv": 40, "spo2": 91.5, "temperature": 36.8, "sleep_hours": 7}, 6},
}

func loadRepoClassifier(t *testing.T) *ONNXClassifier {
	t.Helper()
	clf, err := NewONNXClassifier(DefaultONNXWeightsPath())
	require.NoError(t, err, "модель classifier_weights.json должна загружаться")
	return clf
}

// TestClassifierDiscriminatesStates — главный артефактный тест: модель обязана
// различать функциональные состояния, а не выдавать один класс с conf=1.0.
func TestClassifierDiscriminatesStates(t *testing.T) {
	clf := loadRepoClassifier(t)
	require.NotEmpty(t, clf.featureMin, "в весах должна быть нормализация feature_min")

	seen := map[int]bool{}
	for _, tc := range canonicalStates {
		got, conf, err := clf.Predict(tc.features)
		require.NoError(t, err, tc.name)
		assert.Equalf(t, tc.expected, got,
			"%s: ожидался класс %d, получен %d (conf=%.3f)", tc.name, tc.expected, got, conf)
		assert.GreaterOrEqual(t, conf, 0.0)
		assert.LessOrEqual(t, conf, 1.0)
		seen[got] = true
	}

	assert.GreaterOrEqual(t, len(seen), 5,
		"модель вырождена: распознано только %d классов из %d", len(seen), len(canonicalStates))
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

// TestClassifyHandlerUsesMLModel — сквозной тест обработчика: при наличии
// модели ответ содержит model="ml" и валидный класс.
func TestClassifyHandlerUsesMLModel(t *testing.T) {
	clf := loadRepoClassifier(t)
	s := &classifierServer{log: logger.New("classifier"), onnxClassifier: clf}

	body := `{"user_id":"u1","physiological_data":{"heart_rate":185,"heart_rate_variability":18,"spo2":95,"temperature":37.2,"sleep_hours":7},"user_profile":{"age":30}}`
	req := httptest.NewRequest(http.MethodPost, "/classify", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.classifyHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp classifyResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, modelML, resp.Model, "при загруженной модели должен использоваться ML-путь")
	assert.NotEmpty(t, resp.State)
	assert.NotEmpty(t, resp.PredictedClass)
}
