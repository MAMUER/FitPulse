package planner

import (
	"math"
	"testing"

	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/templates"
	"github.com/stretchr/testify/assert"
)

func TestGeneratePlanFromTemplate_ValidClassification(t *testing.T) {
	plan, weeks, err := GeneratePlanFromTemplate("recovery", &entity.User{}, nil, nil, PlanConstraints{
		DurationWeeks: 2,
		AvailableDays: []int{1, 3, 5},
	})
	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.NotNil(t, weeks)
	assert.Equal(t, "recovery", plan.Classification)
	assert.Equal(t, 2, plan.DurationWeeks)
	assert.Equal(t, []int{1, 3, 5}, plan.AvailableDays)
	assert.Equal(t, 2, len(weeks))
	assert.Equal(t, "Восстановление", plan.PlanData["name"])
}

func TestGeneratePlanFromTemplate_UnknownClassificationFallsBack(t *testing.T) {
	plan, _, err := GeneratePlanFromTemplate("unknown_class", &entity.User{}, nil, nil, PlanConstraints{
		DurationWeeks: 1,
		AvailableDays: []int{1},
	})
	assert.NoError(t, err)
	assert.Equal(t, "endurance_basic", plan.Classification)
}

func TestGeneratePlanFromTemplate_EmptyAvailableDaysDefaults(t *testing.T) {
	plan, weeks, err := GeneratePlanFromTemplate("recovery", &entity.User{}, nil, nil, PlanConstraints{
		DurationWeeks: 1,
		AvailableDays: []int{},
	})
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 3, 5}, plan.AvailableDays)
	assert.Equal(t, 3, len(weeks[0].Days))
}

func TestGeneratePlanFromTemplate_EmptySurvey(t *testing.T) {
	_, _, err := GeneratePlanFromTemplate("endurance_basic", &entity.User{}, nil, nil, PlanConstraints{
		DurationWeeks: 1,
		AvailableDays: []int{1},
	})
	assert.NoError(t, err)
}

func TestGeneratePlanFromTemplate_ExercisesGenerated(t *testing.T) {
	_, weeks, err := GeneratePlanFromTemplate("power_hiit", &entity.User{}, nil, nil, PlanConstraints{
		DurationWeeks: 1,
		AvailableDays: []int{1},
	})
	assert.NoError(t, err)
	assert.NotNil(t, weeks)
	assert.Equal(t, 1, len(weeks))
	assert.Equal(t, 1, len(weeks[0].Days))
	assert.True(t, len(weeks[0].Days[0].Exercises) > 0)
}

func TestApplyModifiers_AgeOver60(t *testing.T) {
	adj := Personalize(PersonalizationInput{Age: 65, HeightCm: 170, WeightKg: 70})
	assert.InDelta(t, 0.4, applyModifiers(0.5, adj), 0.01)
}

func TestApplyModifiers_BMIOver35(t *testing.T) {
	adj := Personalize(PersonalizationInput{Age: 30, HeightCm: 170, WeightKg: 110})
	assert.InDelta(t, 0.35, applyModifiers(0.5, adj), 0.01)
}

func TestApplyModifiers_MenstrualPhase(t *testing.T) {
	adj := Personalize(PersonalizationInput{
		Age: 30, HeightCm: 170, WeightKg: 70, Gender: "female", MenstrualPhase: "menstruation",
	})
	assert.InDelta(t, 0.4, applyModifiers(0.5, adj), 0.01)
}

func TestApplyModifiers_LowHRV(t *testing.T) {
	adj := Personalize(PersonalizationInput{
		Age: 30, HeightCm: 170, WeightKg: 70,
		Biometrics: &Biometrics{HRV: 10, HeartRate: 80},
	})
	assert.InDelta(t, 0.4, applyModifiers(0.8, adj), 0.01)
}

func TestApplyModifiers_LowSleep(t *testing.T) {
	adj := Personalize(PersonalizationInput{
		Age: 30, HeightCm: 170, WeightKg: 70, SleepHours: 4,
	})
	assert.InDelta(t, 0.35, applyModifiers(0.5, adj), 0.01)
}

func TestApplyModifiers_ClampedBetween01(t *testing.T) {
	adj := Personalize(PersonalizationInput{
		Age: 30, HeightCm: 170, WeightKg: 70, SleepHours: 3,
		Biometrics: &Biometrics{HRV: 5},
	})
	intensity := applyModifiers(0.1, adj)
	assert.GreaterOrEqual(t, intensity, 0.1)
	assert.LessOrEqual(t, intensity, 1.0)
}

func TestToInt32PanicsOnOverflow(t *testing.T) {
	assert.Panics(t, func() {
		toInt32(math.MaxInt32 + 1)
	})
	assert.Panics(t, func() {
		toInt32(math.MinInt32 - 1)
	})
}

func TestGeneratePlanFromTemplate_EmptyExercisesTemplate(t *testing.T) {
	original := templates.TrainingTemplates["recovery"]
	templates.TrainingTemplates["recovery"] = templates.TrainingTemplate{
		NameRu:         original.NameRu,
		DurationRange:  original.DurationRange,
		IntensityRange: original.IntensityRange,
		RestRatio:      original.RestRatio,
		Exercises:      []string{},
	}
	defer func() {
		templates.TrainingTemplates["recovery"] = original
	}()

	plan, weeks, err := GeneratePlanFromTemplate("recovery", &entity.User{}, nil, nil, PlanConstraints{
		DurationWeeks: 2,
		AvailableDays: []int{1},
	})
	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Empty(t, weeks[0].Days[0].Exercises)
}
