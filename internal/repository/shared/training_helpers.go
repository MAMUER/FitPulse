// Package shared contains reusable training scanner helpers.

package shared

import (
	"database/sql"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
)

// ScanTrainingPlans scans database/sql rows into training plans with plan data unmarshaling.

func ScanTrainingPlans(rows *sql.Rows) ([]*entity.TrainingPlan, int, error) {

	return scanTrainingPlanRows(rows)

}

// ScanTrainingPlansPGX scans pgx rows into training plans with plan data unmarshaling.

func ScanTrainingPlansPGX(rows pgx.Rows) ([]*entity.TrainingPlan, int, error) {

	return scanTrainingPlanRows(rows)

}

type trainingPlanScanner interface {
	Next() bool

	Scan(dest ...interface{}) error

	Err() error
}

func scanTrainingPlanRows(rows trainingPlanScanner) ([]*entity.TrainingPlan, int, error) {

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

// ScanAchievementsPGX scans pgx rows into achievements.

func ScanAchievementsPGX(rows pgx.Rows) ([]*entity.Achievement, error) {

	return scanAchievementRows(rows)

}

// ScanAchievements scans database/sql rows into achievements.

func ScanAchievements(rows *sql.Rows) ([]*entity.Achievement, error) {

	return scanAchievementRows(rows)

}

type achievementScanner interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}

func scanAchievementRows(rows achievementScanner) ([]*entity.Achievement, error) {
	var achievements []*entity.Achievement
	for rows.Next() {
		a := &entity.Achievement{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.Type, &a.Title, &a.Description, &a.EarnedAt); err != nil {
			return nil, apperrors.Internal("failed to scan achievement", err)
		}
		achievements = append(achievements, a)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Internal("failed to iterate achievements", err)
	}
	return achievements, nil
}

// TrainingProgress holds training progress metrics.

type TrainingProgress struct {
	TotalPlans int `json:"total_plans"`

	CompletedWorkouts int `json:"completed_workouts"`

	CompletionRate float64 `json:"completion_rate"`
}

// ScanTrainingProgress scans progress query results.

func ScanTrainingProgress(totalPlans, completedWorkouts int) TrainingProgress {

	return TrainingProgress{

		TotalPlans: totalPlans,

		CompletedWorkouts: completedWorkouts,

		CompletionRate: float64(completedWorkouts) / float64(totalPlans) * 100,
	}

}
