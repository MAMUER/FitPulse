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
	"github.com/MAMUER/project/internal/domain/port"
)

func setupEmailVerificationRepo(t *testing.T) (*emailVerificationRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewEmailVerificationRepository(db).(*emailVerificationRepository), mock
}

func TestEmailVerificationRepository_Create_Success(t *testing.T) {
	repo, mock := setupEmailVerificationRepo(t)
	ctx := context.Background()
	ev := &port.EmailVerification{
		UserID:    "user-1",
		Email:     "test@example.com",
		EmailHash: "hash",
		Token:     "token-1",
		Used:      false,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO email_verifications").
		WithArgs(ev.UserID, ev.Email, ev.EmailHash, ev.Token, ev.Used, ev.ExpiresAt, ev.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, ev)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEmailVerificationRepository_Create_Error(t *testing.T) {
	repo, mock := setupEmailVerificationRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO email_verifications").
		WillReturnError(assert.AnError)

	err := repo.Create(ctx, &port.EmailVerification{UserID: "user-1"})
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEmailVerificationRepository_GetValidToken_Success(t *testing.T) {
	repo, mock := setupEmailVerificationRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "email", "email_hash", "token", "used", "expires_at", "created_at"}).
		AddRow("ev-1", "user-1", "test@example.com", "hash", "token-1", false, now.Add(24*time.Hour), now)

	mock.ExpectQuery("SELECT id").
		WithArgs("token-1").
		WillReturnRows(rows)

	result, err := repo.GetValidToken(ctx, "token-1")
	require.NoError(t, err)
	assert.Equal(t, "ev-1", result.ID)
	assert.Equal(t, "user-1", result.UserID)
	assert.False(t, result.Used)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEmailVerificationRepository_GetValidToken_NotFound(t *testing.T) {
	repo, mock := setupEmailVerificationRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").
		WithArgs("token-1").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetValidToken(ctx, "token-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEmailVerificationRepository_GetValidToken_QueryError(t *testing.T) {
	repo, mock := setupEmailVerificationRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").WillReturnError(assert.AnError)

	result, err := repo.GetValidToken(ctx, "token-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEmailVerificationRepository_GetByUserID_Success(t *testing.T) {
	repo, mock := setupEmailVerificationRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "email", "email_hash", "token", "used", "expires_at", "created_at"}).
		AddRow("ev-1", "user-1", "test@example.com", "hash", "token-1", false, now.Add(24*time.Hour), now)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnRows(rows)

	result, err := repo.GetByUserID(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "ev-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEmailVerificationRepository_GetByUserID_NotFound(t *testing.T) {
	repo, mock := setupEmailVerificationRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByUserID(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEmailVerificationRepository_MarkUsed_Success(t *testing.T) {
	repo, mock := setupEmailVerificationRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE email_verifications").
		WithArgs("token-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.MarkUsed(ctx, "token-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEmailVerificationRepository_MarkUsed_Error(t *testing.T) {
	repo, mock := setupEmailVerificationRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE email_verifications").
		WithArgs("token-1").
		WillReturnError(assert.AnError)

	err := repo.MarkUsed(ctx, "token-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEmailVerificationRepository_MarkUserEmailVerified_Success(t *testing.T) {
	repo, mock := setupEmailVerificationRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE users").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.MarkUserEmailVerified(ctx, "user-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEmailVerificationRepository_MarkUserEmailVerified_Error(t *testing.T) {
	repo, mock := setupEmailVerificationRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE users").
		WithArgs("user-1").
		WillReturnError(assert.AnError)

	err := repo.MarkUserEmailVerified(ctx, "user-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}
