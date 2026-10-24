package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockDeltaStore struct {
	deltas map[string][]float64
	size   int
}

func newMockDeltaStore(size int) *mockDeltaStore {
	return &mockDeltaStore{
		deltas: make(map[string][]float64),
		size:   size,
	}
}

func (s *mockDeltaStore) GetDelta(userID string) ([]float64, bool) {
	delta, ok := s.deltas[userID]
	if !ok {
		zero := make([]float64, s.size)
		return zero, false
	}
	cp := make([]float64, len(delta))
	copy(cp, delta)
	return cp, true
}

func (s *mockDeltaStore) SetDelta(userID string, delta []float64) {
	cp := make([]float64, len(delta))
	copy(cp, delta)
	s.deltas[userID] = cp
}

func (s *mockDeltaStore) ResetDelta(userID string) {
	s.deltas[userID] = make([]float64, s.size)
}

func (s *mockDeltaStore) DeltaSize() int {
	return s.size
}

func TestInMemoryDeltaStore(t *testing.T) {
	store := newMockDeltaStore(4)

	// GetDelta on non-existent user returns zero vector
	delta, ok := store.GetDelta("user-1")
	assert.False(t, ok)
	assert.Len(t, delta, 4)
	for _, v := range delta {
		assert.Equal(t, 0.0, v)
	}

	// SetDelta and GetDelta
	store.SetDelta("user-1", []float64{1.0, 2.0, 3.0, 4.0})
	delta, ok = store.GetDelta("user-1")
	assert.True(t, ok)
	assert.Equal(t, []float64{1.0, 2.0, 3.0, 4.0}, delta)

	// ResetDelta
	store.ResetDelta("user-1")
	delta, ok = store.GetDelta("user-1")
	assert.True(t, ok)
	for _, v := range delta {
		assert.Equal(t, 0.0, v)
	}

	// DeltaSize
	assert.Equal(t, 4, store.DeltaSize())
}

func TestRealInMemoryDeltaStore(t *testing.T) {
	store := NewInMemoryDeltaStore(4)

	// GetDelta on non-existent user returns zero vector
	delta, ok := store.GetDelta("user-1")
	assert.False(t, ok)
	assert.Len(t, delta, 4)
	for _, v := range delta {
		assert.Equal(t, 0.0, v)
	}

	// DeltaSize
	assert.Equal(t, 4, store.DeltaSize())

	// SetDelta and GetDelta
	store.SetDelta("user-1", []float64{1.0, 2.0, 3.0, 4.0})
	delta, ok = store.GetDelta("user-1")
	assert.True(t, ok)
	assert.Equal(t, []float64{1.0, 2.0, 3.0, 4.0}, delta)

	// GetDelta returns a copy (modifying returned slice doesn't affect store)
	delta[0] = 99.0
	delta2, _ := store.GetDelta("user-1")
	assert.Equal(t, 1.0, delta2[0])

	// ResetDelta
	store.ResetDelta("user-1")
	delta, ok = store.GetDelta("user-1")
	assert.True(t, ok)
	for _, v := range delta {
		assert.Equal(t, 0.0, v)
	}
}

func TestRealInMemoryDeltaStore_SetDeltaWithConcurrentAccess(t *testing.T) {
	store := NewInMemoryDeltaStore(2)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			store.SetDelta("user", []float64{float64(idx), float64(idx)})
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	delta, _ := store.GetDelta("user")
	assert.Len(t, delta, 2)
}

func TestPersonalDeltaClassifier_NilClassifier(t *testing.T) {
	store := newMockDeltaStore(4)
	pc := &PersonalDeltaClassifier{
		classifier: nil,
		store:      store,
	}

	_, _, err := pc.PredictWithDelta(map[string]float64{"x": 1.0}, "user-1")
	assert.Error(t, err)
	assert.Equal(t, "classifier not initialized", err.Error())

	_, err = pc.UpdateDelta("user-1", 0, 1.0, []float64{1.0}, 0.01)
	assert.Error(t, err)
	assert.Equal(t, "classifier not initialized", err.Error())
}

func TestPersonalDeltaClassifier_DeltaNorm(t *testing.T) {
	store := newMockDeltaStore(4)
	store.SetDelta("user-1", []float64{3.0, 4.0, 0.0, 0.0})
	pc := &PersonalDeltaClassifier{
		classifier: nil,
		store:      store,
	}

	norm := pc.DeltaNorm("user-1")
	assert.Equal(t, 5.0, norm) // sqrt(3^2 + 4^2) = 5
}

func TestPersonalDeltaClassifier_ResetUserDelta(t *testing.T) {
	store := newMockDeltaStore(4)
	store.SetDelta("user-1", []float64{1.0, 2.0, 3.0, 4.0})
	pc := &PersonalDeltaClassifier{
		classifier: nil,
		store:      store,
	}

	pc.ResetUserDelta("user-1")
	delta, _ := store.GetDelta("user-1")
	for _, v := range delta {
		assert.Equal(t, 0.0, v)
	}
}

func TestPersonalDeltaClassifier_DeltaStoreSize(t *testing.T) {
	store := newMockDeltaStore(8)
	pc := &PersonalDeltaClassifier{
		classifier: nil,
		store:      store,
	}

	assert.Equal(t, 8, pc.DeltaStoreSize())
}

func TestDeltaUpdateRequest_Fields(t *testing.T) {
	req := DeltaUpdateRequest{
		UserID:       "user-1",
		ClassIndex:   0,
		TargetValue:  1.0,
		Features:     []float64{0.5, 0.3},
		LearningRate: 0.01,
	}
	assert.Equal(t, "user-1", req.UserID)
	assert.Equal(t, 0, req.ClassIndex)
	assert.Equal(t, 1.0, req.TargetValue)
	assert.Equal(t, []float64{0.5, 0.3}, req.Features)
	assert.Equal(t, 0.01, req.LearningRate)
}

func TestDeltaUpdateResponse_Fields(t *testing.T) {
	resp := DeltaUpdateResponse{
		Status:     "ok",
		UserID:     "user-1",
		ClassIndex: 1,
		DeltaNorm:  0.5,
		Applied:    true,
	}
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "user-1", resp.UserID)
	assert.Equal(t, 1, resp.ClassIndex)
	assert.Equal(t, 0.5, resp.DeltaNorm)
	assert.True(t, resp.Applied)
}

func TestErrors(t *testing.T) {
	assert.Error(t, errors.New("classifier not initialized"))
	assert.Error(t, errors.New("invalid class index"))
	assert.Error(t, errors.New("invalid features length"))
	assert.Error(t, errors.New("unsupported onnx model configuration"))
}

func newTestONNXClassifier() *ONNXClassifier {
	numClasses := 2
	numFeatures := 4
	weights := make([]float64, numClasses*numFeatures)
	for i := range weights {
		weights[i] = 0.5
	}
	return &ONNXClassifier{
		Labels:      []int{0, 1}[:numClasses],
		Weights:     weights,
		MultiClass:  1,
		featureKeys: []string{"heart_rate", "hrv", "spo2", "temperature"},
	}
}

func TestNewPersonalDeltaClassifier(t *testing.T) {
	classifier := newTestONNXClassifier()
	store := newMockDeltaStore(8)
	pc := NewPersonalDeltaClassifier(classifier, store)
	assert.NotNil(t, pc)
	assert.Equal(t, classifier, pc.classifier)
	assert.Equal(t, store, pc.store)
}

func TestPersonalDeltaClassifier_PredictWithDelta(t *testing.T) {
	classifier := newTestONNXClassifier()
	store := newMockDeltaStore(8)
	pc := NewPersonalDeltaClassifier(classifier, store)

	features := map[string]float64{
		"heart_rate":  140.0,
		"hrv":         50.0,
		"spo2":        98.0,
		"temperature": 37.0,
	}
	label, confidence, err := pc.PredictWithDelta(features, "user-1")
	assert.NoError(t, err)
	assert.Contains(t, []int{0, 1}, label)
	assert.GreaterOrEqual(t, confidence, 0.0)
	assert.LessOrEqual(t, confidence, 1.0)
}

func TestPersonalDeltaClassifier_PredictWithDelta_NilClassifier(t *testing.T) {
	store := newMockDeltaStore(8)
	pc := &PersonalDeltaClassifier{
		classifier: nil,
		store:      store,
	}
	_, _, err := pc.PredictWithDelta(map[string]float64{"heart_rate": 140.0}, "user-1")
	assert.Error(t, err)
	assert.Equal(t, "classifier not initialized", err.Error())
}

func TestPersonalDeltaClassifier_PredictWithDelta_InvalidConfig(t *testing.T) {
	badClassifier := &ONNXClassifier{
		Labels:      []int{0},
		Weights:     []float64{0.5},
		MultiClass:  0,
		featureKeys: []string{"heart_rate"},
	}
	store := newMockDeltaStore(1)
	pc := NewPersonalDeltaClassifier(badClassifier, store)
	_, _, err := pc.PredictWithDelta(map[string]float64{"heart_rate": 140.0}, "user-1")
	assert.Error(t, err)
	assert.Equal(t, "unsupported onnx model configuration", err.Error())
}

func TestPersonalDeltaClassifier_UpdateDelta(t *testing.T) {
	classifier := newTestONNXClassifier()
	store := newMockDeltaStore(8)
	pc := NewPersonalDeltaClassifier(classifier, store)

	delta, err := pc.UpdateDelta("user-1", 0, 1.0, []float64{1.0, 2.0, 3.0, 4.0}, 0.01)
	assert.NoError(t, err)
	assert.Len(t, delta, 8)
}

func TestPersonalDeltaClassifier_UpdateDelta_InvalidClassIndex(t *testing.T) {
	classifier := newTestONNXClassifier()
	store := newMockDeltaStore(8)
	pc := NewPersonalDeltaClassifier(classifier, store)

	_, err := pc.UpdateDelta("user-1", 5, 1.0, []float64{1.0, 2.0, 3.0, 4.0}, 0.01)
	assert.Error(t, err)
	assert.Equal(t, "invalid class index", err.Error())
}

func TestPersonalDeltaClassifier_UpdateDelta_InvalidFeaturesLength(t *testing.T) {
	classifier := newTestONNXClassifier()
	store := newMockDeltaStore(8)
	pc := NewPersonalDeltaClassifier(classifier, store)

	_, err := pc.UpdateDelta("user-1", 0, 1.0, []float64{1.0, 2.0, 3.0}, 0.01)
	assert.Error(t, err)
	assert.Equal(t, "invalid features length", err.Error())
}

func TestPersonalDeltaClassifier_UpdateDelta_LearningRateDefault(t *testing.T) {
	classifier := newTestONNXClassifier()
	store := newMockDeltaStore(8)
	pc := NewPersonalDeltaClassifier(classifier, store)

	delta, err := pc.UpdateDelta("user-1", 0, 1.0, []float64{1.0, 2.0, 3.0, 4.0}, 0)
	assert.NoError(t, err)
	assert.Len(t, delta, 8)
}

func TestPersonalDeltaClassifier_UpdateDelta_ClampsDelta(t *testing.T) {
	classifier := newTestONNXClassifier()
	store := newMockDeltaStore(8)
	pc := NewPersonalDeltaClassifier(classifier, store)

	// High learning rate should produce delta values that get clamped to [-5, 5]
	_, err := pc.UpdateDelta("user-1", 0, 1.0, []float64{100.0, 100.0, 100.0, 100.0}, 10.0)
	assert.NoError(t, err)

	delta, _ := store.GetDelta("user-1")
	for j := 0; j < 4; j++ {
		assert.LessOrEqual(t, delta[j], 5.0)
		assert.GreaterOrEqual(t, delta[j], -5.0)
	}
}

func TestClassifyWithDelta_PersonalClassifier(t *testing.T) {
	classifier := newTestONNXClassifier()
	store := newMockDeltaStore(8)
	pc := NewPersonalDeltaClassifier(classifier, store)

	s := &classifierServer{
		log:            nil,
		personalClassifier: pc,
	}

	label, confidence, err := s.classifyWithDelta(map[string]float64{
		"heart_rate":  140.0,
		"hrv":         50.0,
		"spo2":        98.0,
		"temperature": 37.0,
	}, "user-1")
	assert.NoError(t, err)
	assert.Contains(t, []int{0, 1}, label)
	assert.GreaterOrEqual(t, confidence, 0.0)
	assert.LessOrEqual(t, confidence, 1.0)
}

func TestClassifyWithDelta_ONNXClassifierFallback(t *testing.T) {
	classifier := newTestONNXClassifier()
	s := &classifierServer{
		log:            nil,
		onnxClassifier: classifier,
	}

	label, confidence, err := s.classifyWithDelta(map[string]float64{
		"heart_rate":  140.0,
		"hrv":         50.0,
		"spo2":        98.0,
		"temperature": 37.0,
	}, "user-1")
	assert.NoError(t, err)
	assert.Contains(t, []int{0, 1}, label)
	assert.GreaterOrEqual(t, confidence, 0.0)
	assert.LessOrEqual(t, confidence, 1.0)
}

func TestClassifyWithDelta_NoClassifier(t *testing.T) {
	s := &classifierServer{log: nil}
	_, _, err := s.classifyWithDelta(map[string]float64{"heart_rate": 140.0}, "user-1")
	assert.Error(t, err)
	assert.Equal(t, "no classifier available", err.Error())
}
