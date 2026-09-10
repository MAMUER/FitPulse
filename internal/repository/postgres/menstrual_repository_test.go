package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

func setupMenstrualRepo(t *testing.T) (*userMenstrualRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewUserMenstrualRepository(db).(*userMenstrualRepository), mock
}

func TestUserMenstrualRepository_ListCycles_Success(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "cycle_start_date", "cycle_end_date", "flow_intensity", "notes", "created_at", "updated_at"}).
		AddRow("cycle-1", "user-1", "2024-01-01", "2024-01-05", "medium", "Mild cramps", now, now).
		AddRow("cycle-2", "user-1", "2024-01-29", "", "", "", now, now)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.ListCycles(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "cycle-1", result[0].ID)
	assert.Equal(t, "2024-01-05", result[0].CycleEndDate)
	assert.Equal(t, "Mild cramps", result[0].Notes)
	assert.Equal(t, "cycle-2", result[1].ID)
	assert.Empty(t, result[1].CycleEndDate)
	assert.Empty(t, result[1].Notes)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_ListCycles_Empty(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "cycle_start_date", "cycle_end_date", "flow_intensity", "notes", "created_at", "updated_at"})

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.ListCycles(ctx, "user-1")
	require.NoError(t, err)
	assert.Empty(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_ListCycles_QueryError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").WillReturnError(assert.AnError)

	result, err := repo.ListCycles(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_ListCycles_ScanError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "cycle_start_date", "cycle_end_date", "flow_intensity", "notes", "created_at", "updated_at"}).
		AddRow(nil, nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.ListCycles(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateCycle_Success(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()
	cycle := &port.UserMenstrualCycle{
		UserID:         "user-1",
		CycleStartDate: "2024-01-01",
		CycleEndDate:   "2024-01-05",
		FlowIntensity:  "medium",
		Notes:          "Mild cramps",
	}

	rows := sqlmock.NewRows([]string{"id"}).AddRow("cycle-1")
	mock.ExpectQuery("INSERT INTO user_menstrual_cycles").
		WithArgs("user-1", "2024-01-01", "2024-01-05", "medium", "Mild cramps").
		WillReturnRows(rows)

	result, err := repo.CreateCycle(ctx, cycle)
	require.NoError(t, err)
	assert.Equal(t, "cycle-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateCycle_NilOptionalFields(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()
	cycle := &port.UserMenstrualCycle{
		UserID:         "user-1",
		CycleStartDate: "2024-01-01",
		CycleEndDate:   "",
		FlowIntensity:  "",
		Notes:          "",
	}

	rows := sqlmock.NewRows([]string{"id"}).AddRow("cycle-2")
	mock.ExpectQuery("INSERT INTO user_menstrual_cycles").
		WithArgs("user-1", "2024-01-01", nil, nil, "").
		WillReturnRows(rows)

	result, err := repo.CreateCycle(ctx, cycle)
	require.NoError(t, err)
	assert.Equal(t, "cycle-2", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateCycle_Error(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("INSERT INTO user_menstrual_cycles").WillReturnError(assert.AnError)

	result, err := repo.CreateCycle(ctx, &port.UserMenstrualCycle{UserID: "user-1", CycleStartDate: "2024-01-01"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_UpdateCycle_Success(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()
	cycle := &port.UserMenstrualCycle{
		ID:           "cycle-1",
		UserID:       "user-1",
		CycleEndDate: "2024-01-05",
		FlowIntensity: "medium",
		Notes:        "Updated notes",
	}

	mock.ExpectExec("UPDATE user_menstrual_cycles").
		WithArgs("2024-01-05", "medium", "Updated notes", "cycle-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	result, err := repo.UpdateCycle(ctx, cycle)
	require.NoError(t, err)
	assert.Equal(t, "cycle-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_UpdateCycle_NilOptionalFields(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()
	cycle := &port.UserMenstrualCycle{
		ID:           "cycle-1",
		UserID:       "user-1",
		CycleEndDate: "",
		FlowIntensity: "",
		Notes:        "",
	}

	mock.ExpectExec("UPDATE user_menstrual_cycles").
		WithArgs(nil, nil, "", "cycle-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	result, err := repo.UpdateCycle(ctx, cycle)
	require.NoError(t, err)
	assert.Equal(t, "cycle-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_UpdateCycle_Error(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE user_menstrual_cycles").WillReturnError(assert.AnError)

	result, err := repo.UpdateCycle(ctx, &port.UserMenstrualCycle{ID: "cycle-1", UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_DeleteCycle_Success(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_menstrual_cycles").
		WithArgs("cycle-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.DeleteCycle(ctx, "cycle-1", "user-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_DeleteCycle_Error(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_menstrual_cycles").WillReturnError(assert.AnError)

	err := repo.DeleteCycle(ctx, "cycle-1", "user-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_ListSymptoms_Success(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"symptom"}).
		AddRow("cramps").
		AddRow("bloating")

	mock.ExpectQuery("SELECT symptom").
		WithArgs("cycle-1").
		WillReturnRows(rows)

	result, err := repo.ListSymptoms(ctx, "cycle-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"cramps", "bloating"}, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_ListSymptoms_Empty(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"symptom"})
	mock.ExpectQuery("SELECT symptom").
		WithArgs("cycle-1").
		WillReturnRows(rows)

	result, err := repo.ListSymptoms(ctx, "cycle-1")
	require.NoError(t, err)
	assert.Empty(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_ListSymptoms_Error(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT symptom").WillReturnError(assert.AnError)

	result, err := repo.ListSymptoms(ctx, "cycle-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateSymptom_Success(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_menstrual_symptoms").WithArgs("cycle-1", "cramps").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateSymptom(ctx, "cycle-1", "cramps")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateSymptom_Error(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_menstrual_symptoms").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnError(assert.AnError)

	err := repo.CreateSymptom(ctx, "cycle-1", "cramps")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_DeleteSymptoms_Success(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_menstrual_symptoms").
		WithArgs("cycle-1").
		WillReturnResult(sqlmock.NewResult(2, 2))

	err := repo.DeleteSymptoms(ctx, "cycle-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_DeleteSymptoms_Error(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_menstrual_symptoms").WillReturnError(assert.AnError)

	err := repo.DeleteSymptoms(ctx, "cycle-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_ListMoods_Success(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"mood"}).
		AddRow("happy").
		AddRow("anxious")

	mock.ExpectQuery("SELECT mood").
		WithArgs("cycle-1").
		WillReturnRows(rows)

	result, err := repo.ListMoods(ctx, "cycle-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"happy", "anxious"}, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_ListMoods_Empty(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"mood"})
	mock.ExpectQuery("SELECT mood").
		WithArgs("cycle-1").
		WillReturnRows(rows)

	result, err := repo.ListMoods(ctx, "cycle-1")
	require.NoError(t, err)
	assert.Empty(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_ListMoods_Error(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT mood").WillReturnError(assert.AnError)

	result, err := repo.ListMoods(ctx, "cycle-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateMood_Success(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_menstrual_moods").WithArgs("cycle-1", "happy").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateMood(ctx, "cycle-1", "happy")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateMood_Error(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_menstrual_moods").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnError(assert.AnError)

	err := repo.CreateMood(ctx, "cycle-1", "happy")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_DeleteMoods_Success(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_menstrual_moods").
		WithArgs("cycle-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.DeleteMoods(ctx, "cycle-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_DeleteMoods_Error(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_menstrual_moods").WillReturnError(assert.AnError)

	err := repo.DeleteMoods(ctx, "cycle-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateCycleWithDetails_Success(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()
	cycle := &port.UserMenstrualCycle{
		UserID:         "user-1",
		CycleStartDate: "2024-01-01",
		CycleEndDate:   "2024-01-05",
		FlowIntensity:  "medium",
		Notes:          "Mild cramps",
		Symptoms:       []string{"cramps", "bloating"},
		Moods:          []string{"happy"},
	}

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO user_menstrual_cycles").
		WithArgs("user-1", "2024-01-01", "2024-01-05", "medium", "Mild cramps").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("cycle-1"))
	mock.ExpectExec("INSERT INTO user_menstrual_symptoms").WithArgs("cycle-1", "cramps").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO user_menstrual_symptoms").WithArgs("cycle-1", "bloating").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO user_menstrual_moods").WithArgs("cycle-1", "happy").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := repo.CreateCycleWithDetails(ctx, cycle)
	require.NoError(t, err)
	assert.Equal(t, "cycle-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateCycleWithDetails_BeginTxError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectBegin().WillReturnError(assert.AnError)

	result, err := repo.CreateCycleWithDetails(ctx, &port.UserMenstrualCycle{UserID: "user-1", CycleStartDate: "2024-01-01"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateCycleWithDetails_InsertCycleError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO user_menstrual_cycles").WillReturnError(assert.AnError)

	result, err := repo.CreateCycleWithDetails(ctx, &port.UserMenstrualCycle{UserID: "user-1", CycleStartDate: "2024-01-01"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateCycleWithDetails_InsertSymptomError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO user_menstrual_cycles").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("cycle-1"))
	mock.ExpectExec("INSERT INTO user_menstrual_symptoms").WithArgs("cycle-1", "cramps").
		WillReturnError(assert.AnError)

	result, err := repo.CreateCycleWithDetails(ctx, &port.UserMenstrualCycle{
		UserID: "user-1", CycleStartDate: "2024-01-01", Symptoms: []string{"cramps"},
	})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateCycleWithDetails_InsertMoodError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO user_menstrual_cycles").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("cycle-1"))
	mock.ExpectExec("INSERT INTO user_menstrual_symptoms").WithArgs("cycle-1", "cramps").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO user_menstrual_moods").WithArgs("cycle-1", "happy").
		WillReturnError(assert.AnError)

	result, err := repo.CreateCycleWithDetails(ctx, &port.UserMenstrualCycle{
		UserID: "user-1", CycleStartDate: "2024-01-01", Symptoms: []string{"cramps"}, Moods: []string{"happy"},
	})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_CreateCycleWithDetails_CommitError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO user_menstrual_cycles").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("cycle-1"))
	mock.ExpectExec("INSERT INTO user_menstrual_symptoms").WithArgs("cycle-1", "cramps").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO user_menstrual_moods").WithArgs("cycle-1", "happy").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(assert.AnError)

	result, err := repo.CreateCycleWithDetails(ctx, &port.UserMenstrualCycle{
		UserID: "user-1", CycleStartDate: "2024-01-01", Symptoms: []string{"cramps"}, Moods: []string{"happy"},
	})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_UpdateCycleWithDetails_Success(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()
	cycle := &port.UserMenstrualCycle{
		ID:           "cycle-1",
		UserID:       "user-1",
		CycleEndDate: "2024-01-05",
		FlowIntensity: "medium",
		Notes:        "Updated notes",
		Symptoms:     []string{"bloating"},
		Moods:        []string{"calm"},
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE user_menstrual_cycles").
		WithArgs("2024-01-05", "medium", "Updated notes", "cycle-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM user_menstrual_symptoms").
		WithArgs("cycle-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO user_menstrual_symptoms").WithArgs("cycle-1", "bloating").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM user_menstrual_moods").
		WithArgs("cycle-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO user_menstrual_moods").WithArgs("cycle-1", "calm").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := repo.UpdateCycleWithDetails(ctx, cycle)
	require.NoError(t, err)
	assert.Equal(t, "cycle-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_UpdateCycleWithDetails_BeginTxError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectBegin().WillReturnError(assert.AnError)

	result, err := repo.UpdateCycleWithDetails(ctx, &port.UserMenstrualCycle{ID: "cycle-1", UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_UpdateCycleWithDetails_UpdateError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE user_menstrual_cycles").WillReturnError(assert.AnError)

	result, err := repo.UpdateCycleWithDetails(ctx, &port.UserMenstrualCycle{ID: "cycle-1", UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_UpdateCycleWithDetails_DeleteSymptomsError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE user_menstrual_cycles").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM user_menstrual_symptoms").WillReturnError(assert.AnError)

	result, err := repo.UpdateCycleWithDetails(ctx, &port.UserMenstrualCycle{ID: "cycle-1", UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_UpdateCycleWithDetails_InsertSymptomError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE user_menstrual_cycles").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM user_menstrual_symptoms").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO user_menstrual_symptoms").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnError(assert.AnError)

	result, err := repo.UpdateCycleWithDetails(ctx, &port.UserMenstrualCycle{ID: "cycle-1", UserID: "user-1", Symptoms: []string{"cramps"}})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_UpdateCycleWithDetails_DeleteMoodsError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE user_menstrual_cycles").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM user_menstrual_symptoms").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM user_menstrual_moods").WillReturnError(assert.AnError)

	result, err := repo.UpdateCycleWithDetails(ctx, &port.UserMenstrualCycle{ID: "cycle-1", UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_UpdateCycleWithDetails_InsertMoodError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE user_menstrual_cycles").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM user_menstrual_symptoms").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM user_menstrual_moods").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO user_menstrual_moods").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnError(assert.AnError)

	result, err := repo.UpdateCycleWithDetails(ctx, &port.UserMenstrualCycle{ID: "cycle-1", UserID: "user-1", Moods: []string{"happy"}})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserMenstrualRepository_UpdateCycleWithDetails_CommitError(t *testing.T) {
	repo, mock := setupMenstrualRepo(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE user_menstrual_cycles").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM user_menstrual_symptoms").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM user_menstrual_moods").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(assert.AnError)

	result, err := repo.UpdateCycleWithDetails(ctx, &port.UserMenstrualCycle{ID: "cycle-1", UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

