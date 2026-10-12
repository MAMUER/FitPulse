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
	"github.com/MAMUER/project/internal/domain/port"
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
		WithArgs("Updated Name", sqlmock.AnyArg(), "user-1").
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
		WithArgs("Updated Name", sqlmock.AnyArg(), "user-1").
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

func TestUserRepository_ExistsByEmail_NoRows(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("test@example.com").
		WillReturnError(sql.ErrNoRows)

	exists, err := repo.ExistsByEmail(ctx, "test@example.com")

	require.Error(t, err)
	assert.False(t, exists)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Count_NoRows(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
		WillReturnError(sql.ErrNoRows)

	count, err := repo.Count(ctx)

	require.Error(t, err)
	assert.Equal(t, 0, count)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ListByRole_Empty(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at", "total_count"})

	mock.ExpectQuery("SELECT id, email, password_hash").
		WithArgs("admin", 10, 0).
		WillReturnRows(rows)

	result, total, err := repo.ListByRole(ctx, "admin", 1, 10)

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 0, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ListByRole_SecondPage(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at", "total_count"}).
		AddRow("user-2", "other@example.com", "hash", "Other User", "user", false, now, now, 15)

	mock.ExpectQuery("SELECT id, email, password_hash").
		WithArgs("user", 5, 10).
		WillReturnRows(rows)

	result, total, err := repo.ListByRole(ctx, "user", 3, 5)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "user-2", result[0].ID)
	assert.Equal(t, 15, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_List_Empty(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at"})

	mock.ExpectQuery("SELECT id, email, password_hash").
		WithArgs(10, 0).
		WillReturnRows(rows)

	result, err := repo.List(ctx, 1, 10)

	require.NoError(t, err)
	assert.Empty(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_List_SecondPage(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-2", "other@example.com", "hash", "Other User", "user", false, now, now)

	mock.ExpectQuery("SELECT id, email, password_hash").
		WithArgs(10, 10).
		WillReturnRows(rows)

	result, err := repo.List(ctx, 2, 10)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "user-2", result[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByID_ScanError(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id"}).
		AddRow("user-1")

	mock.ExpectQuery("SELECT id, email").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetByID(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func setupURPrepareDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	return db, mock
}

func TestUserRepository_PrepareStatements_GetByIDPanic(t *testing.T) {
	db, mock := setupURPrepareDB(t)

	mock.ExpectPrepare(`SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at FROM users WHERE id = \$1`).
		WillReturnError(assert.AnError)

	assert.Panics(t, func() {
		_ = NewUserRepository(db)
	})
}

func TestUserRepository_PrepareStatements_GetByEmailPanic(t *testing.T) {
	db, mock := setupURPrepareDB(t)

	mock.ExpectPrepare(`SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at FROM users WHERE id = \$1`)

	mock.ExpectPrepare(`SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at FROM users WHERE email = \$1`).
		WillReturnError(assert.AnError)

	assert.Panics(t, func() {
		_ = NewUserRepository(db)
	})
}

func TestUserRepository_PrepareStatements_ExistsByEmailPanic(t *testing.T) {
	db, mock := setupURPrepareDB(t)

	mock.ExpectPrepare(`SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at FROM users WHERE id = \$1`)

	mock.ExpectPrepare(`SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at FROM users WHERE email = \$1`)

	mock.ExpectPrepare(`SELECT EXISTS\(SELECT 1 FROM users WHERE email = \$1\)`).
		WillReturnError(assert.AnError)

	assert.Panics(t, func() {
		_ = NewUserRepository(db)
	})
}

func TestUserRepository_PrepareStatements_ListByRolePanic(t *testing.T) {
	db, mock := setupURPrepareDB(t)

	mock.ExpectPrepare(`SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at FROM users WHERE id = \$1`)

	mock.ExpectPrepare(`SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at FROM users WHERE email = \$1`)

	mock.ExpectPrepare(`SELECT EXISTS\(SELECT 1 FROM users WHERE email = \$1\)`)

	mock.ExpectPrepare(`COUNT\(\*\) OVER\(\) AS total_count FROM users WHERE role = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
		WillReturnError(assert.AnError)

	assert.Panics(t, func() {
		_ = NewUserRepository(db)
	})
}

func TestUserRepository_PrepareStatements_CountPanic(t *testing.T) {
	db, mock := setupURPrepareDB(t)

	mock.ExpectPrepare(`SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at FROM users WHERE id = \$1`)

	mock.ExpectPrepare(`SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at FROM users WHERE email = \$1`)

	mock.ExpectPrepare(`SELECT EXISTS\(SELECT 1 FROM users WHERE email = \$1\)`)

	mock.ExpectPrepare(`COUNT\(\*\) OVER\(\) AS total_count FROM users WHERE role = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`)

	mock.ExpectPrepare(`SELECT COUNT\(\*\) FROM users`).
		WillReturnError(assert.AnError)

	assert.Panics(t, func() {
		_ = NewUserRepository(db)
	})
}

func setupURInviteRepo(t *testing.T) (*inviteRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	return NewInviteRepository(db).(*inviteRepository), mock
}

func TestUserRepoInvite_Create_Success(t *testing.T) {
	repo, mock := setupURInviteRepo(t)

	ctx := context.Background()

	now := time.Now()

	invite := &port.Invite{
		Code:      "INVITE-1",
		Role:      "coach",
		Specialty: "cardio",
		MaxUses:   10,
		UsedCount: 0,
		IsActive:  true,
		CreatedAt: now,
	}

	mock.ExpectExec("INSERT INTO invites").
		WithArgs(invite.Code, invite.Role, invite.Specialty, invite.MaxUses, invite.UsedCount, invite.IsActive, invite.CreatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Create(ctx, invite)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoInvite_Create_Error(t *testing.T) {
	repo, mock := setupURInviteRepo(t)

	ctx := context.Background()

	now := time.Now()

	invite := &port.Invite{
		Code:      "INVITE-1",
		Role:      "coach",
		Specialty: "cardio",
		MaxUses:   10,
		UsedCount: 0,
		IsActive:  true,
		CreatedAt: now,
	}

	mock.ExpectExec("INSERT INTO invites").
		WithArgs(invite.Code, invite.Role, invite.Specialty, invite.MaxUses, invite.UsedCount, invite.IsActive, invite.CreatedAt).
		WillReturnError(assert.AnError)

	err := repo.Create(ctx, invite)

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoInvite_GetByCode_Success(t *testing.T) {
	repo, mock := setupURInviteRepo(t)

	ctx := context.Background()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"code", "role", "specialty", "max_uses", "used_count", "is_active", "created_at"}).
		AddRow("INVITE-1", "coach", "cardio", 10, 3, true, now)

	mock.ExpectQuery("SELECT code, role, specialty, max_uses, used_count, is_active, created_at").
		WithArgs("INVITE-1").
		WillReturnRows(rows)

	invite, err := repo.GetByCode(ctx, "INVITE-1")

	require.NoError(t, err)
	require.NotNil(t, invite)
	assert.Equal(t, "INVITE-1", invite.Code)
	assert.Equal(t, "coach", invite.Role)
	assert.Equal(t, "cardio", invite.Specialty)
	assert.Equal(t, 10, invite.MaxUses)
	assert.Equal(t, 3, invite.UsedCount)
	assert.True(t, invite.IsActive)
	assert.Equal(t, now, invite.CreatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoInvite_GetByCode_NotFound(t *testing.T) {
	repo, mock := setupURInviteRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT code, role, specialty").
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	invite, err := repo.GetByCode(ctx, "missing")

	require.Error(t, err)
	assert.Nil(t, invite)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoInvite_GetByCode_Error(t *testing.T) {
	repo, mock := setupURInviteRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT code, role, specialty").
		WithArgs("INVITE-1").
		WillReturnError(assert.AnError)

	invite, err := repo.GetByCode(ctx, "INVITE-1")

	require.Error(t, err)
	assert.Nil(t, invite)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoInvite_List_Success(t *testing.T) {
	repo, mock := setupURInviteRepo(t)

	ctx := context.Background()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"code", "role", "specialty", "max_uses", "used_count", "is_active", "created_at", "total_count"}).
		AddRow("INVITE-1", "coach", "cardio", 10, 3, true, now, 1)

	mock.ExpectQuery("SELECT code, role, specialty, max_uses, used_count, is_active, created_at").
		WithArgs(5, 5).
		WillReturnRows(rows)

	invites, total, err := repo.List(ctx, 2, 5)

	require.NoError(t, err)
	require.Len(t, invites, 1)
	assert.Equal(t, "INVITE-1", invites[0].Code)
	assert.Equal(t, 1, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoInvite_List_Empty(t *testing.T) {
	repo, mock := setupURInviteRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"code", "role", "specialty", "max_uses", "used_count", "is_active", "created_at", "total_count"})

	mock.ExpectQuery("SELECT code, role, specialty").
		WithArgs(10, 0).
		WillReturnRows(rows)

	invites, total, err := repo.List(ctx, 1, 10)

	require.NoError(t, err)
	assert.Empty(t, invites)
	assert.Equal(t, 0, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoInvite_List_QueryError(t *testing.T) {
	repo, mock := setupURInviteRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT code, role, specialty").
		WithArgs(10, 0).
		WillReturnError(assert.AnError)

	invites, total, err := repo.List(ctx, 1, 10)

	require.Error(t, err)
	assert.Nil(t, invites)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoInvite_List_ScanError(t *testing.T) {
	repo, mock := setupURInviteRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"code", "role", "specialty", "max_uses", "used_count", "is_active", "created_at", "total_count"}).
		AddRow("INVITE-1", nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT code, role, specialty").
		WithArgs(10, 0).
		WillReturnRows(rows)

	invites, total, err := repo.List(ctx, 1, 10)

	require.Error(t, err)
	assert.Nil(t, invites)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoInvite_List_RowsError(t *testing.T) {
	repo, mock := setupURInviteRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"code", "role", "specialty", "max_uses", "used_count", "is_active", "created_at", "total_count"}).
		AddRow("INVITE-1", "coach", "cardio", 10, 3, true, time.Now(), 1)

	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT code, role, specialty").
		WithArgs(10, 0).
		WillReturnRows(rows)

	invites, total, err := repo.List(ctx, 1, 10)

	require.Error(t, err)
	assert.Nil(t, invites)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoInvite_Revoke_Success(t *testing.T) {
	repo, mock := setupURInviteRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE invites SET is_active = false").
		WithArgs("INVITE-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Revoke(ctx, "INVITE-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoInvite_Revoke_Error(t *testing.T) {
	repo, mock := setupURInviteRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE invites SET is_active = false").
		WithArgs("INVITE-1").
		WillReturnError(assert.AnError)

	err := repo.Revoke(ctx, "INVITE-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func setupURHealthConditionRepo(t *testing.T) (*healthConditionRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	return NewHealthConditionRepository(db).(*healthConditionRepository), mock
}

func TestUserRepoHealthCondition_Create_Success(t *testing.T) {
	repo, mock := setupURHealthConditionRepo(t)

	ctx := context.Background()

	now := time.Now()

	condition := &entity.HealthCondition{
		ID:            "hc-1",
		UserID:        "user-1",
		ConditionType: "diabetes",
		Description:   "type 2",
		CreatedAt:     now,
	}

	mock.ExpectQuery("INSERT INTO health_conditions").
		WithArgs(condition.ID, condition.UserID, condition.ConditionType, condition.Description, condition.CreatedAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("hc-1"))

	result, err := repo.Create(ctx, condition)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "hc-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoHealthCondition_Create_Error(t *testing.T) {
	repo, mock := setupURHealthConditionRepo(t)

	ctx := context.Background()

	condition := &entity.HealthCondition{
		ID:            "hc-1",
		UserID:        "user-1",
		ConditionType: "diabetes",
		Description:   "type 2",
		CreatedAt:     time.Now(),
	}

	mock.ExpectQuery("INSERT INTO health_conditions").
		WillReturnError(assert.AnError)

	result, err := repo.Create(ctx, condition)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoHealthCondition_List_WithConditionType(t *testing.T) {
	repo, mock := setupURHealthConditionRepo(t)

	ctx := context.Background()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "condition_type", "description", "created_at"}).
		AddRow("hc-1", "user-1", "diabetes", "type 2", now)

	mock.ExpectQuery("SELECT id, user_id, condition_type, description, created_at").
		WithArgs("user-1", "diabetes").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", "diabetes")

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "hc-1", result[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoHealthCondition_List_WithoutConditionType(t *testing.T) {
	repo, mock := setupURHealthConditionRepo(t)

	ctx := context.Background()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "condition_type", "description", "created_at"}).
		AddRow("hc-1", "user-1", "diabetes", "type 2", now)

	mock.ExpectQuery("SELECT id, user_id, condition_type, description, created_at").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", "")

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "user-1", result[0].UserID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoHealthCondition_List_QueryError(t *testing.T) {
	repo, mock := setupURHealthConditionRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, user_id, condition_type").
		WithArgs("user-1", "diabetes").
		WillReturnError(assert.AnError)

	result, err := repo.List(ctx, "user-1", "diabetes")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoHealthCondition_List_ScanError(t *testing.T) {
	repo, mock := setupURHealthConditionRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "condition_type", "description", "created_at"}).
		AddRow("hc-1", nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id, user_id, condition_type").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", "")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoHealthCondition_List_RowsError(t *testing.T) {
	repo, mock := setupURHealthConditionRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "condition_type", "description", "created_at"}).
		AddRow("hc-1", "user-1", "diabetes", "type 2", time.Now())

	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT id, user_id, condition_type").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", "")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoHealthCondition_Delete_Success(t *testing.T) {
	repo, mock := setupURHealthConditionRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM health_conditions WHERE id = \\$1").
		WithArgs("hc-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, "hc-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoHealthCondition_Delete_Error(t *testing.T) {
	repo, mock := setupURHealthConditionRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM health_conditions WHERE id = \\$1").
		WithArgs("hc-1").
		WillReturnError(assert.AnError)

	err := repo.Delete(ctx, "hc-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func setupURBodyCompositionRepo(t *testing.T) (*bodyCompositionRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	return NewBodyCompositionRepository(db).(*bodyCompositionRepository), mock
}

func TestUserRepoBodyComposition_Create_Success(t *testing.T) {
	repo, mock := setupURBodyCompositionRepo(t)

	ctx := context.Background()

	now := time.Now()

	bc := &entity.BodyComposition{
		ID:         "bc-1",
		UserID:     "user-1",
		WeightKG:   70.5,
		HeightCM:   175.0,
		BMI:        23.0,
		RecordedAt: now,
	}

	mock.ExpectQuery("INSERT INTO body_composition").
		WithArgs(bc.ID, bc.UserID, bc.WeightKG, bc.HeightCM, bc.BMI, bc.RecordedAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("bc-1"))

	result, err := repo.Create(ctx, bc)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "bc-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoBodyComposition_Create_Error(t *testing.T) {
	repo, mock := setupURBodyCompositionRepo(t)

	ctx := context.Background()

	bc := &entity.BodyComposition{
		ID:         "bc-1",
		UserID:     "user-1",
		WeightKG:   70.5,
		HeightCM:   175.0,
		BMI:        23.0,
		RecordedAt: time.Now(),
	}

	mock.ExpectQuery("INSERT INTO body_composition").
		WillReturnError(assert.AnError)

	result, err := repo.Create(ctx, bc)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoBodyComposition_List_NoFilters(t *testing.T) {
	repo, mock := setupURBodyCompositionRepo(t)

	ctx := context.Background()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "weight_kg", "height_cm", "bmi", "recorded_at"}).
		AddRow("bc-1", "user-1", 70.5, 175.0, 23.0, now)

	mock.ExpectQuery("SELECT id, user_id, weight_kg, height_cm, bmi, recorded_at").
		WithArgs("user-1", 10).
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", nil, nil, 10)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "bc-1", result[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoBodyComposition_List_FromAndTo(t *testing.T) {
	repo, mock := setupURBodyCompositionRepo(t)

	ctx := context.Background()

	from := time.Now().Add(-48 * time.Hour)

	to := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "weight_kg", "height_cm", "bmi", "recorded_at"}).
		AddRow("bc-1", "user-1", 70.5, 175.0, 23.0, to)

	mock.ExpectQuery("SELECT id, user_id, weight_kg, height_cm, bmi, recorded_at").
		WithArgs("user-1", from, to, 10).
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", &from, &to, 10)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoBodyComposition_List_QueryError(t *testing.T) {
	repo, mock := setupURBodyCompositionRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, user_id, weight_kg").
		WithArgs("user-1", 10).
		WillReturnError(assert.AnError)

	result, err := repo.List(ctx, "user-1", nil, nil, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoBodyComposition_List_ScanError(t *testing.T) {
	repo, mock := setupURBodyCompositionRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "weight_kg", "height_cm", "bmi", "recorded_at"}).
		AddRow("bc-1", nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id, user_id, weight_kg").
		WithArgs("user-1", 10).
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", nil, nil, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoBodyComposition_List_RowsError(t *testing.T) {
	repo, mock := setupURBodyCompositionRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "weight_kg", "height_cm", "bmi", "recorded_at"}).
		AddRow("bc-1", "user-1", 70.5, 175.0, 23.0, time.Now())

	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT id, user_id, weight_kg").
		WithArgs("user-1", 10).
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1", nil, nil, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func setupURMenstrualCycleRepo(t *testing.T) (*menstrualCycleRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	return NewMenstrualCycleRepository(db).(*menstrualCycleRepository), mock
}

func TestUserRepoMenstrualCycle_Create_Success(t *testing.T) {
	repo, mock := setupURMenstrualCycleRepo(t)

	ctx := context.Background()

	now := time.Now()

	cycle := &entity.MenstrualCycle{
		ID:        "mc-1",
		UserID:    "user-1",
		StartDate: now,
		EndDate:   now.Add(24 * time.Hour),
		FlowLevel: "medium",
		Notes:     "notes",
		CreatedAt: now,
	}

	mock.ExpectQuery("INSERT INTO menstrual_cycles").
		WithArgs(cycle.ID, cycle.UserID, cycle.StartDate, cycle.EndDate, cycle.FlowLevel, cycle.Notes, cycle.CreatedAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("mc-1"))

	result, err := repo.Create(ctx, cycle)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "mc-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoMenstrualCycle_Create_Error(t *testing.T) {
	repo, mock := setupURMenstrualCycleRepo(t)

	ctx := context.Background()

	cycle := &entity.MenstrualCycle{
		ID:        "mc-1",
		UserID:    "user-1",
		StartDate: time.Now(),
		EndDate:   time.Now().Add(24 * time.Hour),
		FlowLevel: "medium",
		Notes:     "notes",
		CreatedAt: time.Now(),
	}

	mock.ExpectQuery("INSERT INTO menstrual_cycles").
		WillReturnError(assert.AnError)

	result, err := repo.Create(ctx, cycle)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoMenstrualCycle_List_Success(t *testing.T) {
	repo, mock := setupURMenstrualCycleRepo(t)

	ctx := context.Background()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "start_date", "end_date", "flow_level", "notes", "created_at"}).
		AddRow("mc-1", "user-1", now, now.Add(24*time.Hour), "medium", "notes", now)

	mock.ExpectQuery("SELECT id, user_id, start_date, end_date, flow_level, notes, created_at").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "mc-1", result[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoMenstrualCycle_List_Empty(t *testing.T) {
	repo, mock := setupURMenstrualCycleRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "start_date", "end_date", "flow_level", "notes", "created_at"})

	mock.ExpectQuery("SELECT id, user_id, start_date").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")

	require.NoError(t, err)
	assert.Empty(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoMenstrualCycle_List_QueryError(t *testing.T) {
	repo, mock := setupURMenstrualCycleRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, user_id, start_date").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	result, err := repo.List(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoMenstrualCycle_List_ScanError(t *testing.T) {
	repo, mock := setupURMenstrualCycleRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "start_date", "end_date", "flow_level", "notes", "created_at"}).
		AddRow("mc-1", nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id, user_id, start_date").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoMenstrualCycle_List_RowsError(t *testing.T) {
	repo, mock := setupURMenstrualCycleRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "start_date", "end_date", "flow_level", "notes", "created_at"}).
		AddRow("mc-1", "user-1", time.Now(), time.Now(), "medium", "notes", time.Now())

	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT id, user_id, start_date").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoMenstrualCycle_Update_Success(t *testing.T) {
	repo, mock := setupURMenstrualCycleRepo(t)

	ctx := context.Background()

	now := time.Now()

	cycle := &entity.MenstrualCycle{
		ID:        "mc-1",
		UserID:    "user-1",
		StartDate: now,
		EndDate:   now.Add(24 * time.Hour),
		FlowLevel: "heavy",
		Notes:     "updated",
		CreatedAt: now,
	}

	mock.ExpectExec("UPDATE menstrual_cycles").
		WithArgs(cycle.StartDate, cycle.EndDate, cycle.FlowLevel, cycle.Notes, cycle.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := repo.Update(ctx, cycle)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "mc-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoMenstrualCycle_Update_Error(t *testing.T) {
	repo, mock := setupURMenstrualCycleRepo(t)

	ctx := context.Background()

	cycle := &entity.MenstrualCycle{
		ID:        "mc-1",
		UserID:    "user-1",
		StartDate: time.Now(),
		EndDate:   time.Now().Add(24 * time.Hour),
		FlowLevel: "heavy",
		Notes:     "updated",
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("UPDATE menstrual_cycles").
		WillReturnError(assert.AnError)

	result, err := repo.Update(ctx, cycle)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoMenstrualCycle_Delete_Success(t *testing.T) {
	repo, mock := setupURMenstrualCycleRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM menstrual_cycles WHERE id = \\$1").
		WithArgs("mc-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, "mc-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoMenstrualCycle_Delete_Error(t *testing.T) {
	repo, mock := setupURMenstrualCycleRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM menstrual_cycles WHERE id = \\$1").
		WithArgs("mc-1").
		WillReturnError(assert.AnError)

	err := repo.Delete(ctx, "mc-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func setupURAchievementRepo(t *testing.T) (*achievementRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	return NewAchievementRepository(db).(*achievementRepository), mock
}

func TestUserRepoAchievement_Create_Success(t *testing.T) {
	repo, mock := setupURAchievementRepo(t)

	ctx := context.Background()

	now := time.Now()

	achievement := &entity.Achievement{
		ID:          "ach-1",
		UserID:      "user-1",
		Type:        "first_workout",
		Title:       "First Workout",
		Description: "completed first workout",
		EarnedAt:    now,
	}

	mock.ExpectQuery("INSERT INTO achievements").
		WithArgs(achievement.ID, achievement.UserID, achievement.Type, achievement.Title, achievement.Description, achievement.EarnedAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ach-1"))

	result, err := repo.Create(ctx, achievement)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "ach-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoAchievement_Create_Error(t *testing.T) {
	repo, mock := setupURAchievementRepo(t)

	ctx := context.Background()

	achievement := &entity.Achievement{
		ID:          "ach-1",
		UserID:      "user-1",
		Type:        "first_workout",
		Title:       "First Workout",
		Description: "completed first workout",
		EarnedAt:    time.Now(),
	}

	mock.ExpectQuery("INSERT INTO achievements").
		WillReturnError(assert.AnError)

	result, err := repo.Create(ctx, achievement)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoAchievement_List_Success(t *testing.T) {
	repo, mock := setupURAchievementRepo(t)

	ctx := context.Background()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "title", "description", "earned_at"}).
		AddRow("ach-1", "user-1", "first_workout", "First Workout", "completed first workout", now)

	mock.ExpectQuery("SELECT id, user_id, type, title, description, earned_at").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "ach-1", result[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoAchievement_List_Empty(t *testing.T) {
	repo, mock := setupURAchievementRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "title", "description", "earned_at"})

	mock.ExpectQuery("SELECT id, user_id, type").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")

	require.NoError(t, err)
	assert.Empty(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoAchievement_List_QueryError(t *testing.T) {
	repo, mock := setupURAchievementRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, user_id, type").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	result, err := repo.List(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoAchievement_List_ScanError(t *testing.T) {
	repo, mock := setupURAchievementRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "title", "description", "earned_at"}).
		AddRow("ach-1", nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id, user_id, type").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoAchievement_List_RowsError(t *testing.T) {
	repo, mock := setupURAchievementRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "type", "title", "description", "earned_at"}).
		AddRow("ach-1", "user-1", "first_workout", "First Workout", "completed first workout", time.Now())

	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT id, user_id, type").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func setupURDeviceRepo(t *testing.T) (*deviceRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	return NewDeviceRepository(db).(*deviceRepository), mock
}

func TestUserRepoDevice_List_Success(t *testing.T) {
	repo, mock := setupURDeviceRepo(t)

	ctx := context.Background()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "device_type", "device_name", "is_connected", "last_sync"}).
		AddRow("dev-1", "user-1", "smartwatch", "FitBit", true, now)

	mock.ExpectQuery("SELECT id, user_id, device_type, device_name, is_connected, last_sync").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "dev-1", result[0].ID)
	assert.True(t, result[0].IsConnected)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoDevice_List_QueryError(t *testing.T) {
	repo, mock := setupURDeviceRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, user_id, device_type").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	result, err := repo.List(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoDevice_List_ScanError(t *testing.T) {
	repo, mock := setupURDeviceRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "device_type", "device_name", "is_connected", "last_sync"}).
		AddRow("dev-1", nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id, user_id, device_type").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoDevice_List_RowsError(t *testing.T) {
	repo, mock := setupURDeviceRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "device_type", "device_name", "is_connected", "last_sync"}).
		AddRow("dev-1", "user-1", "smartwatch", "FitBit", true, time.Now())

	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT id, user_id, device_type").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.List(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoDevice_Create_Success(t *testing.T) {
	repo, mock := setupURDeviceRepo(t)

	ctx := context.Background()

	device := &entity.Device{
		ID:         "dev-1",
		UserID:     "user-1",
		DeviceType: "smartwatch",
		DeviceName: "FitBit",
		Token:      "token-1",
	}

	mock.ExpectQuery("INSERT INTO devices").
		WithArgs(device.ID, device.UserID, device.DeviceType, device.DeviceName, device.Token).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("dev-1"))

	result, err := repo.Create(ctx, device)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "dev-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoDevice_Create_Error(t *testing.T) {
	repo, mock := setupURDeviceRepo(t)

	ctx := context.Background()

	device := &entity.Device{
		ID:         "dev-1",
		UserID:     "user-1",
		DeviceType: "smartwatch",
		DeviceName: "FitBit",
		Token:      "token-1",
	}

	mock.ExpectQuery("INSERT INTO devices").
		WillReturnError(assert.AnError)

	result, err := repo.Create(ctx, device)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoDevice_Delete_Success(t *testing.T) {
	repo, mock := setupURDeviceRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM devices WHERE user_id = \\$1 AND id = \\$2").
		WithArgs("user-1", "dev-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, "user-1", "dev-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoDevice_Delete_Error(t *testing.T) {
	repo, mock := setupURDeviceRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM devices WHERE user_id = \\$1 AND id = \\$2").
		WithArgs("user-1", "dev-1").
		WillReturnError(assert.AnError)

	err := repo.Delete(ctx, "user-1", "dev-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_BanUser_Success(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET is_active = FALSE").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.BanUser(ctx, "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_BanUser_Error(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET is_active = FALSE").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.BanUser(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_UnbanUser_Success(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET is_active = TRUE").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UnbanUser(ctx, "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_UnbanUser_Error(t *testing.T) {
	repo, mock := setupUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET is_active = TRUE").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.UnbanUser(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}
