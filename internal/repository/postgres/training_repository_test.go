package postgres

import (
	"context"
	"testing"
	"time"

	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
)

func setupTrainingRepo(t *testing.T) (*TrainingRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return &TrainingRepository{db: db}, mock
}

func TestTrainingRepository_CreatePlan_JSONMarshalError(t *testing.T) {
	repo, _ := setupTrainingRepo(t)
	ctx := context.Background()

	planData := map[string]interface{}{}
	planData["invalid"] = make(chan int)
	plan := &entity.TrainingPlan{
		PlanData: planData,
	}

	result, err := repo.CreatePlan(ctx, plan)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepository_GetPlan_NotFound(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").
		WithArgs("plan-1", "user-1").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetPlan(ctx, "user-1", "plan-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrainingRepository_GetPlan_QueryError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").WillReturnError(assert.AnError)

	result, err := repo.GetPlan(ctx, "user-1", "plan-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrainingRepository_ListPlans_QueryError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").WillReturnError(assert.AnError)

	result, total, err := repo.ListPlans(ctx, "user-1", 1, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrainingRepository_ListPlans_ScanError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "classification", "duration_weeks", "available_days", "plan_data", "created_at", "updated_at", "total_count"}).
		AddRow(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	mock.ExpectQuery("SELECT id").
		WithArgs("user-1", 10, 0).
		WillReturnRows(rows)

	result, total, err := repo.ListPlans(ctx, "user-1", 1, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrainingRepository_ListPlans_CountError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "classification", "duration_weeks", "available_days", "plan_data", "created_at", "updated_at", "total_count"}).
		AddRow("plan-1", "user-1", "strength", 4, nil, []byte{}, time.Now(), time.Now(), 1)
	mock.ExpectQuery("SELECT id").
		WithArgs("user-1", 10, 0).
		WillReturnRows(rows)

	result, total, err := repo.ListPlans(ctx, "user-1", 1, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrainingRepository_CompleteWorkout_Success(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE training_plans").
		WithArgs(sqlmock.AnyArg(), "plan-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CompleteWorkout(ctx, "user-1", "plan-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrainingRepository_CompleteWorkout_Error(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE training_plans").WillReturnError(assert.AnError)

	err := repo.CompleteWorkout(ctx, "user-1", "plan-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrainingRepository_GetProgress_Success(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"total_plans", "completed_workouts"}).
		AddRow(10, 5)

	mock.ExpectQuery("SELECT").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetProgress(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, 10, result["total_plans"])
	assert.Equal(t, 5, result["completed_workouts"])
	assert.Equal(t, 50.0, result["completion_rate"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrainingRepository_GetProgress_QueryError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT").WillReturnError(assert.AnError)

	result, err := repo.GetProgress(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrainingRepository_GetAchievements_Success(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "title", "description", "earned_at"}).
		AddRow("ach-1", "user-1", "strength", "First Lift", "Lift 100kg", now)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetAchievements(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "ach-1", result[0].ID)
	assert.Equal(t, "First Lift", result[0].Title)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrainingRepository_GetAchievements_Empty(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "title", "description", "earned_at"})
	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetAchievements(ctx, "user-1")
	require.NoError(t, err)
	assert.Empty(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrainingRepository_GetAchievements_QueryError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").WillReturnError(assert.AnError)

	result, err := repo.GetAchievements(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrainingRepository_GetAchievements_ScanError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "title", "description", "earned_at"}).
		AddRow(nil, nil, nil, nil, nil, nil)
	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetAchievements(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}
