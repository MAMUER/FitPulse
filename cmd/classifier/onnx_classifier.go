package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// defaultFeatureKeys — базовый набор признаков (4 метрики), используемый,
// если в файле весов нет списка feature_keys (обратная совместимость).
var defaultFeatureKeys = []string{"heart_rate", "hrv", "spo2", "temperature"}

type ONNXClassifier struct {
	Labels      []int
	Weights     []float64
	Intercepts  []float64
	MultiClass  int
	featureKeys []string
	featureMin  []float64
	featureMax  []float64
}

type ONNXWeights struct {
	Labels      []int     `json:"labels"`
	Weights     []float64 `json:"weights"`
	Intercepts  []float64 `json:"intercepts,omitempty"`
	MultiClass  int       `json:"multi_class"`
	FeatureKeys []string  `json:"feature_keys,omitempty"`
	FeatureMin  []float64 `json:"feature_min,omitempty"`
	FeatureMax  []float64 `json:"feature_max,omitempty"`
}

func NewONNXClassifier(weightsPath string) (*ONNXClassifier, error) {
	clean := filepath.Clean(weightsPath)
	if strings.Contains(clean, "..") {
		return nil, errors.New("read onnx weights: invalid path")
	}
	data, err := os.ReadFile(clean)
	if err != nil {
		return nil, fmt.Errorf("read onnx weights: %w", err)
	}

	var w ONNXWeights
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("parse onnx weights: %w", err)
	}

	featureKeys := defaultFeatureKeys
	if keys := validFeatureKeys(w.FeatureKeys); len(keys) > 0 {
		featureKeys = keys
	}

	return &ONNXClassifier{
		Labels:      w.Labels,
		Weights:     w.Weights,
		Intercepts:  w.Intercepts,
		MultiClass:  w.MultiClass,
		featureKeys: featureKeys,
		featureMin:  w.FeatureMin,
		featureMax:  w.FeatureMax,
	}, nil
}

// validFeatureKeys отбрасывает пустые/повторяющиеся ключи.
func validFeatureKeys(keys []string) []string {
	if len(keys) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func (c *ONNXClassifier) Predict(features map[string]float64) (int, float64, error) {
	if len(features) == 0 {
		return 0, 0, errors.New("empty features")
	}

	x := c.extractFeatures(features)

	numClasses := len(c.Labels)
	numFeatures := len(c.featureKeys)
	if !c.isValidConfig(numClasses, numFeatures) {
		return 0, 0, errors.New("unsupported onnx model configuration")
	}

	scores := c.computeScores(x, numClasses, numFeatures)
	best := c.findBestClass(scores)
	confidence := c.confidence(scores)

	return c.Labels[best], confidence, nil
}

// extractFeatures применяет min-max нормализацию, если она сохранена в весах.
// Без этого модель, обученная на [0,1], получает сырые ЧСС/SpO2 и всегда
// выдаёт один и тот же класс.
func (c *ONNXClassifier) extractFeatures(features map[string]float64) []float64 {
	x := make([]float64, len(c.featureKeys))
	for i, key := range c.featureKeys {
		value := features[key]
		if i < len(c.featureMin) && i < len(c.featureMax) {
			span := c.featureMax[i] - c.featureMin[i]
			if span > 0 {
				value = (value - c.featureMin[i]) / span
			}
		}
		x[i] = value
	}
	return x
}

func (c *ONNXClassifier) isValidConfig(numClasses, numFeatures int) bool {
	return c.MultiClass == 1 && numClasses > 1 && len(c.Weights) == numClasses*numFeatures
}

func (c *ONNXClassifier) computeScores(x []float64, numClasses, numFeatures int) []float64 {
	scores := make([]float64, numClasses)
	for cls := 0; cls < numClasses; cls++ {
		offset := cls * numFeatures
		sum := 0.0
		if cls < len(c.Intercepts) {
			sum = c.Intercepts[cls]
		}
		for j := 0; j < numFeatures; j++ {
			sum += c.Weights[offset+j] * x[j]
		}
		scores[cls] = sum
	}
	return scores
}

func (c *ONNXClassifier) findBestClass(scores []float64) int {
	best := 0
	for i := 1; i < len(scores); i++ {
		if scores[i] > scores[best] {
			best = i
		}
	}
	return best
}

// confidence возвращает вероятность выбранного класса. Для мультиклассовой
// модели (multi_class=1) используется softmax — это вероятностная оценка,
// согласованная с логистической регрессией. Иначе — сигмоида.
func (c *ONNXClassifier) confidence(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	if c.MultiClass == 1 {
		peak := scores[0]
		for _, v := range scores[1:] {
			if v > peak {
				peak = v
			}
		}
		total := 0.0
		exps := make([]float64, len(scores))
		for i, v := range scores {
			exps[i] = math.Exp(v - peak)
			total += exps[i]
		}
		best := c.findBestClass(scores)
		if total <= 0 {
			return 0
		}
		prob := exps[best] / total
		return math.Max(0.0, math.Min(1.0, prob))
	}
	return c.sigmoidConfidence(scores[c.findBestClass(scores)])
}

func (c *ONNXClassifier) sigmoidConfidence(maxScore float64) float64 {
	confidence := 1.0 / (1.0 + math.Exp(-maxScore))
	return math.Max(0.0, math.Min(1.0, confidence))
}

func (c *ONNXClassifier) FeatureKeys() []string {
	keys := make([]string, len(c.featureKeys))
	copy(keys, c.featureKeys)
	return keys
}

// Методы интерфейса scoreModel (линейная модель).

func (c *ONNXClassifier) NumClasses() int   { return len(c.Labels) }
func (c *ONNXClassifier) NumFeatures() int  { return len(c.featureKeys) }
func (c *ONNXClassifier) LabelsList() []int { return c.Labels }

func (c *ONNXClassifier) ExtractFeatures(features map[string]float64) []float64 {
	return c.extractFeatures(features)
}

func (c *ONNXClassifier) RawScores(x []float64) []float64 {
	numClasses := len(c.Labels)
	numFeatures := len(c.featureKeys)
	if !c.isValidConfig(numClasses, numFeatures) {
		return nil
	}
	return c.computeScores(x, numClasses, numFeatures)
}

func (c *ONNXClassifier) Confidence(scores []float64) float64 {
	return c.confidence(scores)
}

var getwd = os.Getwd

// defaultTreeModelPath возвращает путь к древовидной модели HistGradientBoosting.
func defaultTreeModelPath() string {
	cwd, err := getwd()
	if err != nil {
		cwd = "."
	}
	root := cwd
	if base := filepath.Base(cwd); base == "cmd" || base == "classifier" {
		root = filepath.Join(cwd, "..", "..")
	}
	abs, absErr := filepath.Abs(filepath.Join(root, "models", "classifier_trees.json"))
	if absErr != nil {
		return filepath.Join("models", "classifier_trees.json")
	}
	return abs
}

func DefaultONNXWeightsPath() string {
	// Относительный путь от корня репозитория; возвращаем абсолютный, чтобы
	// проверка на path traversal в NewONNXClassifier не отклоняла легитимный
	// путь (../.. из cmd/classifier).
	cwd, err := getwd()
	if err != nil {
		cwd = "."
	}
	root := cwd
	base := filepath.Base(cwd)
	if base == "cmd" || base == "classifier" {
		root = filepath.Join(cwd, "..", "..")
	}
	abs, absErr := filepath.Abs(filepath.Join(root, "models", "classifier_weights.json"))
	if absErr != nil {
		return filepath.Join("models", "classifier_weights.json")
	}
	return abs
}
