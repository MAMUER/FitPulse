package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInMemoryDeltaStore(t *testing.T) {
	store := NewInMemoryDeltaStore(20)
	require.NotNil(t, store)
	assert.Equal(t, 20, store.DeltaSize())
}

func TestInMemoryDeltaStoreGetUnknownUser(t *testing.T) {
	store := NewInMemoryDeltaStore(5)
	delta, exists := store.GetDelta("unknown-user")
	require.False(t, exists)
	assert.Equal(t, 5, len(delta))
	for _, v := range delta {
		assert.Equal(t, 0.0, v)
	}
}

func TestInMemoryDeltaStoreSetAndGet(t *testing.T) {
	store := NewInMemoryDeltaStore(3)
	original := []float64{0.1, -0.2, 0.3}
	store.SetDelta("user-1", original)

	delta, exists := store.GetDelta("user-1")
	require.True(t, exists)
	assert.Equal(t, 3, len(delta))
	assert.Equal(t, original, delta)

	// Модификация возвращённого среза не должна влиять на оригинал
	delta[0] = 999.0
	delta2, _ := store.GetDelta("user-1")
	assert.Equal(t, 0.1, delta2[0])
}

func TestInMemoryDeltaStoreReset(t *testing.T) {
	store := NewInMemoryDeltaStore(3)
	store.SetDelta("user-1", []float64{1.0, 2.0, 3.0})
	store.ResetDelta("user-1")

	delta, _ := store.GetDelta("user-1")
	for _, v := range delta {
		assert.Equal(t, 0.0, v)
	}
}

func TestPersonalDeltaClassifierPredictWithZeroDelta(t *testing.T) {
	classifier := &ONNXClassifier{
		Labels:      []int{0, 1},
		Weights:     []float64{1.0, 2.0, 3.0, 4.0},
		MultiClass:  1,
		featureKeys: []string{"a", "b"},
	}
	store := NewInMemoryDeltaStore(4) // 2 classes × 2 features
	pdc := NewPersonalDeltaClassifier(classifier, store)

	features := map[string]float64{"a": 1.0, "b": 2.0}
	class, conf, err := pdc.PredictWithDelta(features, "user-1")
	require.NoError(t, err)
	assert.Equal(t, 1, class)
	assert.Greater(t, conf, 0.99) // sigmoid(11) ≈ 0.99998
}

func TestPersonalDeltaClassifierPredictWithNonZeroDelta(t *testing.T) {
	classifier := &ONNXClassifier{
		Labels:      []int{0, 1},
		Weights:     []float64{1.0, 2.0, 3.0, 4.0},
		MultiClass:  1,
		featureKeys: []string{"a", "b"},
	}
	store := NewInMemoryDeltaStore(4)
	pdc := NewPersonalDeltaClassifier(classifier, store)

	features := map[string]float64{"a": 1.0, "b": 2.0}

	// С нулевой дельтой
	class1, conf1, err := pdc.PredictWithDelta(features, "user-1")
	require.NoError(t, err)

	// Добавляем большую дельту для класса 0, чтобы он стал выигрышным
	// Дельта для класса 0: offset=0, добавим +100 к признаку "a"
	delta := make([]float64, 4)
	delta[0] = 100.0 // class 0, feature "a"
	store.SetDelta("user-1", delta)

	class2, conf2, err := pdc.PredictWithDelta(features, "user-1")
	require.NoError(t, err)

	// Класс 0 теперь должен выиграть благодаря большой дельте
	assert.NotEqual(t, class1, class2)
	assert.Equal(t, 0, class2)
	assert.Greater(t, conf2, conf1)
}

func TestPersonalDeltaClassifierUpdateDelta(t *testing.T) {
	classifier := &ONNXClassifier{
		Labels:      []int{0, 1},
		Weights:     []float64{1.0, 2.0, 3.0, 4.0},
		MultiClass:  1,
		featureKeys: []string{"a", "b"},
	}
	store := NewInMemoryDeltaStore(4)
	pdc := NewPersonalDeltaClassifier(classifier, store)

	features := []float64{1.0, 2.0}

	// До обновления дельта нулевая
	deltaBefore, _ := store.GetDelta("user-1")
	for _, v := range deltaBefore {
		assert.Equal(t, 0.0, v)
	}

	// Обновляем дельту для класса 0 с target=1.0 (нравится)
	updatedDelta, err := pdc.UpdateDelta("user-1", 0, 1.0, features, 0.1)
	require.NoError(t, err)
	require.NotNil(t, updatedDelta)

	// Дельта для класса 0 (offset 0-1) должна измениться
	assert.NotEqual(t, 0.0, updatedDelta[0])
	assert.NotEqual(t, 0.0, updatedDelta[1])

	// Дельта для класса 1 (offset 2-3) должна остаться нулевой
	assert.Equal(t, 0.0, updatedDelta[2])
	assert.Equal(t, 0.0, updatedDelta[3])
}

func TestPersonalDeltaClassifierUpdateDeltaInvalidClassIndex(t *testing.T) {
	classifier := &ONNXClassifier{
		Labels:      []int{0, 1},
		Weights:     []float64{1.0, 2.0, 3.0, 4.0},
		MultiClass:  1,
		featureKeys: []string{"a", "b"},
	}
	store := NewInMemoryDeltaStore(4)
	pdc := NewPersonalDeltaClassifier(classifier, store)

	_, err := pdc.UpdateDelta("user-1", 5, 1.0, []float64{1.0, 2.0}, 0.1)
	require.Error(t, err)
}

func TestPersonalDeltaClassifierUpdateDeltaInvalidFeatures(t *testing.T) {
	classifier := &ONNXClassifier{
		Labels:      []int{0, 1},
		Weights:     []float64{1.0, 2.0, 3.0, 4.0},
		MultiClass:  1,
		featureKeys: []string{"a", "b"},
	}
	store := NewInMemoryDeltaStore(4)
	pdc := NewPersonalDeltaClassifier(classifier, store)

	_, err := pdc.UpdateDelta("user-1", 0, 1.0, []float64{1.0}, 0.1)
	require.Error(t, err)
}

func TestPersonalDeltaClassifierDeltaNorm(t *testing.T) {
	classifier := &ONNXClassifier{
		Labels:      []int{0, 1},
		Weights:     []float64{1.0, 2.0, 3.0, 4.0},
		MultiClass:  1,
		featureKeys: []string{"a", "b"},
	}
	store := NewInMemoryDeltaStore(4)
	pdc := NewPersonalDeltaClassifier(classifier, store)

	// Без дельты норма = 0
	assert.Equal(t, 0.0, pdc.DeltaNorm("user-1"))

	// Устанавливаем дельту
	store.SetDelta("user-1", []float64{3.0, 4.0, 0.0, 0.0})
	assert.InDelta(t, 5.0, pdc.DeltaNorm("user-1"), 0.001) // sqrt(3^2 + 4^2) = 5
}

func TestPersonalDeltaClassifierResetUserDelta(t *testing.T) {
	classifier := &ONNXClassifier{
		Labels:      []int{0, 1},
		Weights:     []float64{1.0, 2.0, 3.0, 4.0},
		MultiClass:  1,
		featureKeys: []string{"a", "b"},
	}
	store := NewInMemoryDeltaStore(4)
	pdc := NewPersonalDeltaClassifier(classifier, store)

	store.SetDelta("user-1", []float64{1.0, 2.0, 3.0, 4.0})
	assert.NotEqual(t, 0.0, pdc.DeltaNorm("user-1"))

	pdc.ResetUserDelta("user-1")
	assert.Equal(t, 0.0, pdc.DeltaNorm("user-1"))
}

func TestDeltaNormHelper(t *testing.T) {
	assert.Equal(t, 0.0, deltaNorm(nil))
	assert.InDelta(t, 5.0, deltaNorm([]float64{3.0, 4.0}), 0.001)
	assert.InDelta(t, 0.0, deltaNorm([]float64{0.0, 0.0}), 0.001)
}
