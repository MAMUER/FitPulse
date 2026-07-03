package medical

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/MAMUER/project/internal/biometric/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMedicalRepository is a mock implementation of MedicalRepository
type MockMedicalRepository struct {
	mock.Mock
}

func (m *MockMedicalRepository) GetActiveConstraints(ctx context.Context, userID string) ([]domain.MedicalConstraint, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.MedicalConstraint), fmt.Errorf("mock: %w", args.Error(1))
}

func (m *MockMedicalRepository) SaveConstraint(ctx context.Context, constraint domain.MedicalConstraint) error {
	args := m.Called(ctx, constraint)
	return fmt.Errorf("mock: %w", args.Error(0))
}

func (m *MockMedicalRepository) DeleteConstraint(ctx context.Context, constraintID string) error {
	args := m.Called(ctx, constraintID)
	return fmt.Errorf("mock: %w", args.Error(0))
}

func (m *MockMedicalRepository) GetConstraintByCode(ctx context.Context, code string) ([]domain.MedicalConstraint, error) {
	args := m.Called(ctx, code)
	return args.Get(0).([]domain.MedicalConstraint), fmt.Errorf("mock: %w", args.Error(1))
}

func TestNewMedicalService(t *testing.T) {
	mockRepo := &MockMedicalRepository{}
	service := NewMedicalService(mockRepo)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
}

const testUserID = "user123"

func TestMedicalServiceEvaluateWorkout_Case1(t *testing.T) {
	runTestCase(t, "no constraints", []domain.MedicalConstraint{}, nil, &domain.WorkoutPlan{
		TargetHeartRate: 140,
		MaxHeartRate:    180,
	}, nil, false)
}

func TestMedicalServiceEvaluateWorkout_Case2(t *testing.T) {
	runTestCase(t, "repository error", nil, errors.New("database error"), &domain.WorkoutPlan{
		TargetHeartRate: 140,
	}, nil, false)
}

func TestMedicalServiceEvaluateWorkout_Case3(t *testing.T) {
	constraint := domain.MedicalConstraint{
		ID:    "constraint1",
		Code:  "I10",
		Label: "Hypertension",
		ImpactOnTraining: []domain.ImpactRule{{
			Metric:       "heart_rate_max",
			ThresholdMax: func() *float64 { v := 160.0; return &v }(),
			Action:       "caution",
		}},
	}
	runTestCase(t, "heart rate above threshold", []domain.MedicalConstraint{constraint}, nil, &domain.WorkoutPlan{
		TargetHeartRate: 150,
		MaxHeartRate:    170,
	}, []domain.ConstraintViolation{{
		ConstraintID: "constraint1",
		Type:         "above_threshold",
		Metric:       "heart_rate_max",
		ActualValue:  170.0,
		Threshold:    160.0,
		Description:  "heart_rate_max exceeds maximum threshold (160.0)",
		Action:       "caution",
	}}, false)
}

func TestMedicalServiceEvaluateWorkout_Case4(t *testing.T) {
	constraint := domain.MedicalConstraint{
		ID:    "constraint2",
		Code:  "I10",
		Label: "Hypertension",
		ImpactOnTraining: []domain.ImpactRule{{
			Metric:       "blood_pressure_systolic",
			ThresholdMax: func() *float64 { v := 140.0; return &v }(),
			Action:       "require_approval",
		}},
	}
	runTestCase(t, "blood pressure violation requiring approval", []domain.MedicalConstraint{constraint}, nil, &domain.WorkoutPlan{
		BloodPressureSystolic: 150,
	}, []domain.ConstraintViolation{{
		ConstraintID: "constraint2",
		Type:         "above_threshold",
		Metric:       "blood_pressure_systolic",
		ActualValue:  150.0,
		Threshold:    140.0,
		Description:  "blood_pressure_systolic exceeds maximum threshold (140.0)",
		Action:       "require_approval",
	}}, true)
}

func TestMedicalServiceEvaluateWorkout_Case5(t *testing.T) {
	constraint := domain.MedicalConstraint{
		ID:    "constraint3",
		Code:  "I50",
		Label: "Heart Failure",
		ImpactOnTraining: []domain.ImpactRule{{
			Metric:       "heart_rate",
			ThresholdMin: func() *float64 { v := 100.0; return &v }(),
			Action:       "modify",
		}},
	}
	runTestCase(t, "below threshold violation", []domain.MedicalConstraint{constraint}, nil, &domain.WorkoutPlan{
		TargetHeartRate: 90,
	}, []domain.ConstraintViolation{{
		ConstraintID: "constraint3",
		Type:         "below_threshold",
		Metric:       "heart_rate",
		ActualValue:  90.0,
		Threshold:    100.0,
		Description:  "heart_rate below minimum threshold (100.0)",
		Action:       "modify",
	}}, false)
}

func TestMedicalServiceEvaluateWorkout_Case6(t *testing.T) {
	constraint := domain.MedicalConstraint{
		ID:    "constraint4",
		Code:  "E11",
		Label: "Diabetes",
		ImpactOnTraining: []domain.ImpactRule{{
			Metric: "glucose_level",
			Action: "caution",
		}},
	}
	runTestCase(t, "unknown metric", []domain.MedicalConstraint{constraint}, nil, &domain.WorkoutPlan{
		TargetHeartRate: 140,
	}, []domain.ConstraintViolation{{
		ConstraintID: "constraint4",
		Type:         "unknown_metric",
		Description:  "Metric 'glucose_level' not recognized, requires manual review",
		Action:       "require_approval",
	}}, true)
}

func TestMedicalServiceEvaluateWorkout_Case7(t *testing.T) {
	constraints := []domain.MedicalConstraint{
		{
			ID:    "constraint5",
			Code:  "I10",
			Label: "Hypertension",
			ImpactOnTraining: []domain.ImpactRule{{
				Metric:       "heart_rate_max",
				ThresholdMax: func() *float64 { v := 160.0; return &v }(),
				Action:       "caution",
			}},
		},
		{
			ID:    "constraint6",
			Code:  "I50",
			Label: "Heart Failure",
			ImpactOnTraining: []domain.ImpactRule{{
				Metric:       "blood_pressure_diastolic",
				ThresholdMax: func() *float64 { v := 85.0; return &v }(),
				Action:       "require_approval",
			}},
		},
	}
	runTestCase(t, "multiple constraints with mixed violations", constraints, nil, &domain.WorkoutPlan{
		MaxHeartRate:           165,
		BloodPressureDiastolic: 90,
	}, []domain.ConstraintViolation{
		{
			ConstraintID: "constraint5",
			Type:         "above_threshold",
			Metric:       "heart_rate_max",
			ActualValue:  165.0,
			Threshold:    160.0,
			Description:  "heart_rate_max exceeds maximum threshold (160.0)",
			Action:       "caution",
		},
		{
			ConstraintID: "constraint6",
			Type:         "above_threshold",
			Metric:       "blood_pressure_diastolic",
			ActualValue:  90.0,
			Threshold:    85.0,
			Description:  "blood_pressure_diastolic exceeds maximum threshold (85.0)",
			Action:       "require_approval",
		},
	}, true)
}

func runTestCase(t *testing.T, name string, constraints []domain.MedicalConstraint, repoError error,
	workout *domain.WorkoutPlan,
	expectedViolations []domain.ConstraintViolation, expectedReview bool) {
	ctx := context.Background()
	userID := testUserID

	mockRepo := &MockMedicalRepository{}
	mockRepo.On("GetActiveConstraints", ctx, userID).Return(constraints, repoError)

	service := NewMedicalService(mockRepo)
	violations, requiresReview := service.EvaluateWorkout(ctx, userID, workout)

	assert.Equal(t, expectedViolations, violations)
	assert.Equal(t, expectedReview, requiresReview)
	mockRepo.AssertExpectations(t)
}

func TestMedicalServiceGetRecommendedModifications(t *testing.T) {
	ctx := context.Background()
	userID := testUserID

	tests := []struct {
		name         string
		constraints  []domain.MedicalConstraint
		repoError    error
		workout      *domain.WorkoutPlan
		expectations []domain.ModificationSuggestion
	}{
		{
			name:         "no constraints",
			constraints:  []domain.MedicalConstraint{},
			repoError:    nil,
			workout:      &domain.WorkoutPlan{MaxHeartRate: 170},
			expectations: nil,
		},
		{
			name:         "repository error",
			constraints:  nil,
			repoError:    errors.New("database error"),
			workout:      &domain.WorkoutPlan{MaxHeartRate: 170},
			expectations: nil,
		},
		{
			name: "high heart rate modification",
			constraints: []domain.MedicalConstraint{
				{
					ID:    "constraint1",
					Code:  "I10",
					Label: "Hypertension",
					ImpactOnTraining: []domain.ImpactRule{
						{
							Metric:       "heart_rate_max",
							ThresholdMax: func() *float64 { v := 160.0; return &v }(),
							Action:       "modify",
						},
					},
				},
			},
			repoError: nil,
			workout: &domain.WorkoutPlan{
				MaxHeartRate: 170, // Above threshold
			},
			expectations: []domain.ModificationSuggestion{
				{
					ConstraintID:   "constraint1",
					Type:           "reduce_intensity",
					Metric:         "heart_rate_max",
					CurrentValue:   170.0,
					SuggestedValue: 160.0,
					Reason:         "Reduce intensity to stay below 160 bpm due to Hypertension",
					Priority:       1,
				},
			},
		},
		{
			name: "high blood pressure modification",
			constraints: []domain.MedicalConstraint{
				{
					ID:    "constraint2",
					Code:  "I10",
					Label: "Hypertension",
					ImpactOnTraining: []domain.ImpactRule{
						{
							Metric:       "blood_pressure_systolic",
							ThresholdMax: func() *float64 { v := 140.0; return &v }(),
							Action:       "modify",
						},
					},
				},
			},
			repoError: nil,
			workout: &domain.WorkoutPlan{
				BloodPressureSystolic: 150, // Above threshold
			},
			expectations: []domain.ModificationSuggestion{
				{
					ConstraintID:   "constraint2",
					Type:           "reduce_intensity",
					Metric:         "blood_pressure_systolic",
					CurrentValue:   150.0,
					SuggestedValue: 140.0,
					Reason:         "High BP detected, target 140 mmHg max",
					Priority:       1,
				},
			},
		},
		{
			name: "no modifications needed",
			constraints: []domain.MedicalConstraint{
				{
					ID:    "constraint3",
					Code:  "I10",
					Label: "Hypertension",
					ImpactOnTraining: []domain.ImpactRule{
						{
							Metric:       "heart_rate_max",
							ThresholdMax: func() *float64 { v := 160.0; return &v }(),
							Action:       "modify",
						},
					},
				},
			},
			repoError: nil,
			workout: &domain.WorkoutPlan{
				MaxHeartRate: 150, // Below threshold
			},
			expectations: nil,
		},
		{
			name: "multiple suggestions",
			constraints: []domain.MedicalConstraint{
				{
					ID:    "constraint4",
					Code:  "I10",
					Label: "Hypertension",
					ImpactOnTraining: []domain.ImpactRule{
						{
							Metric:       "heart_rate_max",
							ThresholdMax: func() *float64 { v := 160.0; return &v }(),
							Action:       "modify",
						},
						{
							Metric:       "blood_pressure_systolic",
							ThresholdMax: func() *float64 { v := 140.0; return &v }(),
							Action:       "modify",
						},
					},
				},
			},
			repoError: nil,
			workout: &domain.WorkoutPlan{
				MaxHeartRate:          170, // Above threshold
				BloodPressureSystolic: 150, // Above threshold
			},
			expectations: []domain.ModificationSuggestion{
				{
					ConstraintID:   "constraint4",
					Type:           "reduce_intensity",
					Metric:         "heart_rate_max",
					CurrentValue:   170.0,
					SuggestedValue: 160.0,
					Reason:         "Reduce intensity to stay below 160 bpm due to Hypertension",
					Priority:       1,
				},
				{
					ConstraintID:   "constraint4",
					Type:           "reduce_intensity",
					Metric:         "blood_pressure_systolic",
					CurrentValue:   150.0,
					SuggestedValue: 140.0,
					Reason:         "High BP detected, target 140 mmHg max",
					Priority:       1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockMedicalRepository{}
			mockRepo.On("GetActiveConstraints", ctx, userID).Return(tt.constraints, tt.repoError)

			service := NewMedicalService(mockRepo)
			suggestions := service.GetRecommendedModifications(ctx, userID, tt.workout)

			assert.Equal(t, tt.expectations, suggestions)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMedicalServiceIsMetricAllowed(t *testing.T) {
	ctx := context.Background()
	userID := testUserID

	tests := []struct {
		name            string
		constraints     []domain.MedicalConstraint
		repoError       error
		metricType      string
		expectedAllowed bool
		expectedReason  string
		expectError     bool
	}{
		{
			name:            "no constraints - metric allowed",
			constraints:     []domain.MedicalConstraint{},
			repoError:       nil,
			metricType:      "heart_rate",
			expectedAllowed: true,
			expectedReason:  "",
			expectError:     false,
		},
		{
			name:            "repository error",
			constraints:     nil,
			repoError:       errors.New("database error"),
			metricType:      "heart_rate",
			expectedAllowed: false,
			expectedReason:  "",
			expectError:     true,
		},
		{
			name: "metric should be avoided",
			constraints: []domain.MedicalConstraint{
				{
					ID:    "constraint1",
					Code:  "I21",
					Label: "Acute myocardial infarction",
					ImpactOnTraining: []domain.ImpactRule{
						{
							Metric: "heart_rate",
							Action: "avoid",
						},
					},
				},
			},
			repoError:       nil,
			metricType:      "heart_rate",
			expectedAllowed: false,
			expectedReason:  "Acute myocardial infarction",
			expectError:     false,
		},
		{
			name: "metric allowed with caution",
			constraints: []domain.MedicalConstraint{
				{
					ID:    "constraint2",
					Code:  "I10",
					Label: "Hypertension",
					ImpactOnTraining: []domain.ImpactRule{
						{
							Metric: "heart_rate",
							Action: "caution",
						},
					},
				},
			},
			repoError:       nil,
			metricType:      "heart_rate",
			expectedAllowed: true,
			expectedReason:  "caution: Hypertension",
			expectError:     false,
		},
		{
			name: "different metric not affected",
			constraints: []domain.MedicalConstraint{
				{
					ID:    "constraint3",
					Code:  "I10",
					Label: "Hypertension",
					ImpactOnTraining: []domain.ImpactRule{
						{
							Metric: "blood_pressure",
							Action: "avoid",
						},
					},
				},
			},
			repoError:       nil,
			metricType:      "heart_rate",
			expectedAllowed: true,
			expectedReason:  "",
			expectError:     false,
		},
		{
			name: "multiple constraints - avoid takes precedence",
			constraints: []domain.MedicalConstraint{
				{
					ID:    "constraint5",
					Code:  "I21",
					Label: "Acute myocardial infarction",
					ImpactOnTraining: []domain.ImpactRule{
						{
							Metric: "heart_rate",
							Action: "avoid",
						},
					},
				},
				{
					ID:    "constraint4",
					Code:  "I10",
					Label: "Hypertension",
					ImpactOnTraining: []domain.ImpactRule{
						{
							Metric: "heart_rate",
							Action: "caution",
						},
					},
				},
			},
			repoError:       nil,
			metricType:      "heart_rate",
			expectedAllowed: false,
			expectedReason:  "Acute myocardial infarction",
			expectError:     false,
		},
		{
			name: "modify action allows metric",
			constraints: []domain.MedicalConstraint{
				{
					ID:    "constraint6",
					Code:  "I50",
					Label: "Heart Failure",
					ImpactOnTraining: []domain.ImpactRule{
						{
							Metric: "heart_rate",
							Action: "modify",
						},
					},
				},
			},
			repoError:       nil,
			metricType:      "heart_rate",
			expectedAllowed: true,
			expectedReason:  "",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockMedicalRepository{}
			mockRepo.On("GetActiveConstraints", ctx, userID).Return(tt.constraints, tt.repoError)

			service := NewMedicalService(mockRepo)
			allowed, reason, err := service.IsMetricAllowed(ctx, userID, tt.metricType)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedAllowed, allowed)
				assert.Equal(t, tt.expectedReason, reason)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// Test private methods by creating a test service and calling them indirectly
func TestCheckRuleAgainstWorkout(t *testing.T) {
	mockRepo := &MockMedicalRepository{}
	service := NewMedicalService(mockRepo)

	constraint := domain.MedicalConstraint{
		ID:    "test-constraint",
		Code:  "I10",
		Label: "Hypertension",
	}

	workout := &domain.WorkoutPlan{
		TargetHeartRate:        140,
		MaxHeartRate:           180,
		BloodPressureSystolic:  120,
		BloodPressureDiastolic: 80,
	}

	// Test heart rate above threshold
	rule := domain.ImpactRule{
		Metric:       "heart_rate_max",
		ThresholdMax: func() *float64 { v := 160.0; return &v }(),
		Action:       "caution",
	}

	violation := service.checkRuleAgainstWorkout(constraint, rule, workout)
	require.NotNil(t, violation)
	assert.Equal(t, "test-constraint", violation.ConstraintID)
	assert.Equal(t, "above_threshold", violation.Type)
	assert.Equal(t, "heart_rate_max", violation.Metric)
	assert.Equal(t, 180.0, violation.ActualValue)
	assert.Equal(t, 160.0, violation.Threshold)

	// Test below threshold
	ruleBelow := domain.ImpactRule{
		Metric:       "heart_rate",
		ThresholdMin: func() *float64 { v := 150.0; return &v }(),
		Action:       "modify",
	}

	violationBelow := service.checkRuleAgainstWorkout(constraint, ruleBelow, workout)
	require.NotNil(t, violationBelow)
	assert.Equal(t, "below_threshold", violationBelow.Type)
	assert.Equal(t, 140.0, violationBelow.ActualValue)
	assert.Equal(t, 150.0, violationBelow.Threshold)

	// Test no violation
	ruleOK := domain.ImpactRule{
		Metric:       "heart_rate_max",
		ThresholdMax: func() *float64 { v := 200.0; return &v }(),
		Action:       "caution",
	}

	violationOK := service.checkRuleAgainstWorkout(constraint, ruleOK, workout)
	assert.Nil(t, violationOK)

	// Test unknown metric
	ruleUnknown := domain.ImpactRule{
		Metric: "unknown_metric",
		Action: "caution",
	}

	violationUnknown := service.checkRuleAgainstWorkout(constraint, ruleUnknown, workout)
	require.NotNil(t, violationUnknown)
	assert.Equal(t, "unknown_metric", violationUnknown.Type)
	assert.Equal(t, "require_approval", violationUnknown.Action)
}

func TestSuggestModification(t *testing.T) {
	mockRepo := &MockMedicalRepository{}
	service := NewMedicalService(mockRepo)

	constraint := domain.MedicalConstraint{
		ID:    "test-constraint",
		Code:  "I10",
		Label: "Hypertension",
	}

	workout := &domain.WorkoutPlan{
		MaxHeartRate:          180,
		BloodPressureSystolic: 150,
	}

	// Test heart rate modification
	ruleHR := domain.ImpactRule{
		Metric:       "heart_rate_max",
		ThresholdMax: func() *float64 { v := 160.0; return &v }(),
		Action:       "modify",
	}

	suggestionHR := service.suggestModification(constraint, ruleHR, workout)
	require.NotNil(t, suggestionHR)
	assert.Equal(t, "test-constraint", suggestionHR.ConstraintID)
	assert.Equal(t, "reduce_intensity", suggestionHR.Type)
	assert.Equal(t, "heart_rate_max", suggestionHR.Metric)
	assert.Equal(t, 180.0, suggestionHR.CurrentValue)
	assert.Equal(t, 160.0, suggestionHR.SuggestedValue)
	assert.Contains(t, suggestionHR.Reason, "Reduce intensity to stay below 160 bpm")

	// Test blood pressure modification
	ruleBP := domain.ImpactRule{
		Metric:       "blood_pressure_systolic",
		ThresholdMax: func() *float64 { v := 140.0; return &v }(),
		Action:       "modify",
	}

	suggestionBP := service.suggestModification(constraint, ruleBP, workout)
	require.NotNil(t, suggestionBP)
	assert.Equal(t, "blood_pressure_systolic", suggestionBP.Metric)
	assert.Equal(t, 150.0, suggestionBP.CurrentValue)
	assert.Equal(t, 140.0, suggestionBP.SuggestedValue)
	assert.Contains(t, suggestionBP.Reason, "High BP detected, target 140 mmHg max")

	// Test no modification needed
	ruleOK := domain.ImpactRule{
		Metric:       "heart_rate_max",
		ThresholdMax: func() *float64 { v := 200.0; return &v }(),
		Action:       "modify",
	}

	suggestionOK := service.suggestModification(constraint, ruleOK, workout)
	assert.Nil(t, suggestionOK)

	// Test unsupported metric
	ruleUnsupported := domain.ImpactRule{
		Metric:       "unsupported_metric",
		ThresholdMax: func() *float64 { v := 100.0; return &v }(),
		Action:       "modify",
	}

	suggestionUnsupported := service.suggestModification(constraint, ruleUnsupported, workout)
	assert.Nil(t, suggestionUnsupported)
}
