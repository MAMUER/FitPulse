package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

// scoreModel — общий контракт для линейной (JSON-веса) и древовидной
// (HistGradientBoosting) моделей. Позволяет PERSONAL-дельтам работать
// одинаково: дельта — линейная поправка к итоговым оценкам классов.
type scoreModel interface {
	NumClasses() int
	NumFeatures() int
	FeatureKeys() []string
	ExtractFeatures(features map[string]float64) []float64
	RawScores(x []float64) []float64
	Confidence(scores []float64) float64
	LabelsList() []int
}

// Tree — один узел дерева HistGradientBoosting.
type Tree struct {
	Feature     []int
	Threshold   []float64
	Left        []int
	Right       []int
	MissingLeft []bool
	Value       []float64
	Leaf        []bool
}

// Eval обходит дерево и возвращает значение листа.
func (t *Tree) Eval(x []float64) float64 {
	index := 0
	for index >= 0 && index < len(t.Leaf) && !t.Leaf[index] {
		feature := t.Feature[index]
		if feature < 0 || feature >= len(x) || math.IsNaN(x[feature]) {
			if t.MissingLeft[index] {
				index = t.Left[index]
			} else {
				index = t.Right[index]
			}
			continue
		}
		if x[feature] <= t.Threshold[index] {
			index = t.Left[index]
		} else {
			index = t.Right[index]
		}
	}
	if index < 0 || index >= len(t.Value) {
		return 0
	}
	return t.Value[index]
}

// TreeModel — HistGradientBoosting, экспортированный в JSON (histgb-v1).
type TreeModel struct {
	numClasses   int
	learningRate float64
	baseline     []float64
	featureKeys  []string
	featureMin   []float64
	featureMax   []float64
	labels       []int
	classes      []string
	trees        []Tree
}

type treeFile struct {
	Format       string    `json:"format"`
	NumClasses   int       `json:"n_classes"`
	LearningRate float64   `json:"learning_rate"`
	Baseline     []float64 `json:"baseline"`
	FeatureKeys  []string  `json:"feature_keys"`
	FeatureMin   []float64 `json:"feature_min"`
	FeatureMax   []float64 `json:"feature_max"`
	Classes      []string  `json:"classes"`
	Trees        []struct {
		Feature     []int     `json:"feature"`
		Threshold   []float64 `json:"threshold"`
		Left        []int     `json:"left"`
		Right       []int     `json:"right"`
		MissingLeft []bool    `json:"missing_left"`
		Value       []float64 `json:"value"`
		Leaf        []bool    `json:"leaf"`
	} `json:"trees"`
}

// NewTreeModel загружает древовидную модель.
func NewTreeModel(path string) (*TreeModel, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return nil, fmt.Errorf("empty tree model path")
	}
	data, err := os.ReadFile(clean)
	if err != nil {
		return nil, fmt.Errorf("read tree model: %w", err)
	}

	var file treeFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse tree model: %w", err)
	}
	if file.Format != "histgb-v1" {
		return nil, fmt.Errorf("unsupported tree model format %q", file.Format)
	}
	if file.NumClasses <= 0 || len(file.Trees) == 0 {
		return nil, fmt.Errorf("invalid tree model")
	}

	model := &TreeModel{
		numClasses:   file.NumClasses,
		learningRate: file.LearningRate,
		baseline:     file.Baseline,
		featureKeys:  file.FeatureKeys,
		featureMin:   file.FeatureMin,
		featureMax:   file.FeatureMax,
		classes:      file.Classes,
		trees:        make([]Tree, 0, len(file.Trees)),
	}
	for i := range file.Trees {
		t := file.Trees[i]
		model.trees = append(model.trees, Tree{
			Feature:     t.Feature,
			Threshold:   t.Threshold,
			Left:        t.Left,
			Right:       t.Right,
			MissingLeft: t.MissingLeft,
			Value:       t.Value,
			Leaf:        t.Leaf,
		})
	}
	model.labels = make([]int, model.numClasses)
	for i := range model.labels {
		model.labels[i] = i
	}
	return model, nil
}

func (m *TreeModel) NumClasses() int   { return m.numClasses }
func (m *TreeModel) NumFeatures() int  { return len(m.featureKeys) }
func (m *TreeModel) LabelsList() []int { return m.labels }

func (m *TreeModel) FeatureKeys() []string {
	keys := make([]string, len(m.featureKeys))
	copy(keys, m.featureKeys)
	return keys
}

// Predict возвращает класс и вероятность (softmax по сырым оценкам).
func (m *TreeModel) Predict(features map[string]float64) (int, float64, error) {
	x := m.ExtractFeatures(features)
	scores := m.RawScores(x)
	best := 0
	for i := 1; i < len(scores); i++ {
		if scores[i] > scores[best] {
			best = i
		}
	}
	if best >= len(m.labels) {
		return 0, 0, fmt.Errorf("invalid prediction index")
	}
	return m.labels[best], m.Confidence(scores), nil
}

// ExtractFeatures нормализует сырые значения; отсутствующий признак → NaN
// (деревья HistGradientBoosting умеют работать с пропусками).
func (m *TreeModel) ExtractFeatures(features map[string]float64) []float64 {
	x := make([]float64, len(m.featureKeys))
	for i, key := range m.featureKeys {
		value, ok := features[key]
		if !ok {
			x[i] = math.NaN()
			continue
		}
		if i < len(m.featureMin) && i < len(m.featureMax) {
			span := m.featureMax[i] - m.featureMin[i]
			if span > 0 {
				value = (value - m.featureMin[i]) / span
			}
		}
		x[i] = value
	}
	return x
}

// RawScores: baseline + сумма значений деревьев класса.
// ВАЖНО: значения листьев sklearn УЖЕ содержат learning_rate (он применяется
// при обучении), поэтому повторно умножать нельзя — иначе argmax ломается на
// пограничных примерах.
func (m *TreeModel) RawScores(x []float64) []float64 {
	scores := make([]float64, m.numClasses)
	copy(scores, m.baseline)
	for t := range m.trees {
		class := t % m.numClasses
		scores[class] += m.trees[t].Eval(x)
	}
	return scores
}

// Confidence — softmax по «сырым» оценкам (как predict_proba у sklearn).
func (m *TreeModel) Confidence(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}
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
	best := 0
	for i := 1; i < len(scores); i++ {
		if scores[i] > scores[best] {
			best = i
		}
	}
	if total <= 0 {
		return 0
	}
	return math.Max(0, math.Min(1, exps[best]/total))
}
