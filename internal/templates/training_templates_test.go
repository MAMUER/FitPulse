package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTemplate_Existing(t *testing.T) {
	tpl, ok := GetTemplate("recovery")
	assert.True(t, ok)
	assert.Equal(t, "Восстановление", tpl.NameRu)
	assert.Equal(t, [2]int{20, 40}, tpl.DurationRange)
	assert.Equal(t, [2]float64{0.3, 0.5}, tpl.IntensityRange)
	assert.Equal(t, 0.7, tpl.RestRatio)
	assert.Equal(t, []string{"лёгкая разминка", "растяжка", "дыхательные упражнения", "йога"}, tpl.Exercises)
}

func TestGetTemplate_NonExisting(t *testing.T) {
	tpl, ok := GetTemplate("unknown_class")
	assert.False(t, ok)
	assert.Equal(t, TrainingTemplate{}, tpl)
}

func TestTrainingTemplates_AllPresent(t *testing.T) {
	expectedKeys := []string{
		"recovery",
		"endurance_basic",
		"endurance_threshold",
		"power_hiit",
		"overtraining",
		"illness",
	}
	for _, key := range expectedKeys {
		_, ok := TrainingTemplates[key]
		assert.True(t, ok, "template %s should exist", key)
	}
}

func TestTrainingTemplates_Values(t *testing.T) {
	tests := []struct {
		key            string
		nameRu         string
		minDuration    int
		maxDuration    int
		minIntensity   float64
		maxIntensity   float64
		restRatio      float64
		exerciseCount  int
	}{
		{"recovery", "Восстановление", 20, 40, 0.3, 0.5, 0.7, 4},
		{"endurance_basic", "Базовая выносливость", 45, 90, 0.5, 0.7, 0.4, 4},
		{"endurance_threshold", "Пороговая выносливость", 30, 60, 0.7, 0.85, 0.3, 4},
		{"power_hiit", "Силовая/HIIT", 20, 45, 0.85, 1.0, 0.5, 4},
		{"overtraining", "Перетренированность", 0, 20, 0.0, 0.3, 0.8, 4},
		{"illness", "Заболевание", 0, 0, 0.0, 0.0, 1.0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			tpl, ok := TrainingTemplates[tt.key]
			assert.True(t, ok)
			assert.Equal(t, tt.nameRu, tpl.NameRu)
			assert.Equal(t, tt.minDuration, tpl.DurationRange[0])
			assert.Equal(t, tt.maxDuration, tpl.DurationRange[1])
			assert.Equal(t, tt.minIntensity, tpl.IntensityRange[0])
			assert.Equal(t, tt.maxIntensity, tpl.IntensityRange[1])
			assert.Equal(t, tt.restRatio, tpl.RestRatio)
			assert.Equal(t, tt.exerciseCount, len(tpl.Exercises))
		})
	}
}
