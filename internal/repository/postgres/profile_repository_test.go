package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

func setupProfileRepo(t *testing.T) (*profileRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return &profileRepository{db: db}, mock
}

func TestProfileRepository_GetProfile_Success(t *testing.T) {
	repo, mock := setupProfileRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "test@example.com", "hash", "Test User", "user", true, now, now)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetProfile(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "user-1", result.ID)
	assert.Equal(t, "Test User", result.FullName)
	assert.True(t, result.EmailVerified)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProfileRepository_GetProfile_NotFound(t *testing.T) {
	repo, mock := setupProfileRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetProfile(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProfileRepository_GetProfile_QueryError(t *testing.T) {
	repo, mock := setupProfileRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").WillReturnError(assert.AnError)

	result, err := repo.GetProfile(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProfileRepository_UpdateProfile_Success(t *testing.T) {
	repo, mock := setupProfileRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE users").
		WithArgs("Test User", sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateProfile(ctx, "user-1", "Test User", nil, nil, "balanced", 7.5)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProfileRepository_UpdateProfile_Error(t *testing.T) {
	repo, mock := setupProfileRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE users").WillReturnError(assert.AnError)

	err := repo.UpdateProfile(ctx, "user-1", "Test User", nil, nil, "balanced", 7.5)
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProfileRepository_UserExists_True(t *testing.T) {
	repo, mock := setupProfileRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("user-1").
		WillReturnRows(rows)

	exists, err := repo.UserExists(ctx, "user-1")
	require.NoError(t, err)
	assert.True(t, exists)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProfileRepository_UserExists_False(t *testing.T) {
	repo, mock := setupProfileRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("user-1").
		WillReturnRows(rows)

	exists, err := repo.UserExists(ctx, "user-1")
	require.NoError(t, err)
	assert.False(t, exists)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProfileRepository_UserExists_Error(t *testing.T) {
	repo, mock := setupProfileRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT EXISTS").WillReturnError(assert.AnError)

	exists, err := repo.UserExists(ctx, "user-1")
	require.Error(t, err)
	assert.False(t, exists)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProfileRepository_CreateProfile_Success(t *testing.T) {
	repo, mock := setupProfileRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_profiles").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateProfile(ctx, "user-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProfileRepository_CreateProfile_Error(t *testing.T) {
	repo, mock := setupProfileRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_profiles").WillReturnError(assert.AnError)

	err := repo.CreateProfile(ctx, "user-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProfileRepository_UpsertProfile_Success(t *testing.T) {
	repo, mock := setupProfileRepo(t)
	ctx := context.Background()
	data := &port.ProfileData{
		Age:          30,
		Gender:       "male",
		HeightCm:     180,
		WeightKg:     75.0,
		FitnessLevel: "intermediate",
		Nutrition:    "balanced",
		SleepHours:   7.5,
	}

	mock.ExpectExec("INSERT INTO user_profiles").
		WithArgs("user-1", int32(30), "male", int32(180), 75.0, "intermediate", "balanced", float32(7.5)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpsertProfile(ctx, "user-1", data)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProfileRepository_UpsertProfile_Error(t *testing.T) {
	repo, mock := setupProfileRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_profiles").WillReturnError(assert.AnError)

	err := repo.UpsertProfile(ctx, "user-1", &port.ProfileData{})
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}
