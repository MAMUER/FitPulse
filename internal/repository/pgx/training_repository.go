package pgx

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/repository/shared"
)

// TrainingRepositoryPGX implements training operations using pgxpool.Pool.
type TrainingRepositoryPGX struct {
	db DB
}

func NewTrainingRepositoryPGX(db DB) *TrainingRepositoryPGX {
	return &TrainingRepositoryPGX{db: db}
}

func (r *TrainingRepositoryPGX) CreatePlan(ctx context.Context, plan *entity.TrainingPlan) (*entity.TrainingPlan, error) {
	planDataJSON, err := json.Marshal(plan.PlanData)
	if err != nil {
		return nil, apperrors.Internal("failed to marshal plan data", err)
	}

	query := `
		INSERT INTO training_plans (id, user_id, classification, duration_weeks, available_days, plan_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	err = r.db.QueryRow(ctx, query,
		plan.ID, plan.UserID, plan.Classification, plan.DurationWeeks, plan.AvailableDays,
		planDataJSON, plan.CreatedAt, plan.UpdatedAt,
	).Scan(&plan.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to create training plan", err)
	}
	return plan, nil
}

func (r *TrainingRepositoryPGX) GetPlan(ctx context.Context, userID, planID string) (*entity.TrainingPlan, error) {
	plan := &entity.TrainingPlan{}
	var planDataJSON []byte

	err := r.db.QueryRow(ctx, shared.QueryGetPlan, planID, userID).Scan(
		&plan.ID, &plan.UserID, &plan.Classification, &plan.DurationWeeks,
		&plan.AvailableDays, &planDataJSON, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("training plan not found")
		}
		return nil, apperrors.Internal("failed to get training plan", err)
	}

	if len(planDataJSON) > 0 {
		if err := json.Unmarshal(planDataJSON, &plan.PlanData); err != nil {
			return nil, apperrors.Internal("failed to unmarshal plan data", err)
		}
	}

	return plan, nil
}

func (r *TrainingRepositoryPGX) ListPlans(ctx context.Context, userID string, page, pageSize int) ([]*entity.TrainingPlan, int, error) {
	offset := (page - 1) * pageSize

	query := shared.QueryListPlans
	rows, err := r.db.Query(ctx, query, userID, pageSize, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list training plans", err)
	}
	defer rows.Close()

	plans, totalCount, err := shared.ScanTrainingPlansPGX(rows)
	if err != nil {
		return nil, 0, err
	}
	return plans, totalCount, nil
}

func (r *TrainingRepositoryPGX) CompleteWorkout(ctx context.Context, userID, planID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE training_plans SET updated_at = $1 WHERE id = $2 AND user_id = $3
	`, time.Now(), planID, userID)
	if err != nil {
		return apperrors.Internal("failed to complete workout", err)
	}
	return nil
}

func (r *TrainingRepositoryPGX) GetProgress(ctx context.Context, userID string) (map[string]interface{}, error) {
	var totalPlans, completedWorkouts int
	err := r.db.QueryRow(ctx, shared.QueryGetProgress, userID).Scan(&totalPlans, &completedWorkouts)
	if err != nil {
		return nil, apperrors.Internal("failed to get progress", err)
	}

	progress := shared.ScanTrainingProgress(totalPlans, completedWorkouts)
	return map[string]interface{}{
		"total_plans":        progress.TotalPlans,
		"completed_workouts": progress.CompletedWorkouts,
		"completion_rate":    progress.CompletionRate,
	}, nil
}

func (r *TrainingRepositoryPGX) GetAchievements(ctx context.Context, userID string) ([]*entity.Achievement, error) {
	rows, err := r.db.Query(ctx, shared.QueryGetAchievements, userID)
	if err != nil {
		return nil, apperrors.Internal("failed to get achievements", err)
	}
	defer rows.Close()

	return shared.ScanAchievementsPGX(rows)
}

func (r *TrainingRepositoryPGX) CreateAchievement(ctx context.Context, achievement *entity.Achievement) (*entity.Achievement, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO achievements (id, user_id, type, title, description, earned_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, achievement.ID, achievement.UserID, achievement.Type, achievement.Title, achievement.Description, achievement.EarnedAt).Scan(&achievement.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to create achievement", err)
	}
	return achievement, nil
}

func (r *TrainingRepositoryPGX) DeletePlan(ctx context.Context, userID, planID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM training_plans WHERE id = $1 AND user_id = $2`, planID, userID)
	if err != nil {
		return apperrors.Internal("failed to delete plan", err)
	}
	return nil
}

func (r *TrainingRepositoryPGX) UpdatePlan(ctx context.Context, plan *entity.TrainingPlan) (*entity.TrainingPlan, error) {
	err := r.db.QueryRow(ctx, `
		UPDATE training_plans SET classification = $1, duration_weeks = $2, available_days = $3, updated_at = $4
		WHERE id = $5 AND user_id = $6
		RETURNING id, user_id, classification, duration_weeks, available_days, created_at, updated_at
	`, plan.Classification, plan.DurationWeeks, plan.AvailableDays, time.Now(),
		plan.ID, plan.UserID,
	).Scan(&plan.ID, &plan.UserID, &plan.Classification, &plan.DurationWeeks, &plan.AvailableDays, &plan.CreatedAt, &plan.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("training plan not found")
		}
		return nil, apperrors.Internal("failed to update training plan", err)
	}
	return plan, nil
}
