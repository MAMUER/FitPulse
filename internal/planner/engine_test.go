package planner

import (
	"testing"

	"github.com/MAMUER/project/internal/domain/entity"
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
	user := &entity.User{Age: 65, HeightCm: 170, WeightKg: 70}
	intensity := applyModifiers(0.5, user, nil, nil)
	assert.Equal(t, 0.4, intensity, 0.01)
}

func TestApplyModifiers_BMIOver35(t *testing.T) {
	user := &entity.User{Age: 30, HeightCm: 170, WeightKg: 110}
	intensity := applyModifiers(0.5, user, nil, nil)
	assert.Equal(t, 0.35, intensity, 0.01)
}

func TestApplyModifiers_Menstruation(t *testing.T) {
	user := &entity.User{Age: 30, HeightCm: 170, WeightKg: 70}
	survey := map[string]interface{}{"menstruation": true}
	intensity := applyModifiers(0.5, user, survey, nil)
	assert.Equal(t, 0.425, intensity, 0.01)
}

func TestApplyModifiers_LowHRV(t *testing.T) {
	user := &entity.User{Age: 30, HeightCm: 170, WeightKg: 70}
	biometrics := &Biometrics{HRV: 10, HeartRate: 80}
	intensity := applyModifiers(0.8, user, nil, biometrics)
	assert.Equal(t, 0.4, intensity, 0.01)
}

func TestApplyModifiers_LowSleep(t *testing.T) {
	user := &entity.User{Age: 30, HeightCm: 170, WeightKg: 70}
	biometrics := &Biometrics{HRV: 50, SleepHours: 4}
	intensity := applyModifiers(0.5, user, nil, biometrics)
	assert.Equal(t, 0.35, intensity, 0.01)
}

func TestApplyModifiers_ClampedBetween01(t *testing.T) {
	user := &entity.User{Age: 30, HeightCm: 170, WeightKg: 70}
	biometrics := &Biometrics{HRV: 5, SleepHours: 3}
	intensity := applyModifiers(0.1, user, nil, biometrics)
	assert.GreaterOrEqual(t, intensity, 0.1)
	assert.LessOrEqual(t, intensity, 1.0)
}

func TestGeneratePlanFromTemplate_PlanDataFields(t *testing.T) {
	plan, _, err := GeneratePlanFromTemplate("endurance_basic", &entity.User{}, nil, nil, PlanConstraints{
		DurationWeeks: 4,
		AvailableDays: []int{2, 4},
	})
	assert.NoError(t, err)
	assert.Equal(t, "Базовая выносливость", plan.PlanData["name"])
	assert.Equal(t, "endurance_basic", plan.PlanData["class"])
	assert.Equal(t, 4, plan.PlanData["weeks"])
}
