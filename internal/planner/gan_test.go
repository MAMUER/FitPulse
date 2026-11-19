package planner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeGANVector_EmptyUsesDefaults(t *testing.T) {
	p := DecodeGANVector(nil)
	assert.Equal(t, 0.5, p.Intensity)
	assert.Equal(t, 45, p.DurationMinutes)
	assert.Equal(t, 3, p.SessionsPerWeek)
	assert.Equal(t, 3, p.Sets)
	assert.Equal(t, 12, p.Reps)
	assert.Equal(t, 60, p.RestSeconds)
}

func TestDecodeGANVector_FullVector(t *testing.T) {
	// 19 значений: 0..18.
	vec := make([]float64, 19)
	vec[0] = 0.8 // intensity
	vec[1] = 0.5 // duration → 30+30=60
	vec[2] = 0.5 // sessions → 1+3=4
	vec[4] = 0.5 // sets → 2+2=4
	vec[5] = 0.5 // reps → 6+7=13
	vec[6] = 0.5 // rest → 30+45=75
	vec[7] = 0.5 // exercise count → 3
	p := DecodeGANVector(vec)
	assert.InDelta(t, 0.8, p.Intensity, 1e-9)
	assert.Equal(t, 60, p.DurationMinutes)
	assert.Equal(t, 4, p.SessionsPerWeek)
	assert.Equal(t, 4, p.Sets)
	assert.Equal(t, 13, p.Reps)
	assert.Equal(t, 75, p.RestSeconds)
	assert.Equal(t, 3, p.ExerciseCount)
	assert.NotEmpty(t, p.SelectionWeights)
}

func TestDecodeGANVector_Clamps(t *testing.T) {
	vec := []float64{-1, 2, -1, 0, 5, -3, 9, 10} // out of range
	p := DecodeGANVector(vec)
	assert.Equal(t, 0.0, p.Intensity)      // clamped to 0
	assert.Equal(t, 90, p.DurationMinutes) // 2→1 → 30+60=90
	assert.Equal(t, 1, p.SessionsPerWeek)  // -1→0 → 1
	assert.Equal(t, 6, p.Sets)             // 5→1 → 2+int(1*4)=6
	assert.Equal(t, 6, p.Reps)             // -3→0 → 6+int(0*15)=6
	assert.Equal(t, 120, p.RestSeconds)    // 9→1 → 30+90=120
	assert.Equal(t, 6, p.ExerciseCount)    // 10→1 → int(1*6)=6
}

func TestSelectExercises_ByWeights(t *testing.T) {
	pool := []string{"a", "b", "c", "d"}
	params := GANPlanParams{SelectionWeights: []float64{0.1, 0.9, 0.5, 0.2}, ExerciseCount: 2}
	selected := selectExercises(pool, params)
	// weights: b=0.9, c=0.5, d=0.2, a=0.1 → top 2 = b, c
	assert.Equal(t, []string{"b", "c"}, selected)
}

func TestSelectExercises_NoWeightsReturnsPool(t *testing.T) {
	pool := []string{"a", "b"}
	assert.Equal(t, pool, selectExercises(pool, GANPlanParams{}))
}

func TestGeneratePlanFromGAN_ProducesPlan(t *testing.T) {
	vec := make([]float64, 19)
	vec[0] = 0.7
	vec[1] = 0.4
	vec[2] = 0.3
	plan, weeks, err := GeneratePlanFromGAN("endurance_basic", vec, PlanConstraints{
		DurationWeeks: 2,
		AvailableDays: []int{1, 3, 5},
	})
	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.NotEmpty(t, weeks)
	assert.Equal(t, "gan", plan.PlanData["generation_method"])
	assert.Equal(t, 2, len(weeks))
	for _, w := range weeks {
		assert.NotEmpty(t, w.Days)
		for _, d := range w.Days {
			assert.NotEmpty(t, d.Exercises)
			for _, ex := range d.Exercises {
				assert.Greater(t, ex.DurationMinutes, int32(0))
				assert.Greater(t, ex.Sets, int32(0))
				assert.Greater(t, ex.Reps, int32(0))
			}
		}
	}
}

func TestGeneratePlanFromGAN_EmptyVectorStillBuilds(t *testing.T) {
	plan, weeks, err := GeneratePlanFromGAN("recovery", nil, PlanConstraints{
		DurationWeeks: 1,
		AvailableDays: []int{1},
	})
	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.NotEmpty(t, weeks)
}

func TestGeneratePlanFromGAN_UnknownClassFallsBack(t *testing.T) {
	plan, _, err := GeneratePlanFromGAN("unknown", []float64{0.5}, PlanConstraints{
		DurationWeeks: 1,
		AvailableDays: []int{1},
	})
	assert.NoError(t, err)
	assert.Equal(t, "endurance_basic", plan.Classification)
}
