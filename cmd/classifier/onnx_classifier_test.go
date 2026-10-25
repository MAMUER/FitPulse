package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewONNXClassifier_InvalidPath(t *testing.T) {
	_, err := NewONNXClassifier("..\\etc\\passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path")
}

func TestNewONNXClassifier_ReadError(t *testing.T) {
	_, err := NewONNXClassifier(filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read onnx weights")
}

func TestNewONNXClassifier_ParseError(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(p, []byte("not-json"), 0o644))
	_, err := NewONNXClassifier(p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse onnx weights")
}

func TestNewONNXClassifier_Success(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "weights.json")
	// 2 classes, 4 features -> 8 weights
	require.NoError(t, os.WriteFile(p, []byte(`{"labels":[0,1],"weights":[0.1,0.2,0.3,0.4,-0.1,-0.2,-0.3,-0.4],"multi_class":1}`), 0o644))
	c, err := NewONNXClassifier(p)
	require.NoError(t, err)
	assert.Equal(t, []int{0, 1}, c.Labels)
	assert.Len(t, c.FeatureKeys(), 4)
}

func TestONNXClassifier_Predict(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "weights.json")
	require.NoError(t, os.WriteFile(p, []byte(`{"labels":[0,1],"weights":[0.1,0.2,0.3,0.4,-0.1,-0.2,-0.3,-0.4],"multi_class":1}`), 0o644))
	c, err := NewONNXClassifier(p)
	require.NoError(t, err)

	res, conf, err := c.Predict(map[string]float64{"heart_rate": 80, "hrv": 50, "spo2": 98, "temperature": 36.6})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, conf, 0.0)
	assert.LessOrEqual(t, conf, 1.0)
	assert.Contains(t, []int{0, 1}, res)
}

func TestONNXClassifier_Predict_Errors(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "weights.json")
	require.NoError(t, os.WriteFile(p, []byte(`{"labels":[0,1],"weights":[0.1,0.2,0.3,0.4,-0.1,-0.2,-0.3,-0.4],"multi_class":1}`), 0o644))
	c, _ := NewONNXClassifier(p)

	_, _, err := c.Predict(map[string]float64{})
	require.Error(t, err)

	c2 := &ONNXClassifier{Labels: []int{0}, Weights: []float64{1}, MultiClass: 1, featureKeys: []string{"a"}}
	_, _, err = c2.Predict(map[string]float64{"a": 1})
	require.Error(t, err)
}

func TestDefaultONNXWeightsPath(t *testing.T) {
	p := DefaultONNXWeightsPath()
	assert.True(t, strings.Contains(p, "classifier_weights.json"))
}
func TestONNXClassifier_findBestClass(t *testing.T) {
	c := &ONNXClassifier{}
	assert.Equal(t, 2, c.findBestClass([]float64{1.0, 2.0, 5.0, 0.5}))
}

func TestDefaultONNXWeightsPath_Cmd(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "cmd")
	require.NoError(t, os.Mkdir(cmdDir, 0o755))
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(cmdDir))
	defer func() { _ = os.Chdir(old) }()
	p := DefaultONNXWeightsPath()
	assert.Contains(t, p, "classifier_weights.json")
}
func TestDefaultONNXWeightsPath_Default(t *testing.T) {
	dir := t.TempDir()
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(old) }()
	p := DefaultONNXWeightsPath()
	assert.Contains(t, p, "classifier_weights.json")
}
