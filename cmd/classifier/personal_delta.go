package main

import (
	"errors"
	"math"
	"sync"
)

// DeltaUpdateRequest содержит обратную связь пользователя для обновления персональной дельты.
type DeltaUpdateRequest struct {
	UserID       string             `json:"user_id"`
	ClassIndex   int                `json:"class_index"`
	TargetValue  float64            `json:"target_value"`
	Features     map[string]float64 `json:"features"`
	LearningRate float64            `json:"learning_rate"`
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

// ResetDelta сбрасывает дельту пользователя.
func (s *InMemoryDeltaStore) ResetDelta(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	zero := make([]float64, s.size)
	s.deltas[userID] = zero
}

// DeltaSize возвращает размер дельты.
func (s *InMemoryDeltaStore) DeltaSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}

// GetAllDeltas возвращает копию всех дельт.
func (s *InMemoryDeltaStore) GetAllDeltas() map[string][]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string][]float64, len(s.deltas))
	for k, v := range s.deltas {
		cp := make([]float64, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

// PersonalDeltaClassifier — обёртка над моделью, добавляющая персональную
// дельту Δ_u к итоговым оценкам классов. Работает и с линейной, и с
// древовидной моделью (scoreModel).
type PersonalDeltaClassifier struct {
	classifier scoreModel
	store      DeltaStore
}

// NewPersonalDeltaClassifier создаёт классификатор с персональными дельтами.
func NewPersonalDeltaClassifier(classifier scoreModel, store DeltaStore) *PersonalDeltaClassifier {
	return &PersonalDeltaClassifier{
		classifier: classifier,
		store:      store,
	}
}

func (p *PersonalDeltaClassifier) PredictWithDelta(features map[string]float64, userID string) (int, float64, error) {
	if p.classifier == nil {
		return 0, 0, errors.New("classifier not initialized")
	}

	numClasses := p.classifier.NumClasses()
	numFeatures := p.classifier.NumFeatures()

	x := p.classifier.ExtractFeatures(features)
	base := p.classifier.RawScores(x)
	if len(base) != numClasses {
		return 0, 0, errors.New("unsupported model configuration")
	}

	delta, _ := p.store.GetDelta(userID)
	if len(delta) != numClasses*numFeatures {
		delta = make([]float64, numClasses*numFeatures)
	}

	scores := make([]float64, numClasses)
	copy(scores, base)
	for class := 0; class < numClasses; class++ {
		offset := class * numFeatures
		for j := 0; j < numFeatures && j < len(x); j++ {
			scores[class] += delta[offset+j] * x[j]
		}
	}

	best := 0
	for i := 1; i < len(scores); i++ {
		if scores[i] > scores[best] {
			best = i
		}
	}

	labels := p.classifier.LabelsList()
	if best >= len(labels) {
		return 0, 0, errors.New("invalid prediction index")
	}
	return labels[best], p.classifier.Confidence(scores), nil
}

// UpdateDelta пересчитывает дельту по обратной связи пользователя.
func (p *PersonalDeltaClassifier) UpdateDelta(userID string, classIndex int, targetValue float64, features map[string]float64, learningRate float64) ([]float64, error) {
	if p.classifier == nil {
		return nil, errors.New("classifier not initialized")
	}

	numClasses := p.classifier.NumClasses()
	numFeatures := p.classifier.NumFeatures()

	if classIndex < 0 || classIndex >= numClasses {
		return nil, errors.New("invalid class index")
	}
	if err := validateFeatureKeys(p.classifier.FeatureKeys(), features); err != nil {
		return nil, err
	}
	if learningRate <= 0 {
		learningRate = 0.01
	}

	x := p.classifier.ExtractFeatures(features)
	if len(x) != numFeatures {
		return nil, errors.New("invalid features length")
	}

	delta := p.deltaForUser(userID, numClasses*numFeatures)
	offset := classIndex * numFeatures

	base := p.classifier.RawScores(x)
	if len(base) != numClasses {
		return nil, errors.New("unsupported model configuration")
	}
	current := base[classIndex]
	for j := 0; j < numFeatures; j++ {
		current += delta[offset+j] * x[j]
	}

	errorTerm := targetValue - sigmoid(current)
	applyGradient(delta, offset, x, learningRate*errorTerm)
	clampDelta(delta, offset, numFeatures)

	p.store.SetDelta(userID, delta)
	return delta, nil
}

func sigmoid(v float64) float64 {
	return 1.0 / (1.0 + math.Exp(-v))
}

// validateFeatureKeys требует наличия всех признаков модели.
func validateFeatureKeys(keys []string, features map[string]float64) error {
	for _, key := range keys {
		if _, ok := features[key]; !ok {
			return errors.New("invalid features length")
		}
	}
	return nil
}

// deltaForUser возвращает дельту пользователя или нулевой вектор нужного размера.
func (p *PersonalDeltaClassifier) deltaForUser(userID string, size int) []float64 {
	delta, _ := p.store.GetDelta(userID)
	if len(delta) != size {
		delta = make([]float64, size)
	}
	return delta
}

// applyGradient обновляет дельту целевого класса.
func applyGradient(delta []float64, offset int, x []float64, step float64) {
	for j := range x {
		if offset+j >= len(delta) {
			break
		}
		delta[offset+j] += step * x[j]
	}
}

// clampDelta ограничивает дельту диапазоном [-5, 5].
func clampDelta(delta []float64, offset, numFeatures int) {
	const maxDelta = 5.0
	for j := 0; j < numFeatures; j++ {
		if offset+j >= len(delta) {
			break
		}
		if delta[offset+j] > maxDelta {
			delta[offset+j] = maxDelta
		}
		if delta[offset+j] < -maxDelta {
			delta[offset+j] = -maxDelta
		}
	}
}

func (p *PersonalDeltaClassifier) DeltaNorm(userID string) float64 {
	delta, _ := p.store.GetDelta(userID)
	var norm float64
	for _, v := range delta {
		norm += v * v
	}
	return math.Sqrt(norm)
}

func (p *PersonalDeltaClassifier) ResetUserDelta(userID string) {
	p.store.ResetDelta(userID)
}

func (p *PersonalDeltaClassifier) DeltaStoreSize() int {
	return p.store.DeltaSize()
}

func (s *classifierServer) classifyWithDelta(features map[string]float64, userID string) (int, float64, error) {
	if model := s.activeModel(); model != nil {
		if p := s.personalClassifier; p != nil {
			return p.PredictWithDelta(features, userID)
		}
		return predictWithModel(model, features)
	}
	return 0, 0, errors.New("no classifier available")
}

// predictWithModel — предсказание без персональной дельты.
func predictWithModel(model scoreModel, features map[string]float64) (int, float64, error) {
	if model == nil {
		return 0, 0, errors.New("classifier not initialized")
	}
	x := model.ExtractFeatures(features)
	scores := model.RawScores(x)
	if len(scores) != model.NumClasses() {
		return 0, 0, errors.New("unsupported model configuration")
	}
	best := 0
	for i := 1; i < len(scores); i++ {
		if scores[i] > scores[best] {
			best = i
		}
	}
	labels := model.LabelsList()
	if best >= len(labels) {
		return 0, 0, errors.New("invalid prediction index")
	}
	return labels[best], model.Confidence(scores), nil
}
