package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"math"
	"testing"
	"time"

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

func setupTrainingRepoDays(t *testing.T) (*TrainingRepository, sqlmock.Sqlmock) {

	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(trainingDaysConverter{}))

	require.NoError(t, err)

	return &TrainingRepository{db: db}, mock

}

func setupPreparedTrainingRepo(t *testing.T) (*TrainingRepository, sqlmock.Sqlmock) {

	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(trainingDaysConverter{}))

	require.NoError(t, err)

	mock.ExpectPrepare("training_plans WHERE id")
	mock.ExpectPrepare("OVER")
	mock.ExpectPrepare("completed_workouts")

	repo := NewTrainingRepository(db)

	require.NotNil(t, repo)

	trainingRepo, ok := repo.(*TrainingRepository)

	require.True(t, ok)

	require.NotNil(t, trainingRepo.stmts["getPlan"])
	require.NotNil(t, trainingRepo.stmts["listPlans"])
	require.NotNil(t, trainingRepo.stmts["getProgress"])

	return trainingRepo, mock

}

type trainingDaysConverter struct{}

func (trainingDaysConverter) ConvertValue(v interface{}) (driver.Value, error) {

	if days, ok := v.([]int); ok {

		return days, nil

	}

	return driver.DefaultParameterConverter.ConvertValue(v)

}

func TestTrainingRepository_CreatePlan_JSONMarshalError(t *testing.T) {

	repo, mock := setupTrainingRepo(t)

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

	require.NoError(t, mock.ExpectationsWereMet())

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

func TestTrainingRepository_CreatePlan_Success(t *testing.T) {

	repo, mock := setupTrainingRepoDays(t)

	ctx := context.Background()

	now := time.Now()

	planData := map[string]interface{}{"weeks": 4}

	planDataJSON, err := json.Marshal(planData)

	require.NoError(t, err)

	plan := &entity.TrainingPlan{

		ID:             "plan-1",
		UserID:         "user-1",
		Classification: "strength",
		DurationWeeks:  4,
		AvailableDays:  []int{1, 3, 5},
		PlanData:       planData,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	mock.ExpectQuery("INSERT INTO training_plans").
		WithArgs("plan-1", "user-1", "strength", 4, []int{1, 3, 5}, planDataJSON, now, now).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("plan-1"))

	result, err := repo.CreatePlan(ctx, plan)

	require.NoError(t, err)

	require.NotNil(t, result)

	assert.Equal(t, "plan-1", result.ID)

	assert.Equal(t, planData, result.PlanData)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_CreatePlan_QueryError(t *testing.T) {

	repo, mock := setupTrainingRepoDays(t)

	ctx := context.Background()

	mock.ExpectQuery("INSERT INTO training_plans").WillReturnError(assert.AnError)

	result, err := repo.CreatePlan(ctx, &entity.TrainingPlan{ID: "plan-1", UserID: "user-1"})

	require.Error(t, err)

	assert.Nil(t, result)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_GetPlan_WithPlanData(t *testing.T) {

	repo, mock := setupTrainingRepoDays(t)

	ctx := context.Background()

	now := time.Now()

	planDataJSON, err := json.Marshal(map[string]interface{}{"weeks": 4})

	require.NoError(t, err)

	rows := mock.NewRows([]string{"id", "user_id", "classification", "duration_weeks", "available_days", "plan_data", "created_at", "updated_at"}).
		AddRow("plan-1", "user-1", "strength", 4, []int{1, 3, 5}, planDataJSON, now, now)

	mock.ExpectQuery("training_plans WHERE id").
		WithArgs("plan-1", "user-1").
		WillReturnRows(rows)

	result, err := repo.GetPlan(ctx, "user-1", "plan-1")

	require.NoError(t, err)

	require.NotNil(t, result)

	assert.Equal(t, "plan-1", result.ID)

	assert.Equal(t, "strength", result.Classification)

	assert.Equal(t, 4, result.DurationWeeks)

	assert.Equal(t, []int{1, 3, 5}, result.AvailableDays)

	assert.Equal(t, float64(4), result.PlanData["weeks"])

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_GetPlan_EmptyPlanData(t *testing.T) {

	repo, mock := setupTrainingRepoDays(t)

	ctx := context.Background()

	now := time.Now()

	rows := mock.NewRows([]string{"id", "user_id", "classification", "duration_weeks", "available_days", "plan_data", "created_at", "updated_at"}).
		AddRow("plan-1", "user-1", "strength", 4, []int{1, 3, 5}, []byte{}, now, now)

	mock.ExpectQuery("training_plans WHERE id").
		WithArgs("plan-1", "user-1").
		WillReturnRows(rows)

	result, err := repo.GetPlan(ctx, "user-1", "plan-1")

	require.NoError(t, err)

	require.NotNil(t, result)

	assert.Nil(t, result.PlanData)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_GetPlan_UnmarshalError(t *testing.T) {

	repo, mock := setupTrainingRepoDays(t)

	ctx := context.Background()

	now := time.Now()

	rows := mock.NewRows([]string{"id", "user_id", "classification", "duration_weeks", "available_days", "plan_data", "created_at", "updated_at"}).
		AddRow("plan-1", "user-1", "strength", 4, []int{1, 3, 5}, []byte(`{"weeks":`), now, now)

	mock.ExpectQuery("training_plans WHERE id").
		WithArgs("plan-1", "user-1").
		WillReturnRows(rows)

	result, err := repo.GetPlan(ctx, "user-1", "plan-1")

	require.Error(t, err)

	assert.Nil(t, result)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_ListPlans_Success(t *testing.T) {

	repo, mock := setupTrainingRepoDays(t)

	ctx := context.Background()

	now := time.Now()

	planDataJSON, err := json.Marshal(map[string]interface{}{"weeks": 4})

	require.NoError(t, err)

	rows := mock.NewRows([]string{"id", "user_id", "classification", "duration_weeks", "available_days", "plan_data", "created_at", "updated_at", "total_count"}).
		AddRow("plan-1", "user-1", "strength", 4, []int{1, 3, 5}, planDataJSON, now, now, 2)

	mock.ExpectQuery("OVER").
		WithArgs("user-1", 10, 0).
		WillReturnRows(rows)

	result, total, err := repo.ListPlans(ctx, "user-1", 1, 10)

	require.NoError(t, err)

	require.Len(t, result, 1)

	assert.Equal(t, "plan-1", result[0].ID)

	assert.Equal(t, []int{1, 3, 5}, result[0].AvailableDays)

	assert.Equal(t, 2, total)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_ListPlans_Empty(t *testing.T) {

	repo, mock := setupTrainingRepoDays(t)

	ctx := context.Background()

	rows := mock.NewRows([]string{"id", "user_id", "classification", "duration_weeks", "available_days", "plan_data", "created_at", "updated_at", "total_count"})

	mock.ExpectQuery("OVER").
		WithArgs("user-1", 10, 0).
		WillReturnRows(rows)

	result, total, err := repo.ListPlans(ctx, "user-1", 1, 10)

	require.NoError(t, err)

	assert.Empty(t, result)

	assert.Equal(t, 0, total)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_ListPlans_SecondPageOffset(t *testing.T) {

	repo, mock := setupTrainingRepoDays(t)

	ctx := context.Background()

	rows := mock.NewRows([]string{"id", "user_id", "classification", "duration_weeks", "available_days", "plan_data", "created_at", "updated_at", "total_count"})

	mock.ExpectQuery("OVER").
		WithArgs("user-1", 10, 20).
		WillReturnRows(rows)

	result, total, err := repo.ListPlans(ctx, "user-1", 3, 10)

	require.NoError(t, err)

	assert.Empty(t, result)

	assert.Equal(t, 0, total)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_GetProgress_NoPlans(t *testing.T) {

	repo, mock := setupTrainingRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"total_plans", "completed_workouts"}).
		AddRow(0, 0)

	mock.ExpectQuery("SELECT").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetProgress(ctx, "user-1")

	require.NoError(t, err)

	assert.Equal(t, 0, result["total_plans"])

	assert.Equal(t, 0, result["completed_workouts"])

	assert.True(t, math.IsNaN(result["completion_rate"].(float64)))

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_GetProgress_ScanError(t *testing.T) {

	repo, mock := setupTrainingRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"total_plans", "completed_workouts"}).
		AddRow("not-a-number", "not-a-number")

	mock.ExpectQuery("SELECT").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetProgress(ctx, "user-1")

	require.Error(t, err)

	assert.Nil(t, result)

	assert.True(t, apperrors.Code(err) == "INTERNAL")

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_NewTrainingRepository(t *testing.T) {

	_, mock := setupPreparedTrainingRepo(t)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_NewTrainingRepository_PrepareError(t *testing.T) {

	failures := []struct {
		name  string
		index int
	}{
		{"get plan", 0},
		{"list plans", 1},
		{"get progress", 2},
	}

	for _, failure := range failures {

		t.Run(failure.name, func(t *testing.T) {

			db, mock, err := sqlmock.New()

			require.NoError(t, err)

			prepareQueries := []string{"training_plans WHERE id", "OVER", "completed_workouts"}

			for i, prepareQuery := range prepareQueries {

				expected := mock.ExpectPrepare(prepareQuery)

				if i == failure.index {

					expected.WillReturnError(assert.AnError)

				}

			}

			assert.Panics(t, func() {

				_ = NewTrainingRepository(db)

			})

		})

	}

}

func TestTrainingRepository_PreparedStatements_GetPlan(t *testing.T) {

	repo, mock := setupPreparedTrainingRepo(t)

	ctx := context.Background()

	now := time.Now()

	planDataJSON, err := json.Marshal(map[string]interface{}{"weeks": 4})

	require.NoError(t, err)

	rows := mock.NewRows([]string{"id", "user_id", "classification", "duration_weeks", "available_days", "plan_data", "created_at", "updated_at"}).
		AddRow("plan-1", "user-1", "strength", 4, []int{1, 3, 5}, planDataJSON, now, now)

	mock.ExpectQuery("training_plans WHERE id").
		WithArgs("plan-1", "user-1").
		WillReturnRows(rows)

	result, err := repo.GetPlan(ctx, "user-1", "plan-1")

	require.NoError(t, err)

	require.NotNil(t, result)

	assert.Equal(t, "plan-1", result.ID)

	assert.Equal(t, float64(4), result.PlanData["weeks"])

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_PreparedStatements_ListPlans(t *testing.T) {

	repo, mock := setupPreparedTrainingRepo(t)

	ctx := context.Background()

	planDataJSON, err := json.Marshal(map[string]interface{}{"weeks": 4})

	require.NoError(t, err)

	rows := mock.NewRows([]string{"id", "user_id", "classification", "duration_weeks", "available_days", "plan_data", "created_at", "updated_at", "total_count"}).
		AddRow("plan-1", "user-1", "strength", 4, []int{1, 3, 5}, planDataJSON, time.Now(), time.Now(), 1)

	mock.ExpectQuery("OVER").
		WithArgs("user-1", 10, 0).
		WillReturnRows(rows)

	result, total, err := repo.ListPlans(ctx, "user-1", 1, 10)

	require.NoError(t, err)

	require.Len(t, result, 1)

	assert.Equal(t, 1, total)

	require.NoError(t, mock.ExpectationsWereMet())

}

func TestTrainingRepository_PreparedStatements_GetProgress(t *testing.T) {

	repo, mock := setupPreparedTrainingRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"total_plans", "completed_workouts"}).
		AddRow(4, 3)

	mock.ExpectQuery("completed_workouts").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetProgress(ctx, "user-1")

	require.NoError(t, err)

	assert.Equal(t, 4, result["total_plans"])

	assert.Equal(t, 3, result["completed_workouts"])

	assert.Equal(t, 75.0, result["completion_rate"])

	require.NoError(t, mock.ExpectationsWereMet())

}
