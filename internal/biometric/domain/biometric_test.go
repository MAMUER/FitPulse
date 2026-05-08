package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllMetricTypes(t *testing.T) {
	metrics := AllMetricTypes()

	expected := []MetricType{
		MetricHeartRate, MetricHRV, MetricSpO2, MetricTemperature,
		MetricBloodPressureSys, MetricBloodPressureDia, MetricECG,
		MetricSleepStage, MetricSteps, MetricDistance, MetricCalories,
		MetricRespiratoryRate, MetricBloodGlucose, MetricOxygenSaturation,
	}

	assert.Equal(t, expected, metrics)
	assert.Len(t, metrics, 14)
}

func TestVendorCapabilities(t *testing.T) {
	caps := VendorCapabilities()

	assert.Contains(t, caps, "apple")
	assert.Contains(t, caps, "samsung")
	assert.Contains(t, caps, "huawei")
	assert.Contains(t, caps, "amazfit")

	// Test Apple capabilities
	apple := caps["apple"]
	assert.True(t, apple[MetricHeartRate])
	assert.True(t, apple[MetricECG])
	assert.False(t, apple[MetricTemperature])

	// Test Samsung capabilities
	samsung := caps["samsung"]
	assert.True(t, samsung[MetricHeartRate])
	assert.True(t, samsung[MetricTemperature])
	assert.False(t, samsung[MetricBloodPressureSys])

	// Test Huawei capabilities
	huawei := caps["huawei"]
	assert.True(t, huawei[MetricHeartRate])
	assert.True(t, huawei[MetricBloodPressureSys])

	// Test Amazfit capabilities
	amazfit := caps["amazfit"]
	assert.True(t, amazfit[MetricHeartRate])
	assert.False(t, amazfit[MetricECG])
}

func TestStandardMedicalCodes(t *testing.T) {
	codes := StandardMedicalCodes()

	assert.Contains(t, codes, "I10")
	assert.Contains(t, codes, "I21")
	assert.Contains(t, codes, "E11")

	// Test specific codes
	i10 := codes["I10"]
	assert.Equal(t, "Essential hypertension", i10.Label)
	assert.Equal(t, "cardiovascular", i10.Category)
	assert.Equal(t, "moderate", i10.Severity)

	i21 := codes["I21"]
	assert.Equal(t, "Acute myocardial infarction", i21.Label)
	assert.Equal(t, "cardiovascular", i21.Category)
	assert.Equal(t, "severe", i21.Severity)
}

func TestGetStandardCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected struct {
			Label    string
			Category string
			Severity string
		}
		found bool
	}{
		{
			name: "existing code I10",
			code: "I10",
			expected: struct {
				Label    string
				Category string
				Severity string
			}{"Essential hypertension", "cardiovascular", "moderate"},
			found: true,
		},
		{
			name: "existing code E11",
			code: "E11",
			expected: struct {
				Label    string
				Category string
				Severity string
			}{"Type 2 diabetes", "endocrine", "moderate"},
			found: true,
		},
		{
			name: "non-existing code",
			code: "NONEXISTENT",
			expected: struct {
				Label    string
				Category string
				Severity string
			}{"", "", ""},
			found: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, found := GetStandardCode(tt.code)
			assert.Equal(t, tt.found, found)
			if found {
				assert.Equal(t, tt.expected.Label, info.Label)
				assert.Equal(t, tt.expected.Category, info.Category)
				assert.Equal(t, tt.expected.Severity, info.Severity)
			}
		})
	}
}

func TestMedicalConstraintValidate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		constraint MedicalConstraint
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid constraint",
			constraint: MedicalConstraint{
				ID:       "test-1",
				Code:     "I10",
				Label:    "Hypertension",
				Category: "cardiovascular",
				Severity: "moderate",
				ImpactOnTraining: []ImpactRule{
					{
						Metric: "heart_rate",
						Action: "caution",
					},
				},
				ValidatedBy: &[]string{"doctor@example.com"}[0],
				ValidatedAt: &now,
				Active:      true,
			},
			wantErr: false,
		},
		{
			name: "missing code",
			constraint: MedicalConstraint{
				ID:       "test-2",
				Label:    "Hypertension",
				Category: "cardiovascular",
				Severity: "moderate",
				ImpactOnTraining: []ImpactRule{
					{
						Metric: "heart_rate",
						Action: "caution",
					},
				},
				Active: true,
			},
			wantErr: true,
			errMsg:  "code is required",
		},
		{
			name: "missing label",
			constraint: MedicalConstraint{
				ID:       "test-3",
				Code:     "I10",
				Category: "cardiovascular",
				Severity: "moderate",
				ImpactOnTraining: []ImpactRule{
					{
						Metric: "heart_rate",
						Action: "caution",
					},
				},
				Active: true,
			},
			wantErr: true,
			errMsg:  "label is required",
		},
		{
			name: "missing category",
			constraint: MedicalConstraint{
				ID:    "test-4",
				Code:  "I10",
				Label: "Hypertension",
				Severity: "moderate",
				ImpactOnTraining: []ImpactRule{
					{
						Metric: "heart_rate",
						Action: "caution",
					},
				},
				Active: true,
			},
			wantErr: true,
			errMsg:  "category is required",
		},
		{
			name: "missing impact rules",
			constraint: MedicalConstraint{
				ID:       "test-5",
				Code:     "I10",
				Label:    "Hypertension",
				Category: "cardiovascular",
				Severity: "moderate",
				Active:   true,
			},
			wantErr: true,
			errMsg:  "impact rules required",
		},
		{
			name: "empty metric in impact rule",
			constraint: MedicalConstraint{
				ID:       "test-6",
				Code:     "I10",
				Label:    "Hypertension",
				Category: "cardiovascular",
				Severity: "moderate",
				ImpactOnTraining: []ImpactRule{
					{
						Action: "caution",
					},
				},
				Active: true,
			},
			wantErr: true,
			errMsg:  "metric required",
		},
		{
			name: "empty action in impact rule",
			constraint: MedicalConstraint{
				ID:       "test-7",
				Code:     "I10",
				Label:    "Hypertension",
				Category: "cardiovascular",
				Severity: "moderate",
				ImpactOnTraining: []ImpactRule{
					{
						Metric: "heart_rate",
					},
				},
				Active: true,
			},
			wantErr: true,
			errMsg:  "action required",
		},
		{
			name: "invalid action in impact rule",
			constraint: MedicalConstraint{
				ID:       "test-8",
				Code:     "I10",
				Label:    "Hypertension",
				Category: "cardiovascular",
				Severity: "moderate",
				ImpactOnTraining: []ImpactRule{
					{
						Metric: "heart_rate",
						Action: "invalid_action",
					},
				},
				Active: true,
			},
			wantErr: true,
			errMsg:  "invalid action: invalid_action",
		},
		{
			name: "valid actions",
			constraint: MedicalConstraint{
				ID:       "test-9",
				Code:     "I10",
				Label:    "Hypertension",
				Category: "cardiovascular",
				Severity: "moderate",
				ImpactOnTraining: []ImpactRule{
					{Metric: "heart_rate", Action: "modify"},
					{Metric: "blood_pressure", Action: "avoid"},
					{Metric: "steps", Action: "caution"},
					{Metric: "temperature", Action: "require_approval"},
				},
				Active: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constraint.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBiometricSampleStruct(t *testing.T) {
	now := time.Now()
	sample := BiometricSample{
		UserID:     "user123",
		DeviceID:   "device456",
		DeviceType: "apple_watch",
		MetricType: "heart_rate",
		Value:      75.5,
		Unit:       "bpm",
		Timestamp:  now,
		Quality:    "good",
		Confidence: 0.95,
		SourceID:   "apple_health",
		Metadata: map[string]interface{}{
			"accuracy": 0.98,
			"notes":    "measured during exercise",
		},
	}

	assert.Equal(t, "user123", sample.UserID)
	assert.Equal(t, "device456", sample.DeviceID)
	assert.Equal(t, "apple_watch", sample.DeviceType)
	assert.Equal(t, "heart_rate", sample.MetricType)
	assert.Equal(t, 75.5, sample.Value)
	assert.Equal(t, "bpm", sample.Unit)
	assert.Equal(t, now, sample.Timestamp)
	assert.Equal(t, "good", sample.Quality)
	assert.Equal(t, 0.95, sample.Confidence)
	assert.Equal(t, "apple_health", sample.SourceID)
	assert.NotNil(t, sample.Metadata)
	assert.Equal(t, 0.98, sample.Metadata["accuracy"])
	assert.Equal(t, "measured during exercise", sample.Metadata["notes"])
}

func TestWorkoutPlanStruct(t *testing.T) {
	now := time.Now()
	plan := WorkoutPlan{
		ID:                     "plan123",
		UserID:                 "user456",
		Name:                   "Morning Cardio",
		Type:                   "cardio",
		DurationMinutes:        45,
		TargetHeartRate:        140,
		MaxHeartRate:           180,
		Intensity:              "moderate",
		BloodPressureSystolic:  120,
		BloodPressureDiastolic: 80,
		Exercises: []Exercise{
			{
				ID:          "ex1",
				Name:        "Running",
				Sets:        1,
				Reps:        1,
				Weight:      0,
				RestSeconds: 0,
				Notes:       "Warm up for 5 minutes",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "plan123", plan.ID)
	assert.Equal(t, "user456", plan.UserID)
	assert.Equal(t, "Morning Cardio", plan.Name)
	assert.Equal(t, "cardio", plan.Type)
	assert.Equal(t, 45, plan.DurationMinutes)
	assert.Equal(t, 140, plan.TargetHeartRate)
	assert.Equal(t, 180, plan.MaxHeartRate)
	assert.Equal(t, "moderate", plan.Intensity)
	assert.Equal(t, 120, plan.BloodPressureSystolic)
	assert.Equal(t, 80, plan.BloodPressureDiastolic)
	assert.Len(t, plan.Exercises, 1)
	assert.Equal(t, now, plan.CreatedAt)
	assert.Equal(t, now, plan.UpdatedAt)
}

func TestExerciseStruct(t *testing.T) {
	exercise := Exercise{
		ID:          "ex123",
		Name:        "Push-ups",
		Sets:        3,
		Reps:        15,
		Weight:      0,
		RestSeconds: 60,
		Notes:       "Keep core tight",
	}

	assert.Equal(t, "ex123", exercise.ID)
	assert.Equal(t, "Push-ups", exercise.Name)
	assert.Equal(t, 3, exercise.Sets)
	assert.Equal(t, 15, exercise.Reps)
	assert.Equal(t, 0.0, exercise.Weight)
	assert.Equal(t, 60, exercise.RestSeconds)
	assert.Equal(t, "Keep core tight", exercise.Notes)
}

func TestConstraintViolationStruct(t *testing.T) {
	violation := ConstraintViolation{
		ConstraintID: "constraint123",
		Type:         "threshold_exceeded",
		Metric:       "heart_rate",
		ActualValue:  180.0,
		Threshold:    160.0,
		Description:  "Heart rate exceeded safe threshold",
		Action:       "stop_exercise",
	}

	assert.Equal(t, "constraint123", violation.ConstraintID)
	assert.Equal(t, "threshold_exceeded", violation.Type)
	assert.Equal(t, "heart_rate", violation.Metric)
	assert.Equal(t, 180.0, violation.ActualValue)
	assert.Equal(t, 160.0, violation.Threshold)
	assert.Equal(t, "Heart rate exceeded safe threshold", violation.Description)
	assert.Equal(t, "stop_exercise", violation.Action)
}

func TestModificationSuggestionStruct(t *testing.T) {
	suggestion := ModificationSuggestion{
		ConstraintID:   "constraint456",
		Type:           "reduce_intensity",
		Metric:         "heart_rate",
		CurrentValue:   170.0,
		SuggestedValue: 140.0,
		Reason:         "High heart rate indicates overexertion",
		Priority:       1,
	}

	assert.Equal(t, "constraint456", suggestion.ConstraintID)
	assert.Equal(t, "reduce_intensity", suggestion.Type)
	assert.Equal(t, "heart_rate", suggestion.Metric)
	assert.Equal(t, 170.0, suggestion.CurrentValue)
	assert.Equal(t, 140.0, suggestion.SuggestedValue)
	assert.Equal(t, "High heart rate indicates overexertion", suggestion.Reason)
	assert.Equal(t, 1, suggestion.Priority)
}