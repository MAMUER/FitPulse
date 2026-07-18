// Package medical provides domain services for medical constraints and training safety.
package medical

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/biometric/domain"
	"github.com/MAMUER/project/internal/logger"
)

// MedicalService manages medical constraints and validates training plans.
type MedicalService struct {
	repo MedicalRepository
	log  *logger.Logger
}

// NewMedicalService creates a new medical service.
func NewMedicalService(repo MedicalRepository, log *logger.Logger) *MedicalService {
	if log == nil {
		log = logger.New("medical-service")
	}
	return &MedicalService{repo: repo, log: log}
}

// EvaluateWorkout checks if a proposed workout respects all medical constraints.
// Returns list of violations and a flag indicating if manual review is needed.
func (s *MedicalService) EvaluateWorkout(ctx context.Context, userID string, workout *domain.WorkoutPlan) (violations []domain.ConstraintViolation, requiresReview bool) {
	constraints, err := s.repo.GetActiveConstraints(ctx, userID)
	if err != nil || len(constraints) == 0 {
		if err != nil {
			s.log.Error("Failed to get active constraints",
				zap.String("user_id", userID),
				zap.Error(err),
			)
		}
		return nil, false
	}

	s.log.Debug("Evaluating workout against constraints",
		zap.String("user_id", userID),
		zap.Int("constraints_count", len(constraints)),
		zap.String("workout_type", workout.Type),
	)

	for _, constraint := range constraints {
		for _, rule := range constraint.ImpactOnTraining {
			violation := s.checkRuleAgainstWorkout(constraint, rule, workout)
			if violation != nil {
				violations = append(violations, *violation)
				if rule.Action == "require_approval" || violation.Action == "require_approval" {
					requiresReview = true
				}
				s.log.Warn("Medical constraint violation detected",
					zap.String("user_id", userID),
					zap.String("constraint_id", constraint.ID),
					zap.String("constraint_code", constraint.Code),
					zap.String("metric", rule.Metric),
					zap.String("action", violation.Action),
					zap.String("type", violation.Type),
				)
			}
		}
	}

	s.log.Debug("Workout evaluation completed",
		zap.String("user_id", userID),
		zap.Int("violations_count", len(violations)),
		zap.Bool("requires_review", requiresReview),
	)

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
		s.log.Error("Failed to get constraints for modifications",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil
	}

	s.log.Debug("Computing modification suggestions",
		zap.String("user_id", userID),
		zap.Int("constraints_count", len(constraints)),
	)

	var suggestions []domain.ModificationSuggestion
	for _, constraint := range constraints {
		for _, rule := range constraint.ImpactOnTraining {
			suggestion := s.suggestModification(constraint, rule, workout)
			if suggestion != nil {
				suggestions = append(suggestions, *suggestion)
				s.log.Info("Workout modification suggested",
					zap.String("user_id", userID),
					zap.String("constraint_id", constraint.ID),
					zap.String("metric", rule.Metric),
				)
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

// IsMetricAllowed checks whether a metric is allowed for a user based on active medical constraints.
func (s *MedicalService) IsMetricAllowed(ctx context.Context, userID, metricType string) (bool, string, error) {
	constraints, err := s.repo.GetActiveConstraints(ctx, userID)
	if err != nil {
		s.log.Error("Failed to get constraints for metric check",
			zap.String("user_id", userID),
			zap.String("metric_type", metricType),
			zap.Error(err),
		)
		return false, "", fmt.Errorf("get active constraints: %w", err)
	}

	s.log.Debug("Checking if metric is allowed",
		zap.String("user_id", userID),
		zap.String("metric_type", metricType),
		zap.Int("constraints_count", len(constraints)),
	)

	for _, constraint := range constraints {
		for _, rule := range constraint.ImpactOnTraining {
			if rule.Metric == metricType {
				if rule.Action == "avoid" {
					s.log.Warn("Metric is not allowed due to medical constraint",
						zap.String("user_id", userID),
						zap.String("metric_type", metricType),
						zap.String("constraint_id", constraint.ID),
						zap.String("constraint_label", constraint.Label),
						zap.String("action", rule.Action),
					)
					return false, "avoid: " + constraint.Label, nil
				}
				if rule.Action == "caution" {
					s.log.Info("Metric allowed with caution",
						zap.String("user_id", userID),
						zap.String("metric_type", metricType),
						zap.String("constraint_id", constraint.ID),
						zap.String("constraint_label", constraint.Label),
					)
					return true, "caution: " + constraint.Label, nil
				}
			}
		}
	}

	s.log.Debug("Metric is allowed",
		zap.String("user_id", userID),
		zap.String("metric_type", metricType),
	)

	return true, "", nil
}

// MedicalRepository defines persistence operations for medical constraints.
type MedicalRepository interface {
	GetActiveConstraints(ctx context.Context, userID string) ([]domain.MedicalConstraint, error)
	SaveConstraint(ctx context.Context, constraint domain.MedicalConstraint) error
	DeleteConstraint(ctx context.Context, constraintID string) error
	GetConstraintByCode(ctx context.Context, code string) ([]domain.MedicalConstraint, error)
}
