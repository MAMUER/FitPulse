// Package postgres provides PostgreSQL repository implementations.

// Package postgres provides PostgreSQL repository implementations.

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
	"github.com/MAMUER/project/internal/repository/shared"
)

type TrainingRepository struct {
	db *sql.DB

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

	if r.stmts["getPlan"], err = r.db.PrepareContext(context.Background(), shared.QueryGetPlan); err != nil {

		panic(err)

	}

	if r.stmts["listPlans"], err = r.db.PrepareContext(context.Background(), shared.QueryListPlans); err != nil {

		panic(err)

	}

	if r.stmts["getProgress"], err = r.db.PrepareContext(context.Background(), shared.QueryGetProgress); err != nil {

		panic(err)

	}

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

	query := shared.QueryGetPlan

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

	query := shared.QueryListPlans

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

	return ScanTrainingPlans(rows)

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

	query := shared.QueryGetProgress

	var totalPlans, completedWorkouts int

	var err error

	if stmt := r.stmts["getProgress"]; stmt != nil {

		err = stmt.QueryRowContext(ctx, userID).Scan(&totalPlans, &completedWorkouts)

	} else {

		err = r.db.QueryRowContext(ctx, query, userID).Scan(&totalPlans, &completedWorkouts)

	}

	if err != nil {

		return nil, apperrors.Internal("failed to get progress", err)

	}

	progress := shared.ScanTrainingProgress(totalPlans, completedWorkouts)

	return map[string]interface{}{

		"total_plans": progress.TotalPlans,

		"completed_workouts": progress.CompletedWorkouts,

		"completion_rate": progress.CompletionRate,
	}, nil

}

func (r *TrainingRepository) GetAchievements(ctx context.Context, userID string) ([]*entity.Achievement, error) {

	query := shared.QueryGetAchievements

	rows, err := r.db.QueryContext(ctx, query, userID)

	if err != nil {

		return nil, apperrors.Internal("failed to get achievements", err)

	}

	defer func() { _ = rows.Close() }()

	achievements, err := shared.ScanAchievements(rows)

	if err != nil {

		return nil, err

	}

	if err := rows.Err(); err != nil {

		return nil, apperrors.Internal("failed to iterate achievements", err)

	}

	return achievements, nil

}
