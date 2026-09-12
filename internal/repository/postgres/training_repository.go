package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
)

type TrainingRepository struct {
	db    *sql.DB
	stmts map[string]*sql.Stmt
}

func NewTrainingRepository(db *sql.DB) port.TrainingRepository {
	repo := &TrainingRepository{db: db}
	repo.prepareStatements()
	return repo
}

func (r *TrainingRepository) prepareStatements() {
	r.stmts = map[string]*sql.Stmt{}
	var err error
	if r.stmts["getPlan"], err = r.db.PrepareContext(context.Background(), `SELECT id, user_id, classification, duration_weeks, available_days, plan_data, created_at, updated_at FROM training_plans WHERE id = $1 AND user_id = $2`); err != nil {
		panic(err)
	}
	if r.stmts["listPlans"], err = r.db.PrepareContext(context.Background(), `SELECT id, user_id, classification, duration_weeks, available_days, plan_data, created_at, updated_at, COUNT(*) OVER() AS total_count FROM training_plans WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`); err != nil {
		panic(err)
	}
	if r.stmts["getProgress"], err = r.db.PrepareContext(context.Background(), `SELECT COUNT(*) as total_plans, COUNT(CASE WHEN updated_at > created_at THEN 1 END) as completed_workouts FROM training_plans WHERE user_id = $1`); err != nil {
		panic(err)
	}
}

func scanAchievements(rows *sql.Rows) ([]*entity.Achievement, error) {
	return scanSlice(rows, func(a *entity.Achievement) error {
		return rows.Scan(&a.ID, &a.UserID, &a.Type, &a.Title, &a.Description, &a.EarnedAt)
	})
}

func scanSlice[T any](rows *sql.Rows, scanFunc func(*T) error) ([]*T, error) {
	var items []*T
	for rows.Next() {
		item := new(T)
		if err := scanFunc(item); err != nil {
			return nil, apperrors.Internal("failed to scan item", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Internal("failed to iterate items", err)
	}
	return items, nil
}

func (r *TrainingRepository) CreatePlan(ctx context.Context, plan *entity.TrainingPlan) (*entity.TrainingPlan, error) {
	planDataJSON, err := json.Marshal(plan.PlanData)
	if err != nil {
		return nil, apperrors.Internal("failed to marshal plan data", err)
	}

	query := `
		INSERT INTO training_plans (id, user_id, classification, duration_weeks, available_days, plan_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	err = r.db.QueryRowContext(ctx, query,
		plan.ID, plan.UserID, plan.Classification, plan.DurationWeeks, plan.AvailableDays,
		planDataJSON, plan.CreatedAt, plan.UpdatedAt,
	).Scan(&plan.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to create training plan", err)
	}
	return plan, nil
}

func (r *TrainingRepository) GetPlan(ctx context.Context, userID, planID string) (*entity.TrainingPlan, error) {
	plan := &entity.TrainingPlan{}
	var planDataJSON []byte

	query := `SELECT id, user_id, classification, duration_weeks, available_days, plan_data, created_at, updated_at FROM training_plans WHERE id = $1 AND user_id = $2`
	var err error
	if stmt := r.stmts["getPlan"]; stmt != nil {
		err = stmt.QueryRowContext(ctx, planID, userID).Scan(
			&plan.ID, &plan.UserID, &plan.Classification, &plan.DurationWeeks,
			&plan.AvailableDays, &planDataJSON, &plan.CreatedAt, &plan.UpdatedAt,
		)
	} else {
		err = r.db.QueryRowContext(ctx, query, planID, userID).Scan(
			&plan.ID, &plan.UserID, &plan.Classification, &plan.DurationWeeks,
			&plan.AvailableDays, &planDataJSON, &plan.CreatedAt, &plan.UpdatedAt,
		)
	}
	if err != nil {
		if err == sql.ErrNoRows {
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

func (r *TrainingRepository) ListPlans(ctx context.Context, userID string, page, pageSize int) ([]*entity.TrainingPlan, int, error) {
	offset := (page - 1) * pageSize

	query := `SELECT id, user_id, classification, duration_weeks, available_days, plan_data, created_at, updated_at, COUNT(*) OVER() AS total_count FROM training_plans WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	var rows *sql.Rows
	var err error
	if stmt := r.stmts["listPlans"]; stmt != nil {
		rows, err = stmt.QueryContext(ctx, userID, pageSize, offset)
	} else {
		rows, err = r.db.QueryContext(ctx, query, userID, pageSize, offset)
	}
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list training plans", err)
	}
	defer func() { _ = rows.Close() }()

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

func (r *TrainingRepository) CompleteWorkout(ctx context.Context, userID, planID string) error {
	query := `
		UPDATE training_plans SET updated_at = $1 WHERE id = $2 AND user_id = $3
	`
	_, err := r.db.ExecContext(ctx, query, time.Now(), planID, userID)
	if err != nil {
		return apperrors.Internal("failed to complete workout", err)
	}
	return nil
}

func (r *TrainingRepository) GetProgress(ctx context.Context, userID string) (map[string]interface{}, error) {
	var totalPlans, completedWorkouts int
	query := `SELECT COUNT(*) as total_plans, COUNT(CASE WHEN updated_at > created_at THEN 1 END) as completed_workouts FROM training_plans WHERE user_id = $1`
	var err error
	if stmt := r.stmts["getProgress"]; stmt != nil {
		err = stmt.QueryRowContext(ctx, userID).Scan(&totalPlans, &completedWorkouts)
	} else {
		err = r.db.QueryRowContext(ctx, query, userID).Scan(&totalPlans, &completedWorkouts)
	}
	if err != nil {
		return nil, apperrors.Internal("failed to get progress", err)
	}

	return map[string]interface{}{
		"total_plans":        totalPlans,
		"completed_workouts": completedWorkouts,
		"completion_rate":    float64(completedWorkouts) / float64(totalPlans) * 100,
	}, nil
}

func (r *TrainingRepository) GetAchievements(ctx context.Context, userID string) ([]*entity.Achievement, error) {
	query := `
		SELECT id, user_id, type, title, description, earned_at
		FROM achievements WHERE user_id = $1 ORDER BY earned_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, apperrors.Internal("failed to get achievements", err)
	}
	defer func() { _ = rows.Close() }()

	return scanAchievements(rows)
}
