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

func setupHealthConditionRepo(t *testing.T) (*userHealthConditionRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewUserHealthConditionRepository(db).(*userHealthConditionRepository), mock
}

func TestUserHealthConditionRepository_List_Success(t *testing.T) {
	repo, mock := setupHealthConditionRepo(t)
	ctx := context.Background()
	now := time.Now()
	diagnosedAt := time.Now().Add(-30 * 24 * time.Hour)
	notes := "Mild symptoms"

	rows := sqlmock.NewRows([]string{"id", "user_id", "condition_type", "condition_name", "severity", "diagnosed_at", "is_active", "notes", "created_at", "updated_at"}).
		AddRow("hc-1", "user-1", "chronic", "Hypertension", "moderate", diagnosedAt, true, notes, now, now)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "hc-1", result[0].ID)
	assert.Equal(t, "Hypertension", result[0].ConditionName)
	assert.NotNil(t, result[0].DiagnosedAt)
	assert.NotNil(t, result[0].Notes)
	assert.Equal(t, "Mild symptoms", *result[0].Notes)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserHealthConditionRepository_List_Empty(t *testing.T) {
	repo, mock := setupHealthConditionRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "condition_type", "condition_name", "severity", "diagnosed_at", "is_active", "notes", "created_at", "updated_at"})

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")
	require.NoError(t, err)
	assert.Empty(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserHealthConditionRepository_List_QueryError(t *testing.T) {
	repo, mock := setupHealthConditionRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").WillReturnError(assert.AnError)

	result, err := repo.List(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserHealthConditionRepository_List_ScanError(t *testing.T) {
	repo, mock := setupHealthConditionRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "condition_type", "condition_name", "severity", "diagnosed_at", "is_active", "notes", "created_at", "updated_at"}).
		AddRow(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserHealthConditionRepository_List_RowsError(t *testing.T) {
	repo, mock := setupHealthConditionRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "condition_type", "condition_name", "severity", "diagnosed_at", "is_active", "notes", "created_at", "updated_at"}).
		AddRow("hc-1", "user-1", "chronic", "Hypertension", "moderate", nil, true, nil, time.Now(), time.Now())
	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserHealthConditionRepository_Upsert_WithDiagnosedAtAndNotes(t *testing.T) {
	repo, mock := setupHealthConditionRepo(t)
	ctx := context.Background()
	diagnosedAt := time.Now().Add(-30 * 24 * time.Hour)
	notes := "Mild symptoms"
	condition := &port.UserHealthCondition{
		UserID:        "user-1",
		ConditionType: "chronic",
		ConditionName: "Hypertension",
		Severity:      "moderate",
		DiagnosedAt:   &diagnosedAt,
		IsActive:      true,
		Notes:         &notes,
	}

	rows := sqlmock.NewRows([]string{"id"}).AddRow("hc-1")
	mock.ExpectQuery("INSERT INTO user_health_conditions").
		WithArgs("user-1", "chronic", "Hypertension", "moderate", diagnosedAt, true, notes).
		WillReturnRows(rows)

	result, err := repo.Upsert(ctx, condition)
	require.NoError(t, err)
	assert.Equal(t, "hc-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserHealthConditionRepository_Upsert_NilDiagnosedAtAndNotes(t *testing.T) {
	repo, mock := setupHealthConditionRepo(t)
	ctx := context.Background()
	condition := &port.UserHealthCondition{
		UserID:        "user-1",
		ConditionType: "allergy",
		ConditionName: "Peanuts",
		Severity:      "mild",
		IsActive:      true,
	}

	rows := sqlmock.NewRows([]string{"id"}).AddRow("hc-2")
	mock.ExpectQuery("INSERT INTO user_health_conditions").
		WithArgs("user-1", "allergy", "Peanuts", "mild", nil, true, nil).
		WillReturnRows(rows)

	result, err := repo.Upsert(ctx, condition)
	require.NoError(t, err)
	assert.Equal(t, "hc-2", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserHealthConditionRepository_Upsert_Error(t *testing.T) {
	repo, mock := setupHealthConditionRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("INSERT INTO user_health_conditions").
		WillReturnError(assert.AnError)

	result, err := repo.Upsert(ctx, &port.UserHealthCondition{UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserHealthConditionRepository_Delete_Success(t *testing.T) {
	repo, mock := setupHealthConditionRepo(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_health_conditions").
		WithArgs("hc-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(ctx, "hc-1", "user-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserHealthConditionRepository_Delete_Error(t *testing.T) {
	repo, mock := setupHealthConditionRepo(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_health_conditions").
		WithArgs("hc-1", "user-1").
		WillReturnError(assert.AnError)

	err := repo.Delete(ctx, "hc-1", "user-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}
