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
	"github.com/MAMUER/project/internal/domain/entity"
)

func setupUserRepo(t *testing.T) (*UserRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	mock.ExpectPrepare(`SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at FROM users WHERE id = \$1`).
		WillReturnError(nil)

	mock.ExpectPrepare(`SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at FROM users WHERE email = \$1`).
		WillReturnError(nil)

	mock.ExpectPrepare(`SELECT EXISTS\(SELECT 1 FROM users WHERE email = \$1\)`).
		WillReturnError(nil)

	mock.ExpectPrepare(`SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at, COUNT\(\*\) OVER\(\) AS total_count FROM users WHERE role = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
		WillReturnError(nil)

	mock.ExpectPrepare(`SELECT COUNT\(\*\) FROM users`).
		WillReturnError(nil)

	repo := NewUserRepository(db).(*UserRepository)
	return repo, mock
}

func TestUserRepository_GetByID_Success(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "test@example.com", "hash", "Test User", "user", true, now, now)

	mock.ExpectQuery("SELECT id, email").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetByID(ctx, "user-1")

	require.NoError(t, err)
	assert.Equal(t, "user-1", result.ID)
	assert.Equal(t, "test@example.com", result.Email)
	assert.Equal(t, "Test User", result.FullName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByID(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByID_QueryError(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	result, err := repo.GetByID(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail_Success(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "test@example.com", "hash", "Test User", "user", true, now, now)

	mock.ExpectQuery("SELECT id, email").
		WithArgs("test@example.com").
		WillReturnRows(rows)

	result, err := repo.GetByEmail(ctx, "test@example.com")

	require.NoError(t, err)
	assert.Equal(t, "user-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email").
		WithArgs("test@example.com").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByEmail(ctx, "test@example.com")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Create_Success(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()
	user := &entity.User{
		ID:             "user-1",
		Email:          "test@example.com",
		PasswordHash:   "hash",
		FullName:       "Test User",
		Role:           "user",
		EmailVerified:  false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.ID, user.Email, user.Email, user.PasswordHash, user.FullName, user.FullName, user.Role, user.EmailVerified, user.CreatedAt, user.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Create(ctx, user)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Create_Error(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()
	user := &entity.User{
		ID:             "user-1",
		Email:          "test@example.com",
		PasswordHash:   "hash",
		FullName:       "Test User",
		Role:           "user",
		EmailVerified:  false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.ID, user.Email, user.Email, user.PasswordHash, user.FullName, user.FullName, user.Role, user.EmailVerified, user.CreatedAt, user.UpdatedAt).
		WillReturnError(assert.AnError)

	err := repo.Create(ctx, user)

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update_Success(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()
	user := &entity.User{
		ID:        "user-1",
		FullName:  "Updated Name",
		UpdatedAt: time.Now(),
	}

	mock.ExpectExec("UPDATE users SET full_name").
		WithArgs("Updated Name", user.UpdatedAt, "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(ctx, user)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update_Error(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()
	user := &entity.User{
		ID:        "user-1",
		FullName:  "Updated Name",
		UpdatedAt: time.Now(),
	}

	mock.ExpectExec("UPDATE users SET full_name").
		WithArgs("Updated Name", user.UpdatedAt, "user-1").
		WillReturnError(assert.AnError)

	err := repo.Update(ctx, user)

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Delete_Success(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM users WHERE id = \\$1").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Delete_Error(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM users WHERE id = \\$1").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.Delete(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_List_Success(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "test@example.com", "hash", "Test User", "user", true, now, now)

	mock.ExpectQuery("SELECT id, email, password_hash").
		WithArgs(10, 0).
		WillReturnRows(rows)

	result, err := repo.List(ctx, 1, 10)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "user-1", result[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_List_QueryError(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email, password_hash").
		WithArgs(10, 0).
		WillReturnError(assert.AnError)

	result, err := repo.List(ctx, 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_List_ScanError(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id, email, password_hash").
		WithArgs(10, 0).
		WillReturnRows(rows)

	result, err := repo.List(ctx, 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_List_RowsError(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "test@example.com", "hash", "Test User", "user", true, time.Now(), time.Now())

	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT id, email, password_hash").
		WithArgs(10, 0).
		WillReturnRows(rows)

	result, err := repo.List(ctx, 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ListByRole_Success(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at", "total_count"}).
		AddRow("user-1", "test@example.com", "hash", "Test User", "admin", true, now, now, 1)

	mock.ExpectQuery("SELECT id, email, password_hash").
		WithArgs("admin", 10, 0).
		WillReturnRows(rows)

	result, total, err := repo.ListByRole(ctx, "admin", 1, 10)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "user-1", result[0].ID)
	assert.Equal(t, 1, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ListByRole_QueryError(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email, password_hash").
		WithArgs("admin", 10, 0).
		WillReturnError(assert.AnError)

	result, total, err := repo.ListByRole(ctx, "admin", 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ListByRole_ScanError(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at", "total_count"}).
		AddRow("user-1", nil, nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id, email, password_hash").
		WithArgs("admin", 10, 0).
		WillReturnRows(rows)

	result, total, err := repo.ListByRole(ctx, "admin", 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ListByRole_RowsError(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at", "total_count"}).
		AddRow("user-1", "test@example.com", "hash", "Test User", "admin", true, time.Now(), time.Now(), 1)

	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT id, email, password_hash").
		WithArgs("admin", 10, 0).
		WillReturnRows(rows)

	result, total, err := repo.ListByRole(ctx, "admin", 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Count_Success(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	count, err := repo.Count(ctx)

	require.NoError(t, err)
	assert.Equal(t, 42, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Count_Error(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
		WillReturnError(assert.AnError)

	count, err := repo.Count(ctx)

	require.Error(t, err)
	assert.Equal(t, 0, count)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ExistsByEmail_Success(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := repo.ExistsByEmail(ctx, "test@example.com")

	require.NoError(t, err)
	assert.True(t, exists)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ExistsByEmail_Error(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("test@example.com").
		WillReturnError(assert.AnError)

	exists, err := repo.ExistsByEmail(ctx, "test@example.com")

	require.Error(t, err)
	assert.False(t, exists)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}
