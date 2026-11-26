package planner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPersonalize_AgeBranches(t *testing.T) {
	young := Personalize(PersonalizationInput{Age: 16, WeightKg: 60, HeightCm: 170})
	assert.Less(t, young.IntensityFactor, 1.0)

	old := Personalize(PersonalizationInput{Age: 80, WeightKg: 60, HeightCm: 170})
	assert.Less(t, old.IntensityFactor, 0.7)
	assert.Negative(t, old.SessionsDelta)
}

func TestPersonalize_BMIBranches(t *testing.T) {
	under := Personalize(PersonalizationInput{Age: 30, WeightKg: 45, HeightCm: 180})
	assert.InDelta(t, 0.9, under.IntensityFactor, 1e-9)

	over := Personalize(PersonalizationInput{Age: 30, WeightKg: 120, HeightCm: 180})
	assert.InDelta(t, 0.7, over.IntensityFactor, 1e-9)
}

func TestPersonalize_Restrictions(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  float64
	}{
		{"heart", "Сердечная недостаточность", 0.6},
		{"hypertension", "Гипертония", 0.75},
		{"diabetes", "Сахарный диабет", 0.85},
		{"kidney", "Заболевание почек", 0.8},
		{"liver", "Заболевание печени", 0.8},
		{"orthopedic", "Ортопедическое ограничение", 0.8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adj := Personalize(PersonalizationInput{
				Age: 40, WeightKg: 75, HeightCm: 180, Restrictions: []string{tc.value},
			})
			assert.InDelta(t, tc.want, adj.IntensityFactor, 1e-9)
		})
	}
}

func TestPersonalize_Goals(t *testing.T) {
	loss := Personalize(PersonalizationInput{Age: 30, WeightKg: 80, HeightCm: 180, Goal: "weight_loss"})
	assert.InDelta(t, 0.95, loss.IntensityFactor, 1e-9)

	gain := Personalize(PersonalizationInput{Age: 30, WeightKg: 80, HeightCm: 180, Goal: "muscle_gain"})
	assert.InDelta(t, 1.0, gain.IntensityFactor, 1e-9)
}

func TestPersonalize_SleepAndBiometrics(t *testing.T) {
	lowSleep := Personalize(PersonalizationInput{Age: 30, WeightKg: 75, HeightCm: 180, SleepHours: 6})
	assert.InDelta(t, 0.85, lowSleep.IntensityFactor, 1e-9)

	lowSpo2 := Personalize(PersonalizationInput{
		Age: 30, WeightKg: 75, HeightCm: 180,
		Biometrics: &Biometrics{HRV: 50, HeartRate: 80, Spo2: 92},
	})
	assert.InDelta(t, 0.7, lowSpo2.IntensityFactor, 1e-9)

	stress := Personalize(PersonalizationInput{Age: 30, WeightKg: 75, HeightCm: 180, StressLevel: 6})
	assert.InDelta(t, 0.9, stress.IntensityFactor, 1e-9)
}

func TestPersonalizeVector_Empty(t *testing.T) {
	out, notes := PersonalizeVector(nil, PersonalizationInput{Age: 30, WeightKg: 75, HeightCm: 180})
	assert.Nil(t, out)
	assert.Empty(t, notes)
}

func TestMenstrualPhaseFromCycle_Extra(t *testing.T) {
	now, _ := time.Parse("2006-01-02", "2026-11-22")
	cases := map[string]string{
		"2026-11-22": "menstruation",
		"2026-11-14": "follicular",
		"2026-11-08": "ovulation",
		"2026-11-01": "luteal",
	}
	for start, want := range cases {
		phase, _ := MenstrualPhaseFromCycle(start, now)
		assert.Equalf(t, want, phase, "start=%s", start)
	}
	phase, _ := MenstrualPhaseFromCycle("", now)
	assert.Empty(t, phase)
}
