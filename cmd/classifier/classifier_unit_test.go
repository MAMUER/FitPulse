package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultIfZero(t *testing.T) {
	tests := []struct {
		name string
		val  float64
		def  float64
		want float64
	}{
		{"zero returns default", 0, 50.0, 50.0},
		{"nonzero returns value", 75.0, 50.0, 75.0},
		{"both zero", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultIfZero(tt.val, tt.def)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name string
		ss   []string
		sep  string
		want string
	}{
		{"empty slice", []string{}, ", ", ""},
		{"single element", []string{"a"}, ", ", "a"},
		{"multiple elements", []string{"a", "b", "c"}, ", ", "a, b, c"},
		{"empty separator", []string{"a", "b"}, "", "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinStrings(tt.ss, tt.sep)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClassifyState(t *testing.T) {
	tests := []struct {
		name         string
		data         physiologicalData
		age          int
		wantZone     int
		wantConfMin  float64
		wantConfMax  float64
		wantProbsLen int
	}{
		{
			name:         "recovery zone",
			data:         physiologicalData{HeartRate: 100, HeartRateVariability: 50},
			age:          30,
			wantZone:     0,
			wantConfMin:  0.35,
			wantConfMax:  1.0,
			wantProbsLen: 7,
		},
		{
			name:         "endurance zone",
			data:         physiologicalData{HeartRate: 140, HeartRateVariability: 50},
			age:          30,
			wantZone:     1,
			wantConfMin:  0.35,
			wantConfMax:  1.0,
			wantProbsLen: 7,
		},
		{
			name:         "threshold zone",
			data:         physiologicalData{HeartRate: 160, HeartRateVariability: 50},
			age:          30,
			wantZone:     2,
			wantConfMin:  0.35,
			wantConfMax:  1.0,
			wantProbsLen: 7,
		},
		{
			name:         "strength zone",
			data:         physiologicalData{HeartRate: 180, HeartRateVariability: 50},
			age:          30,
			wantZone:     3,
			wantConfMin:  0.35,
			wantConfMax:  1.0,
			wantProbsLen: 7,
		},
		{
			name:         "age too high uses default hrMax",
			data:         physiologicalData{HeartRate: 100, HeartRateVariability: 50},
			age:          250,
			wantZone:     0,
			wantConfMin:  0.35,
			wantConfMax:  1.0,
			wantProbsLen: 7,
		},
		{
			name:         "fever with low spo2 classifies as illness",
			data:         physiologicalData{Temperature: 37.7, SpO2: 93.0, HeartRate: 90, HeartRateVariability: 40},
			age:          30,
			wantZone:     5,
			wantConfMin:  0.55,
			wantConfMax:  0.95,
			wantProbsLen: 7,
		},
		{
			name:         "fever without supporting signs classifies as uncertainty",
			data:         physiologicalData{Temperature: 37.6, SpO2: 98.0, HeartRate: 90, HeartRateVariability: 50},
			age:          30,
			wantZone:     6,
			wantConfMin:  0.35,
			wantConfMax:  0.95,
			wantProbsLen: 7,
		},
		{
			name:         "mild temperature with low hrv is uncertainty",
			data:         physiologicalData{Temperature: 37.35, SpO2: 96.0, HeartRate: 90, HeartRateVariability: 24},
			age:          30,
			wantZone:     6,
			wantConfMin:  0.35,
			wantConfMax:  0.95,
			wantProbsLen: 7,
		},
		{
			name:         "low spo2 alone is uncertainty",
			data:         physiologicalData{Temperature: 36.6, SpO2: 92.0, HeartRate: 90, HeartRateVariability: 50},
			age:          30,
			wantZone:     6,
			wantConfMin:  0.35,
			wantConfMax:  0.95,
			wantProbsLen: 7,
		},
		{
			name:         "overtraining signs classify as overtraining",
			data:         physiologicalData{Temperature: 36.6, SpO2: 98.0, HeartRate: 55, HeartRateVariability: 22, SleepHours: 4.0},
			age:          30,
			wantZone:     4,
			wantConfMin:  0.55,
			wantConfMax:  0.90,
			wantProbsLen: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone, conf, probs := classifyState(tt.data, tt.age)
			assert.Equal(t, tt.wantZone, zone)
			assert.GreaterOrEqual(t, conf, tt.wantConfMin)
			assert.LessOrEqual(t, conf, tt.wantConfMax)
			assert.Len(t, probs, tt.wantProbsLen)
		})
	}
}

func TestGeneratePersonalizedNotes(t *testing.T) {
	tests := []struct {
		name      string
		data      physiologicalData
		profile   *userProfile
		predClass int
		wantNil   bool
	}{
		{
			name:      "nil profile returns nil",
			data:      physiologicalData{},
			profile:   nil,
			predClass: 0,
			wantNil:   true,
		},
		{
			name: "beginner gets advice",
			data: physiologicalData{},
			profile: &userProfile{
				Age:          30,
				FitnessLevel: "beginner",
			},
			predClass: 0,
			wantNil:   false,
		},
		{
			name: "age over 50 gets advice",
			data: physiologicalData{},
			profile: &userProfile{
				Age: 55,
			},
			predClass: 0,
			wantNil:   false,
		},
		{
			name: "health conditions listed",
			data: physiologicalData{},
			profile: &userProfile{
				Age:              30,
				HealthConditions: []string{"гипертония", "диабет"},
			},
			predClass: 0,
			wantNil:   false,
		},
		{
			name: "weight loss goal in recovery class",
			data: physiologicalData{},
			profile: &userProfile{
				Age:   30,
				Goals: []string{"похудение"},
			},
			predClass: 0,
			wantNil:   false,
		},
		{
			name: "no profile fields matched returns nil",
			data: physiologicalData{},
			profile: &userProfile{
				Age:          30,
				FitnessLevel: "advanced",
				Goals:        []string{"силовые"},
			},
			predClass: 3,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generatePersonalizedNotes(tt.data, tt.profile, tt.predClass)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.NotEmpty(t, *got)
			}
		})
	}
}
