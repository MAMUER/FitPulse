package planner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/domain/entity"
)

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	require.NoError(t, err)
	return parsed
}

func TestPersonalize_HealthyAdultNoChanges(t *testing.T) {
	adj := Personalize(PersonalizationInput{
		Age: 30, Gender: "male", WeightKg: 75, HeightCm: 180, Goal: "general_health",
	})
	assert.InDelta(t, 1.0, adj.IntensityFactor, 1e-9)
	assert.InDelta(t, 1.0, adj.DurationFactor, 1e-9)
	assert.Empty(t, adj.Notes)
}

func TestPersonalize_AgeAndBMI(t *testing.T) {
	t.Run("age over 60", func(t *testing.T) {
		adj := Personalize(PersonalizationInput{Age: 65, WeightKg: 75, HeightCm: 180})
		assert.Less(t, adj.IntensityFactor, 1.0)
	})

	t.Run("obesity", func(t *testing.T) {
		adj := Personalize(PersonalizationInput{Age: 40, WeightKg: 140, HeightCm: 180})
		assert.InDelta(t, 0.7, adj.IntensityFactor, 1e-9)
		assert.Less(t, adj.SetsFactor, 1.0)
	})

	t.Run("underweight", func(t *testing.T) {
		adj := Personalize(PersonalizationInput{Age: 25, WeightKg: 45, HeightCm: 180})
		assert.InDelta(t, 0.9, adj.IntensityFactor, 1e-9)
	})
}

func TestPersonalize_MenstrualPhaseOnlyForWomen(t *testing.T) {
	base := PersonalizationInput{Age: 28, Gender: "female", WeightKg: 60, HeightCm: 168}

	menses := base
	menses.MenstrualPhase = "menstruation"
	assert.InDelta(t, 0.8, Personalize(menses).IntensityFactor, 1e-9)

	luteal := base
	luteal.MenstrualPhase = "luteal"
	assert.InDelta(t, 0.9, Personalize(luteal).IntensityFactor, 1e-9)

	male := base
	male.Gender = "male"
	male.MenstrualPhase = "menstruation"
	assert.InDelta(t, 1.0, Personalize(male).IntensityFactor, 1e-9,
		"менструальная фаза не должна влиять на мужчин")
}

func TestPersonalize_Biometrics(t *testing.T) {
	t.Run("low HRV caps intensity", func(t *testing.T) {
		adj := Personalize(PersonalizationInput{
			Age: 30, WeightKg: 75, HeightCm: 180,
			Biometrics: &Biometrics{HRV: 15, HeartRate: 55, Spo2: 98},
		})
		assert.InDelta(t, 0.4, adj.IntensityCap, 1e-9)
		assert.InDelta(t, 0.4, applyModifiers(0.8, adj), 1e-9)
	})

	t.Run("high resting HR", func(t *testing.T) {
		adj := Personalize(PersonalizationInput{
			Age: 30, WeightKg: 75, HeightCm: 180,
			Biometrics: &Biometrics{HRV: 60, HeartRate: 105, Spo2: 98},
		})
		assert.InDelta(t, 0.7, adj.IntensityFactor, 1e-9)
	})

	t.Run("fever reduces load", func(t *testing.T) {
		adj := Personalize(PersonalizationInput{
			Age: 30, WeightKg: 75, HeightCm: 180,
			Biometrics: &Biometrics{HRV: 60, HeartRate: 70, Spo2: 98, Temperature: 38.5},
		})
		assert.InDelta(t, 0.6, adj.IntensityFactor, 1e-9)
		assert.Less(t, adj.DurationFactor, 1.0)
	})
}

func TestPersonalize_RestrictionsAndStress(t *testing.T) {
	adj := Personalize(PersonalizationInput{
		Age: 40, WeightKg: 75, HeightCm: 180,
		Restrictions: []string{"Гипертония"},
		StressLevel:  8,
	})
	// 0.75 (гипертония) * 0.75 (стресс) = 0.5625
	assert.InDelta(t, 0.5625, adj.IntensityFactor, 1e-9)
	require.NotEmpty(t, adj.Notes)
}

func TestPersonalize_Sleep(t *testing.T) {
	adj := Personalize(PersonalizationInput{Age: 30, WeightKg: 75, HeightCm: 180, SleepHours: 4.5})
	assert.InDelta(t, 0.7, adj.IntensityFactor, 1e-9)
}

func TestPersonalizeVector_KeepsLengthAndScales(t *testing.T) {
	vec := make([]float64, 19)
	for i := range vec {
		vec[i] = 0.8
	}

	out, notes := PersonalizeVector(vec, PersonalizationInput{
		Age: 30, WeightKg: 75, HeightCm: 180,
		Biometrics: &Biometrics{HRV: 15, HeartRate: 55, Spo2: 98},
	})

	require.Len(t, out, 19, "длина вектора GAN не должна меняться")
	assert.Less(t, out[0], vec[0], "интенсивность должна снизиться")
	assert.NotEmpty(t, notes)
	// Исходный вектор не мутируем.
	assert.InDelta(t, 0.8, vec[0], 1e-9)
}

func TestPersonalizationFromUser_ParsesSurvey(t *testing.T) {
	user := &entity.User{Age: 32, Gender: "female", WeightKg: 62, HeightCm: 170, Goals: []string{"weight_loss"}}
	survey := map[string]interface{}{
		"allergies":    []interface{}{"Орехи", "Молоко"},
		"restrictions": []interface{}{"Сахарный диабет 2 типа"},
		"sleepHours":   "6–7 ч",
		"goals":        []interface{}{"Снизить стресс"},
	}

	in := PersonalizationFromUser(user, survey, nil)

	assert.Equal(t, []string{"Орехи", "Молоко"}, in.Allergies)
	require.Len(t, in.Restrictions, 1)
	assert.Contains(t, in.Restrictions[0], "диабет")
	assert.InDelta(t, 6.0, in.SleepHours, 1e-9)
	assert.InDelta(t, 6.0, in.StressLevel, 1e-9, "цель «Снизить стресс» косвенно повышает стресс")
}

func TestMenstrualPhaseFromCycle(t *testing.T) {
	now := mustTime(t, "2026-11-22")

	tests := []struct {
		start string
		want  string
	}{
		{"2026-11-22", "menstruation"}, // день 1
		{"2026-11-14", "follicular"},   // день 9
		{"2026-11-08", "ovulation"},    // день 15
		{"2026-11-01", "luteal"},       // день 22
	}
	for _, tc := range tests {
		phase, _ := MenstrualPhaseFromCycle(tc.start, now)
		assert.Equalf(t, tc.want, phase, "start=%s", tc.start)
	}

	phase, _ := MenstrualPhaseFromCycle("", now)
	assert.Empty(t, phase)
}

func TestGeneratePlanFromGAN_Personalized(t *testing.T) {
	vec := make([]float64, 19)
	for i := range vec {
		vec[i] = 0.8
	}

	constraints := PlanConstraints{DurationWeeks: 1, AvailableDays: []int{1, 3}}
	personalization := PersonalizationInput{
		Age: 70, Gender: "female", WeightKg: 100, HeightCm: 165,
		MenstrualPhase: "menstruation",
		Biometrics:     &Biometrics{HRV: 15, HeartRate: 100, Spo2: 98},
	}

	plan, weeks, err := GeneratePlanFromGAN("endurance_basic", vec, constraints, personalization)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Len(t, weeks, 1)

	notes, ok := plan.PlanData["personalization"].([]string)
	require.True(t, ok, "персонализация должна попасть в plan_data")
	assert.NotEmpty(t, notes)

	// Интенсивность в плане должна быть ниже исходных 0.8.
	intensity, ok := plan.PlanData["intensity"].(float64)
	require.True(t, ok)
	assert.Less(t, intensity, 0.8)
}

func TestGeneratePlanFromTemplate_Personalized(t *testing.T) {
	user := &entity.User{Age: 70, Gender: "female", WeightKg: 100, HeightCm: 165}
	plan, weeks, err := GeneratePlanFromTemplate(
		"endurance_basic",
		user,
		map[string]interface{}{"restrictions": []interface{}{"Гипертония"}},
		&Biometrics{HRV: 15, HeartRate: 100, Spo2: 98},
		PlanConstraints{DurationWeeks: 1, AvailableDays: []int{1, 3}},
	)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Len(t, weeks, 1)
	assert.NotEmpty(t, plan.PlanData["personalization"])
}
