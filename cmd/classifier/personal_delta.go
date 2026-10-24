package main

import (
	"errors"
	"math"
	"sync"
)

// DeltaUpdateRequest содержит обратную связь пользователя для обновления персональной дельты.
type DeltaUpdateRequest struct {
	UserID       string    `json:"user_id"`
	ClassIndex   int       `json:"class_index"`
	TargetValue  float64   `json:"target_value"`
	Features     []float64 `json:"features"`
	LearningRate float64   `json:"learning_rate"`
}

// DeltaUpdateResponse ответ после обновления дельты.
type DeltaUpdateResponse struct {
	Status     string  `json:"status"`
	UserID     string  `json:"user_id"`
	ClassIndex int     `json:"class_index"`
	DeltaNorm  float64 `json:"delta_norm"`
	Applied    bool    `json:"applied"`
}

// DeltaStore интерфейс для хранения персональных дельт пользователей.
type DeltaStore interface {
	GetDelta(userID string) ([]float64, bool)
	SetDelta(userID string, delta []float64)
	ResetDelta(userID string)
	DeltaSize() int
}

// InMemoryDeltaStore реализация DeltaStore в памяти.
type InMemoryDeltaStore struct {
	mu     sync.RWMutex
	deltas map[string][]float64
	size   int
}

// NewInMemoryDeltaStore создаёт новый in-memory хранилище дельт.
func NewInMemoryDeltaStore(deltaSize int) *InMemoryDeltaStore {
	return &InMemoryDeltaStore{
		deltas: make(map[string][]float64),
		size:   deltaSize,
	}
}

// GetDelta возвращает дельту пользователя или нулевой вектор если не существует.
func (s *InMemoryDeltaStore) GetDelta(userID string) ([]float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	delta, ok := s.deltas[userID]
	if !ok {
		zero := make([]float64, s.size)
		return zero, false
	}
	deltaCopy := make([]float64, len(delta))
	copy(deltaCopy, delta)
	return deltaCopy, true
}

// SetDelta сохраняет дельту пользователя.
func (s *InMemoryDeltaStore) SetDelta(userID string, delta []float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deltaCopy := make([]float64, len(delta))
	copy(deltaCopy, delta)
	s.deltas[userID] = deltaCopy
}

// ResetDelta сбрасывает дельту пользователя к нулям.
func (s *InMemoryDeltaStore) ResetDelta(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	zero := make([]float64, s.size)
	s.deltas[userID] = zero
}

// DeltaSize возвращает размер вектора дельты.
func (s *InMemoryDeltaStore) DeltaSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}

// PersonalDeltaClassifier обёртка над ONNXClassifier с поддержкой персональной дельты.
// Инференс: score = (W + Δ)·X
type PersonalDeltaClassifier struct {
	classifier *ONNXClassifier
	store      DeltaStore
}

// NewPersonalDeltaClassifier создаёт новый классификатор с поддержкой дельт.
func NewPersonalDeltaClassifier(classifier *ONNXClassifier, store DeltaStore) *PersonalDeltaClassifier {
	return &PersonalDeltaClassifier{
		classifier: classifier,
		store:      store,
	}
}

// PredictWithDelta выполняет инференс с учётом персональной дельты пользователя.
func (p *PersonalDeltaClassifier) PredictWithDelta(features map[string]float64, userID string) (int, float64, error) {
	if p.classifier == nil {
		return 0, 0, errors.New("classifier not initialized")
	}

	x := p.classifier.extractFeatures(features)
	numClasses := len(p.classifier.Labels)
	numFeatures := len(p.classifier.featureKeys)

	if !p.classifier.isValidConfig(numClasses, numFeatures) {
		return 0, 0, errors.New("unsupported onnx model configuration")
	}

	delta, _ := p.store.GetDelta(userID)
	if len(delta) != numClasses*numFeatures {
		delta = make([]float64, numClasses*numFeatures)
	}

	scores := make([]float64, numClasses)
	for cls := 0; cls < numClasses; cls++ {
		offset := cls * numFeatures
		sum := 0.0
		for j := 0; j < numFeatures; j++ {
			w := p.classifier.Weights[offset+j]
			d := delta[offset+j]
			sum += (w + d) * x[j]
		}
		scores[cls] = sum
	}

	best := p.classifier.findBestClass(scores)
	confidence := p.classifier.sigmoidConfidence(scores[best])

	return p.classifier.Labels[best], confidence, nil
}

// UpdateDelta обновляет персональную дельту: Δ += lr * (target - prediction) * X
func (p *PersonalDeltaClassifier) UpdateDelta(userID string, classIndex int, targetValue float64, features []float64, learningRate float64) ([]float64, error) {
	if p.classifier == nil {
		return nil, errors.New("classifier not initialized")
	}

	numClasses := len(p.classifier.Labels)
	numFeatures := len(p.classifier.featureKeys)

	if classIndex < 0 || classIndex >= numClasses {
		return nil, errors.New("invalid class index")
	}
	if len(features) != numFeatures {
		return nil, errors.New("invalid features length")
	}
	if learningRate <= 0 {
		learningRate = 0.01
	}

	delta, _ := p.store.GetDelta(userID)
	if len(delta) != numClasses*numFeatures {
		delta = make([]float64, numClasses*numFeatures)
	}

	currentScore := 0.0
	offset := classIndex * numFeatures
	for j := 0; j < numFeatures; j++ {
		currentScore += p.classifier.Weights[offset+j] * features[j]
	}
	currentPrediction := p.classifier.sigmoidConfidence(currentScore)

	errorTerm := targetValue - currentPrediction
	for j := 0; j < numFeatures; j++ {
		delta[offset+j] += learningRate * errorTerm * features[j]
	}

	maxDelta := 5.0
	for j := 0; j < numFeatures; j++ {
		if delta[offset+j] > maxDelta {
			delta[offset+j] = maxDelta
		}
		if delta[offset+j] < -maxDelta {
			delta[offset+j] = -maxDelta
		}
	}

	p.store.SetDelta(userID, delta)

	return delta, nil
}

// DeltaNorm возвращает норму текущей дельты пользователя.
func (p *PersonalDeltaClassifier) DeltaNorm(userID string) float64 {
	delta, _ := p.store.GetDelta(userID)
	var norm float64
	for _, v := range delta {
		norm += v * v
	}
	return math.Sqrt(norm)
}

// ResetUserDelta сбрасывает дельту пользователя.
func (p *PersonalDeltaClassifier) ResetUserDelta(userID string) {
	p.store.ResetDelta(userID)
}

// DeltaStoreSize возвращает размер вектора дельты.
func (p *PersonalDeltaClassifier) DeltaStoreSize() int {
	return p.store.DeltaSize()
}

// classifyWithDelta используется в classifyHandler для инференса с дельтой.
func (s *classifierServer) classifyWithDelta(features map[string]float64, userID string) (int, float64, error) {
	if s.personalClassifier != nil {
		return s.personalClassifier.PredictWithDelta(features, userID)
	}
	if s.onnxClassifier != nil {
		return s.onnxClassifier.Predict(features)
	}
	return 0, 0, errors.New("no classifier available")
}
