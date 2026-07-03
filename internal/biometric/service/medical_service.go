// Package medical provides domain services for medical constraints and training safety.
package medical

import (
	"context"
	"fmt"

	"github.com/MAMUER/project/internal/biometric/domain"
)

// MedicalService manages medical constraints and validates training plans.
type MedicalService struct {
	repo MedicalRepository
}

// NewMedicalService creates a new medical service.
func NewMedicalService(repo MedicalRepository) *MedicalService {
	return &MedicalService{repo: repo}
}

// EvaluateWorkout checks if a proposed workout respects all medical constraints.
// Returns list of violations and a flag indicating if manual review is needed.
func (s *MedicalService) EvaluateWorkout(ctx context.Context, userID string, workout *domain.WorkoutPlan) (violations []domain.ConstraintViolation, requiresReview bool) {
	constraints, err := s.repo.GetActiveConstraints(ctx, userID)
	if err != nil || len(constraints) == 0 {
		return nil, false
	}
	for _, constraint := range constraints {
		for _, rule := range constraint.ImpactOnTraining {
			violation := s.checkRuleAgainstWorkout(constraint, rule, workout)
			if violation != nil {
				violations = append(violations, *violation)
				if rule.Action == "require_approval" || violation.Action == "require_approval" {
					requiresReview = true
				}
			}
		}
	}
	return violations, requiresReview
}

func (s *MedicalService) checkRuleAgainstWorkout(constraint domain.MedicalConstraint, rule domain.ImpactRule, workout *domain.WorkoutPlan) *domain.ConstraintViolation {
	var actual float64
	switch rule.Metric {
	case "heart_rate":
		actual = float64(workout.TargetHeartRate)
	case "heart_rate_max":
		actual = float64(workout.MaxHeartRate)
	case "blood_pressure_systolic":
		actual = float64(workout.BloodPressureSystolic)
	case "blood_pressure_diastolic":
		actual = float64(workout.BloodPressureDiastolic)
	default:
		return &domain.ConstraintViolation{
			ConstraintID: constraint.ID,
			Type:         "unknown_metric",
			Description:  fmt.Sprintf("Metric '%s' not recognized, requires manual review", rule.Metric),
			Action:       "require_approval",
		}
	}
	if rule.ThresholdMin != nil && actual < *rule.ThresholdMin {
		return &domain.ConstraintViolation{
			ConstraintID: constraint.ID,
			Type:         "below_threshold",
			Metric:       rule.Metric,
			ActualValue:  actual,
			Threshold:    *rule.ThresholdMin,
			Description:  fmt.Sprintf("%s below minimum threshold (%.1f)", rule.Metric, *rule.ThresholdMin),
			Action:       rule.Action,
		}
	}
	if rule.ThresholdMax != nil && actual > *rule.ThresholdMax {
		return &domain.ConstraintViolation{
			ConstraintID: constraint.ID,
			Type:         "above_threshold",
			Metric:       rule.Metric,
			ActualValue:  actual,
			Threshold:    *rule.ThresholdMax,
			Description:  fmt.Sprintf("%s exceeds maximum threshold (%.1f)", rule.Metric, *rule.ThresholdMax),
			Action:       rule.Action,
		}
	}
	return nil
}

// GetRecommendedModifications suggests workout modifications based on constraints.
// Returns a modified workout plan or suggestions.
func (s *MedicalService) GetRecommendedModifications(ctx context.Context, userID string, workout *domain.WorkoutPlan) []domain.ModificationSuggestion {
	constraints, err := s.repo.GetActiveConstraints(ctx, userID)
	if err != nil {
		return nil
	}
	var suggestions []domain.ModificationSuggestion
	for _, constraint := range constraints {
		for _, rule := range constraint.ImpactOnTraining {
			suggestion := s.suggestModification(constraint, rule, workout)
			if suggestion != nil {
				suggestions = append(suggestions, *suggestion)
			}
		}
	}
	return suggestions
}

func (s *MedicalService) suggestModification(constraint domain.MedicalConstraint, rule domain.ImpactRule, workout *domain.WorkoutPlan) *domain.ModificationSuggestion {
	switch rule.Metric {
	case "heart_rate_max":
		if float64(workout.MaxHeartRate) > *rule.ThresholdMax {
			return &domain.ModificationSuggestion{
				ConstraintID:   constraint.ID,
				Type:           "reduce_intensity",
				Metric:         "heart_rate_max",
				CurrentValue:   float64(workout.MaxHeartRate),
				SuggestedValue: *rule.ThresholdMax,
				Reason:         fmt.Sprintf("Reduce intensity to stay below %d bpm due to %s", int(*rule.ThresholdMax), constraint.Label),
				Priority:       1,
			}
		}
	case "blood_pressure_systolic":
		if float64(workout.BloodPressureSystolic) > *rule.ThresholdMax {
			return &domain.ModificationSuggestion{
				ConstraintID:   constraint.ID,
				Type:           "reduce_intensity",
				Metric:         "blood_pressure_systolic",
				CurrentValue:   float64(workout.BloodPressureSystolic),
				SuggestedValue: *rule.ThresholdMax,
				Reason:         fmt.Sprintf("High BP detected, target %d mmHg max", int(*rule.ThresholdMax)),
				Priority:       1,
			}
		}
	}
	return nil
}

// IsMetricAllowed checks if a metric is safe to monitor/use for this user.
func (s *MedicalService) IsMetricAllowed(ctx context.Context, userID, metricType string) (bool, string, error) {
	constraints, err := s.repo.GetActiveConstraints(ctx, userID)
	if err != nil {
		return false, "", fmt.Errorf("get active constraints: %w", err)
	}
	for _, constraint := range constraints {
		for _, rule := range constraint.ImpactOnTraining {
			if rule.Metric == metricType {
				if rule.Action == "avoid" {
					return false, constraint.Label, nil
				}
				if rule.Action == "caution" {
					return true, fmt.Sprintf("caution: %s", constraint.Label), nil
				}
			}
		}
	}
	return true, "", nil
}

// MedicalRepository defines persistence operations for medical constraints.
type MedicalRepository interface {
	GetActiveConstraints(ctx context.Context, userID string) ([]domain.MedicalConstraint, error)
	SaveConstraint(ctx context.Context, constraint domain.MedicalConstraint) error
	DeleteConstraint(ctx context.Context, constraintID string) error
	GetConstraintByCode(ctx context.Context, code string) ([]domain.MedicalConstraint, error)
}
