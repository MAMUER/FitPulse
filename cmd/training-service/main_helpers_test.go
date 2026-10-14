package main

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrainingToInt32(t *testing.T) {
	assert.Equal(t, int32(5), toInt32(5))
	assert.Equal(t, int32(0), toInt32(0))
	assert.Equal(t, int32(math.MaxInt32), toInt32(math.MaxInt32))
	assert.Equal(t, int32(0), toInt32(math.MaxInt32+1))
	assert.Equal(t, int32(0), toInt32(math.MinInt32-1))
}

func TestMapClassToTrainingGoal(t *testing.T) {
	cases := map[string]string{
		"recovery":            "recovery",
		"endurance_basic":     "endurance_e1e2",
		"endurance_threshold": "threshold_e3",
		"power_hiit":          "strength_hiit",
		"overtraining":        "recovery",
		"illness":             "recovery",
		"unknown":             "general_fitness",
	}
	for class, want := range cases {
		assert.Equal(t, want, mapClassToTrainingGoal(class), "класс %s", class)
	}

	// Неизвестный класс маппится в общую цель.
	assert.Equal(t, "general_fitness", mapClassToTrainingGoal("nonexistent_class"))
	assert.Equal(t, "general_fitness", mapClassToTrainingGoal(""))
}

func TestCalculatePlanDates(t *testing.T) {
	s := &trainingServer{}

	start, end := s.calculatePlanDates(4)

	require.WithinDuration(t, time.Now(), start, time.Minute)
	expectedEnd := start.AddDate(0, 0, 4*7)
	assert.Equal(t, expectedEnd, end)
	assert.True(t, end.After(start))

	_, zeroEnd := s.calculatePlanDates(0)
	assert.Equal(t, start.Format(time.RFC3339), zeroEnd.AddDate(0, 0, 0).Format(time.RFC3339))
}
