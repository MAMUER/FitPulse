package main

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDayNameToIndex(t *testing.T) {
	tests := []struct {
		name string
		day  string
		want int
	}{
		{"monday", "monday", 0},
		{"Monday", "Monday", 0},
		{"tuesday", "tuesday", 1},
		{"wednesday", "wednesday", 2},
		{"thursday", "thursday", 3},
		{"friday", "friday", 4},
		{"saturday", "saturday", 5},
		{"sunday", "sunday", 6},
		{"invalid", "funday", -1},
		{"empty", "", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dayNameToIndex(tt.day)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateBasicWeeklyWorkouts(t *testing.T) {
	tests := []struct {
		name          string
		trainingClass string
		availableDays []int32
		wantLen       int
		wantType      string
	}{
		{
			name:          "recovery class with 3 days",
			trainingClass: "recovery",
			availableDays: []int32{0, 2, 4},
			wantLen:       3,
			wantType:      "recovery",
		},
		{
			name:          "strength class with 2 days",
			trainingClass: "strength",
			availableDays: []int32{1, 3},
			wantLen:       2,
			wantType:      "strength",
		},
		{
			name:          "unknown class falls back to recovery",
			trainingClass: "unknown",
			availableDays: []int32{0},
			wantLen:       1,
			wantType:      "recovery",
		},
		{
			name:          "empty available days",
			trainingClass: "recovery",
			availableDays: []int32{},
			wantLen:       0,
			wantType:      "recovery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateBasicWeeklyWorkouts(tt.trainingClass, tt.availableDays)
			assert.Len(t, got, tt.wantLen)
			for _, w := range got {
				assert.Equal(t, tt.wantType, w["type"])
				assert.NotNil(t, w["duration"])
				assert.NotNil(t, w["exercises"])
			}
		})
	}
}

func TestStringValue(t *testing.T) {
	tests := []struct {
		name string
		ns   sql.NullString
		want string
	}{
		{"valid string", sql.NullString{String: "hello", Valid: true}, "hello"},
		{"invalid null", sql.NullString{Valid: false}, ""},
		{"empty valid", sql.NullString{String: "", Valid: true}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringValue(tt.ns)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInt32Value(t *testing.T) {
	tests := []struct {
		name string
		ni   sql.NullInt32
		want int32
	}{
		{"valid int", sql.NullInt32{Int32: 42, Valid: true}, 42},
		{"invalid null", sql.NullInt32{Valid: false}, 0},
		{"zero valid", sql.NullInt32{Int32: 0, Valid: true}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := int32Value(tt.ni)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFloat64Value(t *testing.T) {
	tests := []struct {
		name string
		nf   sql.NullFloat64
		want float64
	}{
		{"valid float", sql.NullFloat64{Float64: 3.14, Valid: true}, 3.14},
		{"invalid null", sql.NullFloat64{Valid: false}, 0},
		{"zero valid", sql.NullFloat64{Float64: 0, Valid: true}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := float64Value(tt.nf)
			assert.Equal(t, tt.want, got)
		})
	}
}
