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
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/domain/entity"
)

func setupPgsodiumUserRepo(t *testing.T) (*PgsodiumUserRepository, sqlmock.Sqlmock) {
	t.Helper()

	db.ResetPgsodiumKeyID()
	db.ResetTestEncryptionKey()

	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewPgsodiumUserRepository(dbConn)
	return repo, mock
}

func TestPgsodiumUserRepository_Create_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()
	user := &entity.User{
		ID:            "user-1",
		Email:         "test@example.com",
		PasswordHash:  "hash",
		FullName:      "Test User",
		Role:          "user",
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	mock.ExpectExec("INSERT INTO users").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Create(ctx, user)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_Create_NonceError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()
	user := &entity.User{
		ID:            "user-1",
		Email:         "test@example.com",
		PasswordHash:  "hash",
		FullName:      "Test User",
		Role:          "user",
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := repo.Create(ctx, user)

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_GetByID_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email_hash", "password_hash", "full_name_hash", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "email_hash", "hash", "full_name_hash", "user", true, now, now)

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetByID(ctx, "user-1")

	require.NoError(t, err)
	assert.Equal(t, "user-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByID(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_GetByID_QueryError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	result, err := repo.GetByID(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_GetByEmail_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()
	now := time.Now()

	emailHash := db.BlindIndex("test@example.com")

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name_hash", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "test@example.com", "hash", "full_name_hash", "user", true, now, now)

	mock.ExpectQuery("SELECT id, email").
		WithArgs(emailHash).
		WillReturnRows(rows)

	result, err := repo.GetByEmail(ctx, "test@example.com")

	require.NoError(t, err)
	assert.Equal(t, "user-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_GetByEmail_NotFound(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	emailHash := db.BlindIndex("test@example.com")

	mock.ExpectQuery("SELECT id, email").
		WithArgs(emailHash).
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByEmail(ctx, "test@example.com")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_Update_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()
	user := &entity.User{
		ID:       "user-1",
		FullName: "Updated Name",
	}

	mock.ExpectExec("UPDATE users SET full_name_encrypted").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(ctx, user)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_Update_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()
	user := &entity.User{
		ID:       "user-1",
		FullName: "Updated Name",
	}

	mock.ExpectExec("UPDATE users SET full_name_encrypted").
		WillReturnError(assert.AnError)

	err := repo.Update(ctx, user)

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_Delete_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM users WHERE id = \\$1").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_Delete_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM users WHERE id = \\$1").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.Delete(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_List_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "test@example.com", "Test User", "user", true, now, now)

	mock.ExpectQuery("SELECT u.id").
		WillReturnRows(rows)

	result, err := repo.List(ctx, 1, 10)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "user-1", result[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_List_QueryError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT u.id").
		WillReturnError(assert.AnError)

	result, err := repo.List(ctx, 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_Count_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	count, err := repo.Count(ctx)

	require.NoError(t, err)
	assert.Equal(t, 42, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_Count_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
		WillReturnError(assert.AnError)

	count, err := repo.Count(ctx)

	require.Error(t, err)
	assert.Equal(t, 0, count)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_ExistsByEmail_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	emailHash := db.BlindIndex("test@example.com")

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(emailHash).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := repo.ExistsByEmail(ctx, "test@example.com")

	require.NoError(t, err)
	assert.True(t, exists)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepository_ExistsByEmail_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	emailHash := db.BlindIndex("test@example.com")

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(emailHash).
		WillReturnError(assert.AnError)

	exists, err := repo.ExistsByEmail(ctx, "test@example.com")

	require.Error(t, err)
	assert.False(t, exists)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNullableProfileFields_ApplyTo_AllFields(t *testing.T) {
	profile := &entity.User{}
	fields := nullableProfileFields{
		Nickname:        sql.NullString{String: "nickname", Valid: true},
		ProfilePhotoURL: sql.NullString{String: "photo.jpg", Valid: true},
		Age:             sql.NullInt32{Int32: 25, Valid: true},
		Gender:          sql.NullString{String: "male", Valid: true},
		HeightCm:        sql.NullInt32{Int32: 180, Valid: true},
		WeightKg:        sql.NullFloat64{Float64: 75.5, Valid: true},
		FitnessLevel:    sql.NullString{String: "intermediate", Valid: true},
		Nutrition:       sql.NullString{String: "balanced", Valid: true},
		SleepHours:      sql.NullFloat64{Float64: 7.5, Valid: true},
	}

	fields.applyTo(profile)

	assert.Equal(t, "nickname", profile.Nickname)
	assert.Equal(t, "photo.jpg", profile.ProfilePhotoURL)
	assert.Equal(t, int32(25), profile.Age)
	assert.Equal(t, "male", profile.Gender)
	assert.Equal(t, int32(180), profile.HeightCm)
	assert.Equal(t, float64(75.5), profile.WeightKg)
	assert.Equal(t, "intermediate", profile.FitnessLevel)
	assert.Equal(t, "balanced", profile.Nutrition)
	assert.Equal(t, float32(7.5), profile.SleepHours)
}

func TestNullableProfileFields_ApplyTo_InvalidFields(t *testing.T) {
	profile := &entity.User{}
	fields := nullableProfileFields{}

	fields.applyTo(profile)

	assert.Empty(t, profile.Nickname)
	assert.Empty(t, profile.ProfilePhotoURL)
	assert.Empty(t, profile.Age)
	assert.Empty(t, profile.Gender)
	assert.Empty(t, profile.HeightCm)
	assert.Empty(t, profile.WeightKg)
	assert.Empty(t, profile.FitnessLevel)
	assert.Empty(t, profile.Nutrition)
	assert.Empty(t, profile.SleepHours)
}
