package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ONNXClassifier loads a simple linear model exported from Python
// and performs inference in pure Go without external ONNX runtime.
type ONNXClassifier struct {
	Labels      []int
	Weights     []float64
	MultiClass  int
	featureKeys []string
}

// ONNXWeights mirrors the JSON structure produced by train_classifier.py.
type ONNXWeights struct {
	Labels     []int     `json:"labels"`
	Weights    []float64 `json:"weights"`
	MultiClass int       `json:"multi_class"`
}

// NewONNXClassifier loads model weights from the given JSON path.
func NewONNXClassifier(weightsPath string) (*ONNXClassifier, error) {
	data, err := os.ReadFile(weightsPath)
	if err != nil {
		return nil, fmt.Errorf("read onnx weights: %w", err)
	}

	var w ONNXWeights
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("parse onnx weights: %w", err)
	}

	featureKeys := []string{"heart_rate", "hrv", "spo2", "temperature"}

	return &ONNXClassifier{
		Labels:      w.Labels,
		Weights:     w.Weights,
		MultiClass:  w.MultiClass,
		featureKeys: featureKeys,
	}, nil
}

// Predict returns the predicted class index and confidence for the given feature map.
func (c *ONNXClassifier) Predict(features map[string]float64) (int, float64, error) {
	if len(features) == 0 {
		return 0, 0, fmt.Errorf("empty features")
	}

	x := c.extractFeatures(features)

	numClasses := len(c.Labels)
	numFeatures := len(c.featureKeys)
	if !c.isValidConfig(numClasses, numFeatures) {
		return 0, 0, fmt.Errorf("unsupported onnx model configuration")
	}

	scores := c.computeScores(x, numClasses, numFeatures)
	best := c.findBestClass(scores)
	confidence := c.sigmoidConfidence(scores[best])

	return c.Labels[best], confidence, nil
}

func (c *ONNXClassifier) extractFeatures(features map[string]float64) []float64 {
	x := make([]float64, len(c.featureKeys))
	for i, key := range c.featureKeys {
		x[i] = features[key]
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

func (c *ONNXClassifier) sigmoidConfidence(maxScore float64) float64 {
	confidence := 1.0 / (1.0 + float64(-maxScore))
	if confidence > 1 {
		confidence = 1
	}
	if confidence < 0 {
		confidence = 0
	}
	return confidence
}

// FeatureKeys returns the expected feature order.
func (c *ONNXClassifier) FeatureKeys() []string {
	return c.featureKeys
}

// DefaultONNXWeightsPath returns the expected path for exported weights.
func DefaultONNXWeightsPath() string {
	// Assuming binary runs from repo root or cmd/classifier
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) == "cmd" || filepath.Base(cwd) == "classifier" {
		return filepath.Join("..", "..", "models", "classifier_weights.json")
	}
	return filepath.Join("models", "classifier_weights.json")
}
