package pgx

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
)

// TrainingRepositoryPGX implements training operations using pgxpool.Pool.
type TrainingRepositoryPGX struct {
	db DB
}

func scanAchievements(rows pgx.Rows) ([]*entity.Achievement, error) {
	return scanSlice(rows, func(a *entity.Achievement) error {
		return rows.Scan(&a.ID, &a.UserID, &a.Type, &a.Title, &a.Description, &a.EarnedAt)
	})
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
	query := `
		SELECT id, user_id, classification, duration_weeks, available_days, plan_data, created_at, updated_at
		FROM training_plans WHERE id = $1 AND user_id = $2
	`
	plan := &entity.TrainingPlan{}
	var planDataJSON []byte

	err := r.db.QueryRow(ctx, query, planID, userID).Scan(
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

	query := `
		SELECT id, user_id, classification, duration_weeks, available_days, plan_data, created_at, updated_at,
		       COUNT(*) OVER() AS total_count
		FROM training_plans WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, pageSize, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list training plans", err)
	}
	defer rows.Close()

	var plans []*entity.TrainingPlan
	var totalCount int
	for rows.Next() {
		plan := &entity.TrainingPlan{}
		var planDataJSON []byte
		if err := rows.Scan(
			&plan.ID, &plan.UserID, &plan.Classification, &plan.DurationWeeks,
			&plan.AvailableDays, &planDataJSON, &plan.CreatedAt, &plan.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, apperrors.Internal("failed to scan training plan", err)
		}
		if len(planDataJSON) > 0 {
			if err := json.Unmarshal(planDataJSON, &plan.PlanData); err != nil {
				return nil, 0, apperrors.Internal("failed to unmarshal plan data", err)
			}
		}
		plans = append(plans, plan)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.Internal("failed to iterate training plans", err)
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
	err := r.db.QueryRow(ctx, `
		SELECT 
			COUNT(*) as total_plans,
			COUNT(CASE WHEN updated_at > created_at THEN 1 END) as completed_workouts
		FROM training_plans WHERE user_id = $1
	`, userID).Scan(&totalPlans, &completedWorkouts)
	if err != nil {
		return nil, apperrors.Internal("failed to get progress", err)
	}

	return map[string]interface{}{
		"total_plans":        totalPlans,
		"completed_workouts": completedWorkouts,
		"completion_rate":    float64(completedWorkouts) / float64(totalPlans) * 100,
	}, nil
}

func (r *TrainingRepositoryPGX) GetAchievements(ctx context.Context, userID string) ([]*entity.Achievement, error) {
	query := `
		SELECT id, user_id, type, title, description, earned_at
		FROM achievements WHERE user_id = $1 ORDER BY earned_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, apperrors.Internal("failed to get achievements", err)
	}
	defer rows.Close()

	return scanAchievements(rows)
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
