package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
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

func pgsodiumUserRepoEncrypted(t *testing.T) (*PgsodiumUserRepository, sqlmock.Sqlmock) {
	t.Helper()

	db.ResetPgsodiumKeyID()
	db.ResetTestEncryptionKey()
	db.SetPgsodiumKeyIDForTest(1)

	t.Cleanup(func() {
		db.ResetPgsodiumKeyID()
		db.ResetLastSetPgsodiumKeyIDForTest()
	})

	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)

	return NewPgsodiumUserRepository(dbConn), mock
}

type pgsodiumUserValueConverter struct{}

func (pgsodiumUserValueConverter) ConvertValue(v interface{}) (driver.Value, error) {
	if strSlice, ok := v.([]string); ok {
		return strSlice, nil
	}

	return driver.DefaultParameterConverter.ConvertValue(v)
}

func pgsodiumUserRepoWithRows(t *testing.T) (*PgsodiumUserRepository, sqlmock.Sqlmock) {
	t.Helper()

	db.ResetPgsodiumKeyID()
	db.ResetTestEncryptionKey()

	t.Cleanup(func() {
		db.ResetPgsodiumKeyID()
		db.ResetLastSetPgsodiumKeyIDForTest()
	})

	dbConn, mock, err := sqlmock.New(sqlmock.ValueConverterOption(pgsodiumUserValueConverter{}))
	require.NoError(t, err)

	return NewPgsodiumUserRepository(dbConn), mock
}

func pgsodiumUserIDRow(userID string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "email_hash", "password_hash", "full_name_hash", "role", "email_verified", "created_at", "updated_at"}).
		AddRow(userID, "email_hash", "hash", "full_name_hash", "user", true, time.Now(), time.Now())
}

func TestPgsodiumUserRepo_CreateWithInvite_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO users").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.CreateWithInvite(ctx, "user-1", "test@example.com", "email_hash", "hash", "Test User", "coach")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_CreateWithInvite_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO users").
		WillReturnError(assert.AnError)

	err := repo.CreateWithInvite(ctx, "user-1", "test@example.com", "email_hash", "hash", "Test User", "coach")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_CreateWithInvite_DuplicateEmail(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO users").
		WillReturnError(&pq.Error{Code: "23505"})

	err := repo.CreateWithInvite(ctx, "user-1", "test@example.com", "email_hash", "hash", "Test User", "coach")

	require.Error(t, err)
	assert.True(t, apperrors.IsConflict(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetByID_DecryptSuccess(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs("user-1").
		WillReturnRows(pgsodiumUserIDRow("user-1"))

	mock.ExpectQuery("convert_from").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("decrypted@example.com"))

	mock.ExpectQuery("full_name_encrypted").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"full_name"}).AddRow("Decrypted Name"))

	result, err := repo.GetByID(ctx, "user-1")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "decrypted@example.com", result.Email)
	assert.Equal(t, "Decrypted Name", result.FullName)
	assert.Equal(t, "user-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetByID_DecryptEmailError(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs("user-1").
		WillReturnRows(pgsodiumUserIDRow("user-1"))

	mock.ExpectQuery("convert_from").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	result, err := repo.GetByID(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetByID_DecryptFullNameError(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs("user-1").
		WillReturnRows(pgsodiumUserIDRow("user-1"))

	mock.ExpectQuery("convert_from").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("decrypted@example.com"))

	mock.ExpectQuery("full_name_encrypted").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	result, err := repo.GetByID(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetByEmail_DecryptSuccess(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	emailHash := db.BlindIndex("test@example.com")

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name_hash", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "encrypted_email", "hash", "full_name_hash", "user", true, time.Now(), time.Now())

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs(emailHash).
		WillReturnRows(rows)

	mock.ExpectQuery("convert_from").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("decrypted@example.com"))

	mock.ExpectQuery("convert_from").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"full_name"}).AddRow("Decrypted Name"))

	result, err := repo.GetByEmail(ctx, "test@example.com")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "user-1", result.ID)
	assert.Equal(t, "decrypted@example.com", result.Email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetByEmail_DecryptError(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	emailHash := db.BlindIndex("test@example.com")

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name_hash", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "encrypted_email", "hash", "full_name_hash", "user", true, time.Now(), time.Now())

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs(emailHash).
		WillReturnRows(rows)

	mock.ExpectQuery("convert_from").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	result, err := repo.GetByEmail(ctx, "test@example.com")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetByEmailHash_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name_hash", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "test@example.com", "hash", "full_name_hash", "user", true, time.Now(), time.Now())

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs("email_hash").
		WillReturnRows(rows)

	result, err := repo.GetByEmailHash(ctx, "email_hash")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "user-1", result.ID)
	assert.Equal(t, "test@example.com", result.Email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetByEmailHash_NotFound(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs("email_hash").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByEmailHash(ctx, "email_hash")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetByEmailHash_QueryError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs("email_hash").
		WillReturnError(assert.AnError)

	result, err := repo.GetByEmailHash(ctx, "email_hash")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetByEmailHash_DecryptSuccess(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name_hash", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "encrypted_email", "hash", "full_name_hash", "user", true, time.Now(), time.Now())

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs("email_hash").
		WillReturnRows(rows)

	mock.ExpectQuery("convert_from").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("decrypted@example.com"))

	mock.ExpectQuery("convert_from").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"full_name"}).AddRow("Decrypted Name"))

	result, err := repo.GetByEmailHash(ctx, "email_hash")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "decrypted@example.com", result.Email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetByEmailHash_DecryptError(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name_hash", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "encrypted_email", "hash", "full_name_hash", "user", true, time.Now(), time.Now())

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs("email_hash").
		WillReturnRows(rows)

	mock.ExpectQuery("convert_from").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	result, err := repo.GetByEmailHash(ctx, "email_hash")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetEmailByID_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"email", "totp_enabled"}).
		AddRow("test@example.com", true)

	mock.ExpectQuery("SELECT email, totp_enabled").
		WithArgs("user-1").
		WillReturnRows(rows)

	email, totpEnabled, err := repo.GetEmailByID(ctx, "user-1")

	require.NoError(t, err)
	assert.Equal(t, "test@example.com", email)
	assert.True(t, totpEnabled)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetEmailByID_NotFound(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT email, totp_enabled").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	email, totpEnabled, err := repo.GetEmailByID(ctx, "user-1")

	require.Error(t, err)
	assert.Empty(t, email)
	assert.False(t, totpEnabled)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetEmailByID_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT email, totp_enabled").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	email, totpEnabled, err := repo.GetEmailByID(ctx, "user-1")

	require.Error(t, err)
	assert.Empty(t, email)
	assert.False(t, totpEnabled)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetEmailByID_DecryptSuccess(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"email", "totp_enabled"}).
		AddRow("decrypted@example.com", true)

	mock.ExpectQuery("totp_enabled FROM users WHERE id").
		WithArgs("user-1").
		WillReturnRows(rows)

	email, totpEnabled, err := repo.GetEmailByID(ctx, "user-1")

	require.NoError(t, err)
	assert.Equal(t, "decrypted@example.com", email)
	assert.True(t, totpEnabled)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetEmailByID_DecryptNotFound(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	mock.ExpectQuery("totp_enabled FROM users WHERE id").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	email, totpEnabled, err := repo.GetEmailByID(ctx, "user-1")

	require.Error(t, err)
	assert.Empty(t, email)
	assert.False(t, totpEnabled)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetEmailByID_DecryptError(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	mock.ExpectQuery("totp_enabled FROM users WHERE id").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	email, totpEnabled, err := repo.GetEmailByID(ctx, "user-1")

	require.Error(t, err)
	assert.Empty(t, email)
	assert.False(t, totpEnabled)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetClaimsByID_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"email", "role", "totp_enabled", "totp_backup_codes_remaining"}).
		AddRow("test@example.com", "admin", true, 3)

	mock.ExpectQuery("SELECT email, role, totp_enabled").
		WithArgs("user-1").
		WillReturnRows(rows)

	email, role, totpEnabled, remaining, err := repo.GetClaimsByID(ctx, "user-1")

	require.NoError(t, err)
	assert.Equal(t, "test@example.com", email)
	assert.Equal(t, "admin", role)
	assert.True(t, totpEnabled)
	assert.Equal(t, int32(3), remaining)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetClaimsByID_NotFound(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT email, role, totp_enabled").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	email, role, totpEnabled, remaining, err := repo.GetClaimsByID(ctx, "user-1")

	require.Error(t, err)
	assert.Empty(t, email)
	assert.Empty(t, role)
	assert.False(t, totpEnabled)
	assert.Equal(t, int32(0), remaining)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetClaimsByID_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT email, role, totp_enabled").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	email, role, totpEnabled, remaining, err := repo.GetClaimsByID(ctx, "user-1")

	require.Error(t, err)
	assert.Empty(t, email)
	assert.Empty(t, role)
	assert.False(t, totpEnabled)
	assert.Equal(t, int32(0), remaining)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetClaimsByID_DecryptSuccess(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"email", "role", "totp_enabled", "totp_backup_codes_remaining"}).
		AddRow("decrypted@example.com", "admin", false, 0)

	mock.ExpectQuery("role, totp_enabled").
		WithArgs("user-1").
		WillReturnRows(rows)

	email, role, totpEnabled, remaining, err := repo.GetClaimsByID(ctx, "user-1")

	require.NoError(t, err)
	assert.Equal(t, "decrypted@example.com", email)
	assert.Equal(t, "admin", role)
	assert.False(t, totpEnabled)
	assert.Equal(t, int32(0), remaining)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetClaimsByID_DecryptNotFound(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	mock.ExpectQuery("role, totp_enabled").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	_, _, _, _, err := repo.GetClaimsByID(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetClaimsByID_DecryptError(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	mock.ExpectQuery("role, totp_enabled").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	_, _, _, _, err := repo.GetClaimsByID(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetProfileWithPgsodium_Success(t *testing.T) {
	repo, mock := pgsodiumUserRepoWithRows(t)

	ctx := context.Background()

	now := time.Now()

	rows := mock.NewRows([]string{"id", "email", "full_name", "nickname", "profile_photo_url", "role", "age", "gender", "height_cm", "weight_kg", "fitness_level", "goals", "nutrition", "sleep_hours", "created_at", "updated_at"}).
		AddRow("user-1", "test@example.com", "Test User", "nick", "photo.jpg", "user", 30, "female", 170, 65.5, "active", []string{"lose_weight", "run"}, "balanced", 8.5, now, now)

	mock.ExpectQuery("user_profiles_with_goals").
		WithArgs("user-1").
		WillReturnRows(rows)

	profile, err := repo.GetProfileWithPgsodium(ctx, "user-1")

	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "user-1", profile.ID)
	assert.Equal(t, "test@example.com", profile.Email)
	assert.Equal(t, "nick", profile.Nickname)
	assert.Equal(t, "photo.jpg", profile.ProfilePhotoURL)
	assert.Equal(t, int32(30), profile.Age)
	assert.Equal(t, "female", profile.Gender)
	assert.Equal(t, int32(170), profile.HeightCm)
	assert.InDelta(t, 65.5, profile.WeightKg, 0.001)
	assert.Equal(t, "active", profile.FitnessLevel)
	assert.Equal(t, []string{"lose_weight", "run"}, profile.Goals)
	assert.Equal(t, "balanced", profile.Nutrition)
	assert.InDelta(t, 8.5, float64(profile.SleepHours), 0.001)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetProfileWithPgsodium_NullableFields(t *testing.T) {
	repo, mock := pgsodiumUserRepoWithRows(t)

	ctx := context.Background()

	now := time.Now()

	rows := mock.NewRows([]string{"id", "email", "full_name", "nickname", "profile_photo_url", "role", "age", "gender", "height_cm", "weight_kg", "fitness_level", "goals", "nutrition", "sleep_hours", "created_at", "updated_at"}).
		AddRow("user-1", "test@example.com", "Test User", nil, nil, "user", nil, nil, nil, nil, nil, []string{}, nil, nil, now, now)

	mock.ExpectQuery("user_profiles_with_goals").
		WithArgs("user-1").
		WillReturnRows(rows)

	profile, err := repo.GetProfileWithPgsodium(ctx, "user-1")

	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Empty(t, profile.Nickname)
	assert.Empty(t, profile.ProfilePhotoURL)
	assert.Empty(t, profile.Goals)
	assert.Equal(t, int32(0), profile.Age)
	assert.Empty(t, profile.Gender)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetProfileWithPgsodium_NotFound(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("user_profiles_with_goals").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	profile, err := repo.GetProfileWithPgsodium(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, profile)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetProfileWithPgsodium_QueryError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("user_profiles_with_goals").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	profile, err := repo.GetProfileWithPgsodium(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, profile)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetProfileWithPgsodium_ScanError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "full_name", "nickname", "profile_photo_url", "role", "age", "gender", "height_cm", "weight_kg", "fitness_level", "goals", "nutrition", "sleep_hours", "created_at", "updated_at"}).
		AddRow(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("user_profiles_with_goals").
		WithArgs("user-1").
		WillReturnRows(rows)

	profile, err := repo.GetProfileWithPgsodium(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, profile)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetProfileWithPgsodium_Encrypted(t *testing.T) {
	repo, mock := pgsodiumUserRepoWithRows(t)

	ctx := context.Background()

	db.SetPgsodiumKeyIDForTest(1)

	now := time.Now()

	rows := mock.NewRows([]string{"id", "email", "full_name", "nickname", "profile_photo_url", "role", "age", "gender", "height_cm", "weight_kg", "fitness_level", "goals", "nutrition", "sleep_hours", "created_at", "updated_at"}).
		AddRow("user-1", "decrypted@example.com", "Test User", "nick", "photo.jpg", "user", 30, "female", 170, 65.5, "active", []string{"lose_weight"}, "balanced", 8.5, now, now)

	mock.ExpectQuery("user_profiles_with_goals").
		WithArgs("user-1").
		WillReturnRows(rows)

	profile, err := repo.GetProfileWithPgsodium(ctx, "user-1")

	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "decrypted@example.com", profile.Email)
	assert.Equal(t, "nick", profile.Nickname)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ProfileSelectQuery(t *testing.T) {
	repo, _ := setupPgsodiumUserRepo(t)

	assert.Contains(t, repo.profileSelectQuery(), "user_profiles_with_goals")
	assert.Contains(t, repo.profileSelectQuery(), "u.email")

	repoEnc, _ := pgsodiumUserRepoEncrypted(t)

	encryptedQuery := repoEnc.profileSelectQuery()

	assert.Contains(t, encryptedQuery, "user_profiles_with_goals")
	assert.Contains(t, encryptedQuery, "convert_from(pgsodium.crypto_aead_aegis256_decrypt")
}

func TestPgsodiumUserRepo_UpdateUserDetails_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateUserDetails(ctx, "user-1", "New Name", "new_nick")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_UpdateUserDetails_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users").
		WillReturnError(assert.AnError)

	err := repo.UpdateUserDetails(ctx, "user-1", "New Name", "new_nick")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_UpdateNicknameWithPgsodium_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateNicknameWithPgsodium(ctx, "user-1", "nick", []byte("nonce"), "nick_hash")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_UpdateNicknameWithPgsodium_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users").
		WillReturnError(assert.AnError)

	err := repo.UpdateNicknameWithPgsodium(ctx, "user-1", "nick", []byte("nonce"), "nick_hash")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_BanUser_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET is_active = FALSE").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.BanUser(ctx, "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_BanUser_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET is_active = FALSE").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.BanUser(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_UnbanUser_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET is_active = TRUE").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UnbanUser(ctx, "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_UnbanUser_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET is_active = TRUE").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.UnbanUser(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_List_Empty(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "full_name", "role", "email_verified", "created_at", "updated_at"})

	mock.ExpectQuery("SELECT u.id").
		WithArgs(10, 10).
		WillReturnRows(rows)

	result, err := repo.List(ctx, 2, 10)

	require.NoError(t, err)
	assert.Empty(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_List_ScanError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow(nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT u.id").
		WithArgs(10, 0).
		WillReturnRows(rows)

	result, err := repo.List(ctx, 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_List_RowsError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "test@example.com", "Test User", "user", true, time.Now(), time.Now())

	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT u.id").
		WithArgs(10, 0).
		WillReturnRows(rows)

	result, err := repo.List(ctx, 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ListByRole_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "email", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "admin@example.com", "Admin", "admin", true, now, now)

	mock.ExpectQuery("WHERE u.role").
		WithArgs("admin", 10, 0).
		WillReturnRows(rows)

	mock.ExpectQuery("SELECT COUNT").
		WithArgs("admin").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	result, total, err := repo.ListByRole(ctx, "admin", 1, 10)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "admin@example.com", result[0].Email)
	assert.Equal(t, 1, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ListByRole_Empty(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "full_name", "role", "email_verified", "created_at", "updated_at"})

	mock.ExpectQuery("WHERE u.role").
		WithArgs("user", 5, 5).
		WillReturnRows(rows)

	mock.ExpectQuery("SELECT COUNT").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	result, total, err := repo.ListByRole(ctx, "user", 2, 5)

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 0, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ListByRole_QueryError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("WHERE u.role").
		WithArgs("admin", 10, 0).
		WillReturnError(assert.AnError)

	result, total, err := repo.ListByRole(ctx, "admin", 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ListByRole_ScanError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow(nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("WHERE u.role").
		WithArgs("admin", 10, 0).
		WillReturnRows(rows)

	result, total, err := repo.ListByRole(ctx, "admin", 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ListByRole_RowsError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "admin@example.com", "Admin", "admin", true, time.Now(), time.Now())

	rows.CloseError(assert.AnError)

	mock.ExpectQuery("WHERE u.role").
		WithArgs("admin", 10, 0).
		WillReturnRows(rows)

	result, total, err := repo.ListByRole(ctx, "admin", 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ListByRole_CountError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email", "full_name", "role", "email_verified", "created_at", "updated_at"}).
		AddRow("user-1", "admin@example.com", "Admin", "admin", true, time.Now(), time.Now())

	mock.ExpectQuery("WHERE u.role").
		WithArgs("admin", 10, 0).
		WillReturnRows(rows)

	mock.ExpectQuery("SELECT COUNT").
		WithArgs("admin").
		WillReturnError(assert.AnError)

	result, total, err := repo.ListByRole(ctx, "admin", 1, 10)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ConfirmEmail_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"user_id", "email", "used", "expires_at"}).
		AddRow("user-1", "test@example.com", false, time.Now().Add(time.Hour))

	mock.ExpectQuery("SELECT user_id").
		WithArgs("token-1").
		WillReturnRows(rows)

	err := repo.ConfirmEmail(ctx, "token-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ConfirmEmailWithPgsodium_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"user_id", "email", "used", "expires_at"}).
		AddRow("user-1", "test@example.com", false, time.Now().Add(time.Hour))

	mock.ExpectQuery("SELECT user_id").
		WithArgs("token-1").
		WillReturnRows(rows)

	userID, email, err := repo.ConfirmEmailWithPgsodium(ctx, "token-1")

	require.NoError(t, err)
	assert.Equal(t, "user-1", userID)
	assert.Equal(t, "test@example.com", email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ConfirmEmailWithPgsodium_NullExpiry(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"user_id", "email", "used", "expires_at"}).
		AddRow("user-1", "test@example.com", false, nil)

	mock.ExpectQuery("SELECT user_id").
		WithArgs("token-1").
		WillReturnRows(rows)

	userID, email, err := repo.ConfirmEmailWithPgsodium(ctx, "token-1")

	require.NoError(t, err)
	assert.Equal(t, "user-1", userID)
	assert.Equal(t, "test@example.com", email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ConfirmEmailWithPgsodium_NotFound(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT user_id").
		WithArgs("token-1").
		WillReturnError(sql.ErrNoRows)

	userID, email, err := repo.ConfirmEmailWithPgsodium(ctx, "token-1")

	require.Error(t, err)
	assert.Empty(t, userID)
	assert.Empty(t, email)
	assert.True(t, apperrors.Code(err) == "INVALID_ARGUMENT")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ConfirmEmailWithPgsodium_QueryError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT user_id").
		WithArgs("token-1").
		WillReturnError(assert.AnError)

	_, _, err := repo.ConfirmEmailWithPgsodium(ctx, "token-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ConfirmEmailWithPgsodium_AlreadyUsed(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"user_id", "email", "used", "expires_at"}).
		AddRow("user-1", "test@example.com", true, time.Now().Add(time.Hour))

	mock.ExpectQuery("SELECT user_id").
		WithArgs("token-1").
		WillReturnRows(rows)

	_, _, err := repo.ConfirmEmailWithPgsodium(ctx, "token-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INVALID_ARGUMENT")
	assert.Contains(t, apperrors.Message(err), "already been used")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ConfirmEmailWithPgsodium_Expired(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"user_id", "email", "used", "expires_at"}).
		AddRow("user-1", "test@example.com", false, time.Now().Add(-time.Hour))

	mock.ExpectQuery("SELECT user_id").
		WithArgs("token-1").
		WillReturnRows(rows)

	_, _, err := repo.ConfirmEmailWithPgsodium(ctx, "token-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INVALID_ARGUMENT")
	assert.Contains(t, apperrors.Message(err), "expired")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_CreateEmailVerification_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO email_verifications").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.CreateEmailVerification(ctx, "user-1", "test@example.com", "email_hash", "token-1", []byte("email_nonce"), []byte("token_nonce"))

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_CreateEmailVerification_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO email_verifications").
		WillReturnError(assert.AnError)

	err := repo.CreateEmailVerification(ctx, "user-1", "test@example.com", "email_hash", "token-1", []byte("email_nonce"), []byte("token_nonce"))

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_LoginWithPgsodium_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	emailHash := db.BlindIndex("test@example.com")

	mock.ExpectQuery("SELECT email_confirmed").
		WithArgs(emailHash).
		WillReturnRows(sqlmock.NewRows([]string{"email_confirmed"}).AddRow(true))

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "role"}).
		AddRow("user-1", "test@example.com", "hash", "user")

	mock.ExpectQuery("password_hash, role FROM users WHERE email_hash").
		WithArgs(emailHash).
		WillReturnRows(rows)

	user, err := repo.LoginWithPgsodium(ctx, "test@example.com")

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "user-1", user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "hash", user.PasswordHash)
	assert.Equal(t, "user", user.Role)
	assert.True(t, user.EmailVerified)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_LoginWithPgsodium_NotFound(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	emailHash := db.BlindIndex("test@example.com")

	mock.ExpectQuery("SELECT email_confirmed").
		WithArgs(emailHash).
		WillReturnError(sql.ErrNoRows)

	user, err := repo.LoginWithPgsodium(ctx, "test@example.com")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_LoginWithPgsodium_QueryError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	emailHash := db.BlindIndex("test@example.com")

	mock.ExpectQuery("SELECT email_confirmed").
		WithArgs(emailHash).
		WillReturnError(assert.AnError)

	user, err := repo.LoginWithPgsodium(ctx, "test@example.com")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_LoginWithPgsodium_EmailNotConfirmed(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	emailHash := db.BlindIndex("test@example.com")

	mock.ExpectQuery("SELECT email_confirmed").
		WithArgs(emailHash).
		WillReturnRows(sqlmock.NewRows([]string{"email_confirmed"}).AddRow(false))

	user, err := repo.LoginWithPgsodium(ctx, "test@example.com")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, apperrors.IsUnauthorized(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_LoginWithPgsodium_InvalidCredentials(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	emailHash := db.BlindIndex("test@example.com")

	mock.ExpectQuery("SELECT email_confirmed").
		WithArgs(emailHash).
		WillReturnRows(sqlmock.NewRows([]string{"email_confirmed"}).AddRow(true))

	mock.ExpectQuery("password_hash, role FROM users WHERE email_hash").
		WithArgs(emailHash).
		WillReturnError(sql.ErrNoRows)

	user, err := repo.LoginWithPgsodium(ctx, "test@example.com")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, apperrors.IsUnauthorized(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_LoginWithPgsodium_LoginQueryError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	emailHash := db.BlindIndex("test@example.com")

	mock.ExpectQuery("SELECT email_confirmed").
		WithArgs(emailHash).
		WillReturnRows(sqlmock.NewRows([]string{"email_confirmed"}).AddRow(true))

	mock.ExpectQuery("password_hash, role FROM users WHERE email_hash").
		WithArgs(emailHash).
		WillReturnError(assert.AnError)

	user, err := repo.LoginWithPgsodium(ctx, "test@example.com")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_CreateYandexUser_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO users").
		WillReturnResult(sqlmock.NewResult(0, 1))

	userID, role, emailConfirmed, err := repo.CreateYandexUser(ctx, "yandex-1", "email_hash", "test@example.com", "Test User", "nick", "nick_hash")

	require.NoError(t, err)
	assert.NotEmpty(t, userID)
	assert.Equal(t, "client", role)
	assert.True(t, emailConfirmed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_CreateYandexUser_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO users").
		WillReturnError(assert.AnError)

	userID, role, emailConfirmed, err := repo.CreateYandexUser(ctx, "yandex-1", "email_hash", "test@example.com", "Test User", "nick", "nick_hash")

	require.Error(t, err)
	assert.Empty(t, userID)
	assert.Empty(t, role)
	assert.False(t, emailConfirmed)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_CreateYandexUser_Duplicate(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO users").
		WillReturnError(&pq.Error{Code: "23505"})

	_, _, _, err := repo.CreateYandexUser(ctx, "yandex-1", "email_hash", "test@example.com", "Test User", "nick", "nick_hash")

	require.Error(t, err)
	assert.True(t, apperrors.IsConflict(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetPasswordHash_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT password_hash FROM users WHERE id").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow("hash"))

	passwordHash, err := repo.GetPasswordHash(ctx, "user-1")

	require.NoError(t, err)
	assert.Equal(t, "hash", passwordHash)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetPasswordHash_NotFound(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT password_hash FROM users WHERE id").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	passwordHash, err := repo.GetPasswordHash(ctx, "user-1")

	require.Error(t, err)
	assert.Empty(t, passwordHash)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetPasswordHash_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT password_hash FROM users WHERE id").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	passwordHash, err := repo.GetPasswordHash(ctx, "user-1")

	require.Error(t, err)
	assert.Empty(t, passwordHash)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_NicknameExists_True(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("nick", "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := repo.NicknameExists(ctx, "nick", "user-1")

	require.NoError(t, err)
	assert.True(t, exists)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_NicknameExists_False(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("nick", "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := repo.NicknameExists(ctx, "nick", "user-1")

	require.NoError(t, err)
	assert.False(t, exists)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_NicknameExists_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("nick", "user-1").
		WillReturnError(assert.AnError)

	exists, err := repo.NicknameExists(ctx, "nick", "user-1")

	require.Error(t, err)
	assert.False(t, exists)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_FindYandexUser_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email_hash", "password_hash", "full_name_hash", "role", "email_confirmed", "created_at", "updated_at"}).
		AddRow("user-1", "email_hash", "hash", "full_name_hash", "client", true, time.Now(), time.Now())

	mock.ExpectQuery("external_id").
		WithArgs("yandex-1").
		WillReturnRows(rows)

	user, err := repo.FindYandexUser(ctx, "yandex-1")

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "user-1", user.ID)
	assert.Equal(t, "client", user.Role)
	assert.True(t, user.EmailVerified)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_FindYandexUser_NotFound(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("external_id").
		WithArgs("yandex-1").
		WillReturnError(sql.ErrNoRows)

	user, err := repo.FindYandexUser(ctx, "yandex-1")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_FindYandexUser_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("external_id").
		WithArgs("yandex-1").
		WillReturnError(assert.AnError)

	user, err := repo.FindYandexUser(ctx, "yandex-1")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_FindYandexUser_DecryptSuccess(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email_hash", "password_hash", "full_name_hash", "role", "email_confirmed", "created_at", "updated_at"}).
		AddRow("user-1", "email_hash", "hash", "full_name_hash", "client", true, time.Now(), time.Now())

	mock.ExpectQuery("external_id").
		WithArgs("yandex-1").
		WillReturnRows(rows)

	mock.ExpectQuery("convert_from").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("decrypted@example.com"))

	mock.ExpectQuery("full_name_encrypted").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"full_name"}).AddRow("Decrypted Name"))

	user, err := repo.FindYandexUser(ctx, "yandex-1")

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "decrypted@example.com", user.Email)
	assert.Equal(t, "Decrypted Name", user.FullName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_LinkYandexAccount_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("provider = 'yandex'").
		WithArgs("yandex-1", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.LinkYandexAccount(ctx, "yandex-1", "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_LinkYandexAccount_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("provider = 'yandex'").
		WithArgs("yandex-1", "user-1").
		WillReturnError(assert.AnError)

	err := repo.LinkYandexAccount(ctx, "yandex-1", "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_MarkEmailVerified_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET email_verified").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.MarkEmailVerified(ctx, "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_MarkEmailVerified_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET email_verified").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.MarkEmailVerified(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_CreateUserProfile_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_profiles").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.CreateUserProfile(ctx, "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_CreateUserProfile_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_profiles").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.CreateUserProfile(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetUserRole_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT role FROM users WHERE id").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("admin"))

	role, err := repo.GetUserRole(ctx, "user-1")

	require.NoError(t, err)
	assert.Equal(t, "admin", role)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetUserRole_NotFound(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT role FROM users WHERE id").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	role, err := repo.GetUserRole(ctx, "user-1")

	require.Error(t, err)
	assert.Empty(t, role)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetUserRole_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT role FROM users WHERE id").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	role, err := repo.GetUserRole(ctx, "user-1")

	require.Error(t, err)
	assert.Empty(t, role)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_UpdatePassword_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET password_hash").
		WithArgs("new_hash", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdatePassword(ctx, "user-1", "new_hash")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_UpdatePassword_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET password_hash").
		WithArgs("new_hash", "user-1").
		WillReturnError(assert.AnError)

	err := repo.UpdatePassword(ctx, "user-1", "new_hash")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_UpdateProfilePhoto_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET profile_photo_url = ").
		WithArgs("photo.jpg", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateProfilePhoto(ctx, "user-1", "photo.jpg")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_UpdateProfilePhoto_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET profile_photo_url = ").
		WithArgs("photo.jpg", "user-1").
		WillReturnError(assert.AnError)

	err := repo.UpdateProfilePhoto(ctx, "user-1", "photo.jpg")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_UpdateProfilePhoto_Encrypted(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	mock.ExpectExec("profile_photo_url_encrypted").
		WithArgs("photo.jpg", sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateProfilePhoto(ctx, "user-1", "photo.jpg")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_UpdateProfilePhoto_EncryptedError(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	mock.ExpectExec("profile_photo_url_encrypted").
		WithArgs("photo.jpg", sqlmock.AnyArg(), "user-1").
		WillReturnError(assert.AnError)

	err := repo.UpdateProfilePhoto(ctx, "user-1", "photo.jpg")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_UpdateProfilePhoto_EncryptedEmptyURL(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET profile_photo_url = ").
		WithArgs("", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateProfilePhoto(ctx, "user-1", "")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_RemoveProfilePhoto_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET profile_photo_url = NULL").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RemoveProfilePhoto(ctx, "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_RemoveProfilePhoto_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("UPDATE users SET profile_photo_url = NULL").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.RemoveProfilePhoto(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_RemoveProfilePhoto_Encrypted(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	mock.ExpectExec("profile_photo_url_encrypted = NULL").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RemoveProfilePhoto(ctx, "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_RemoveProfilePhoto_EncryptedError(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	mock.ExpectExec("profile_photo_url_encrypted = NULL").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.RemoveProfilePhoto(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetTOTPState_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"totp_secret_encrypted", "totp_enabled", "totp_backup_codes_hash", "totp_backup_codes_remaining"}).
		AddRow([]byte("secret"), true, `{"hash1","hash2"}`, 2)

	mock.ExpectQuery("SELECT totp_secret_encrypted").
		WithArgs("user-1").
		WillReturnRows(rows)

	secret, enabled, codes, remaining, err := repo.GetTOTPState(ctx, "user-1")

	require.NoError(t, err)
	assert.Equal(t, []byte("secret"), secret)
	assert.True(t, enabled)
	assert.Equal(t, []string{"hash1", "hash2"}, codes)
	assert.Equal(t, int32(2), remaining)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetTOTPState_NotFound(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT totp_secret_encrypted").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	secret, enabled, codes, remaining, err := repo.GetTOTPState(ctx, "user-1")

	require.Error(t, err)
	assert.Nil(t, secret)
	assert.False(t, enabled)
	assert.Nil(t, codes)
	assert.Equal(t, int32(0), remaining)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_GetTOTPState_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectQuery("SELECT totp_secret_encrypted").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	_, _, _, _, err := repo.GetTOTPState(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_EnableTOTP_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("SET totp_secret_encrypted").
		WithArgs([]byte("secret"), sqlmock.AnyArg(), 5, "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.EnableTOTP(ctx, "user-1", []byte("secret"), []string{"hash1"}, 5)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_EnableTOTP_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("SET totp_secret_encrypted").
		WithArgs([]byte("secret"), sqlmock.AnyArg(), 5, "user-1").
		WillReturnError(assert.AnError)

	err := repo.EnableTOTP(ctx, "user-1", []byte("secret"), []string{"hash1"}, 5)

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_RemoveBackupCode_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("array_remove").
		WithArgs("hash1", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RemoveBackupCode(ctx, "user-1", "hash1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_RemoveBackupCode_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("array_remove").
		WithArgs("hash1", "user-1").
		WillReturnError(assert.AnError)

	err := repo.RemoveBackupCode(ctx, "user-1", "hash1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_DisableTOTP_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("totp_secret_encrypted = NULL").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DisableTOTP(ctx, "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_DisableTOTP_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("totp_secret_encrypted = NULL").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.DisableTOTP(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_DeleteProfileData_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("email_hash = NULL").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteProfileData(ctx, "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_DeleteProfileData_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("email_hash = NULL").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.DeleteProfileData(ctx, "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ReplaceUserGoals_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_goals").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("INSERT INTO user_goals").
		WithArgs("user-1", "goal-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("INSERT INTO user_goals").
		WithArgs("user-1", "goal-2").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.ReplaceUserGoals(ctx, "user-1", []string{"goal-1", "goal-2"})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ReplaceUserGoals_DeleteError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_goals").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.ReplaceUserGoals(ctx, "user-1", []string{"goal-1"})

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ReplaceUserGoals_InsertError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_goals").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("INSERT INTO user_goals").
		WithArgs("user-1", "goal-1").
		WillReturnError(assert.AnError)

	err := repo.ReplaceUserGoals(ctx, "user-1", []string{"goal-1"})

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ReplaceUserContraindications_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_contraindications").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("INSERT INTO user_contraindications").
		WithArgs("user-1", "asthma").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.ReplaceUserContraindications(ctx, "user-1", []string{"asthma"})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ReplaceUserContraindications_DeleteError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_contraindications").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.ReplaceUserContraindications(ctx, "user-1", []string{"asthma"})

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ReplaceUserContraindications_InsertError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_contraindications").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("INSERT INTO user_contraindications").
		WithArgs("user-1", "asthma").
		WillReturnError(assert.AnError)

	err := repo.ReplaceUserContraindications(ctx, "user-1", []string{"asthma"})

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_DeleteHealthCondition_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_health_conditions").
		WithArgs("hc-1", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteHealthCondition(ctx, "hc-1", "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_DeleteHealthCondition_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_health_conditions").
		WithArgs("hc-1", "user-1").
		WillReturnError(assert.AnError)

	err := repo.DeleteHealthCondition(ctx, "hc-1", "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_DeleteMenstrualCycle_Success(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_menstrual_cycles").
		WithArgs("mc-1", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteMenstrualCycle(ctx, "mc-1", "user-1")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_DeleteMenstrualCycle_Error(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	mock.ExpectExec("DELETE FROM user_menstrual_cycles").
		WithArgs("mc-1", "user-1").
		WillReturnError(assert.AnError)

	err := repo.DeleteMenstrualCycle(ctx, "mc-1", "user-1")

	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_BuildUserInsertQuery(t *testing.T) {
	query, args := buildUserInsertQuery(userInsertData{
		userID:         "user-1",
		email:          "test@example.com",
		emailHash:      "email_hash",
		passwordHash:   "hash",
		fullName:       "Test User",
		emailNonce:     []byte("email_nonce"),
		fullNameNonce:  []byte("full_name_nonce"),
		fullNameHash:   "full_name_hash",
		role:           "user",
		emailConfirmed: true,
	})

	assert.Contains(t, query, "INSERT INTO users (id, email_encrypted, email_nonce, email_hash, password_hash, full_name_encrypted, full_name_nonce, full_name_hash, role, email_confirmed, created_at, updated_at) VALUES ($1, ")
	assert.Contains(t, query, "NOW(), NOW())")

	expectedArgs := []interface{}{"user-1", "test@example.com", []byte("email_nonce"), []byte("email_nonce"), "email_hash", "hash", "Test User", []byte("full_name_nonce"), []byte("full_name_nonce"), "full_name_hash", "user", true}
	assert.Equal(t, expectedArgs, args)
}

func TestPgsodiumUserRepo_BuildEmailVerificationInsertQuery(t *testing.T) {
	query, args := buildEmailVerificationInsertQuery("user-1", "test@example.com", "email_hash", "token-1", []byte("email_nonce"), []byte("token_nonce"))

	assert.Contains(t, query, "INSERT INTO email_verifications (user_id, email_encrypted, email_nonce, email_hash, token, token_encrypted, token_nonce, expires_at, used) VALUES ($1, ")
	assert.Contains(t, query, "NOW() + INTERVAL '24 hours', false)")

	expectedArgs := []interface{}{"user-1", "test@example.com", []byte("email_nonce"), []byte("email_nonce"), "email_hash", "token-1", []byte("token_nonce"), []byte("token_nonce")}
	assert.Equal(t, expectedArgs, args)
}

func TestPgsodiumUserRepo_BuildYandexUserInsertQuery(t *testing.T) {
	query, args := buildYandexUserInsertQuery(yandexUserInsertData{
		userID:        "user-1",
		emailVal:      "test@example.com",
		emailHash:     "email_hash",
		emailNonce:    []byte("email_nonce"),
		fullName:      "Test User",
		fullNameNonce: []byte("full_name_nonce"),
		nickname:      "nick",
		nicknameNonce: []byte("nickname_nonce"),
		nicknameHash:  "nick_hash",
		yandexID:      "yandex-1",
	})

	assert.Contains(t, query, "INSERT INTO users (id, email_encrypted, email_nonce, email_hash, password_hash, full_name_encrypted, full_name_nonce, full_name_hash, nickname_encrypted, nickname_nonce, nickname_hash, role, provider, external_id, email_confirmed, created_at, updated_at) VALUES ($1, ")
	assert.Contains(t, query, "'client', 'yandex', $13, true, NOW(), NOW())")

	expectedArgs := []interface{}{"user-1", "test@example.com", []byte("email_nonce"), []byte("email_nonce"), "email_hash", "Test User", []byte("full_name_nonce"), []byte("full_name_nonce"), "nick", []byte("nickname_nonce"), []byte("nickname_nonce"), "nick_hash", "yandex-1"}
	assert.Equal(t, expectedArgs, args)
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

func TestPgsodiumUserRepo_GetByEmail_QueryError(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	emailHash := db.BlindIndex("test@example.com")

	mock.ExpectQuery("SELECT id, email_hash").
		WithArgs(emailHash).
		WillReturnError(assert.AnError)

	result, err := repo.GetByEmail(ctx, "test@example.com")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_FindYandexUser_DecryptError(t *testing.T) {
	repo, mock := pgsodiumUserRepoEncrypted(t)

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "email_hash", "password_hash", "full_name_hash", "role", "email_confirmed", "created_at", "updated_at"}).
		AddRow("user-1", "email_hash", "hash", "full_name_hash", "client", true, time.Now(), time.Now())

	mock.ExpectQuery("external_id").
		WithArgs("yandex-1").
		WillReturnRows(rows)

	mock.ExpectQuery("convert_from").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	user, err := repo.FindYandexUser(ctx, "yandex-1")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgsodiumUserRepo_ListItems_UnsupportedTable(t *testing.T) {
	repo, mock := setupPgsodiumUserRepo(t)

	ctx := context.Background()

	err := repo.deleteListItems(ctx, "user-1", "unknown_table")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INVALID_ARGUMENT")

	err = repo.insertListItem(ctx, "user-1", "unknown_table", "value")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INVALID_ARGUMENT")

	require.NoError(t, mock.ExpectationsWereMet())
}
