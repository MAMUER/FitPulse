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

func setupRefreshTokenRepo(t *testing.T) (*refreshTokenRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewRefreshTokenRepository(db).(*refreshTokenRepository), mock
}

func TestRefreshTokenRepository_GetValid_Success(t *testing.T) {
	repo, mock := setupRefreshTokenRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "token", "used", "expires_at", "created_at"}).
		AddRow("rt-1", "user-1", "token-1", false, now.Add(24*time.Hour), now)

	mock.ExpectQuery("SELECT id").
		WithArgs("token-1").
		WillReturnRows(rows)

	result, err := repo.GetValid(ctx, "token-1")
	require.NoError(t, err)
	assert.Equal(t, "rt-1", result.ID)
	assert.Equal(t, "user-1", result.UserID)
	assert.False(t, result.Used)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_GetValid_NotFound(t *testing.T) {
	repo, mock := setupRefreshTokenRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").
		WithArgs("token-1").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetValid(ctx, "token-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_GetValid_QueryError(t *testing.T) {
	repo, mock := setupRefreshTokenRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").WillReturnError(assert.AnError)

	result, err := repo.GetValid(ctx, "token-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_Create_Success(t *testing.T) {
	repo, mock := setupRefreshTokenRepo(t)
	ctx := context.Background()
	rt := &port.RefreshToken{
		UserID:    "user-1",
		Token:     "token-1",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	mock.ExpectExec("INSERT INTO refresh_tokens").
		WithArgs("user-1", "token-1", rt.ExpiresAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, rt)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_Create_Error(t *testing.T) {
	repo, mock := setupRefreshTokenRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO refresh_tokens").WillReturnError(assert.AnError)

	err := repo.Create(ctx, &port.RefreshToken{UserID: "user-1", Token: "token-1"})
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_MarkUsed_Success(t *testing.T) {
	repo, mock := setupRefreshTokenRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE refresh_tokens").
		WithArgs("token-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.MarkUsed(ctx, "token-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshTokenRepository_MarkUsed_Error(t *testing.T) {
	repo, mock := setupRefreshTokenRepo(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE refresh_tokens").WillReturnError(assert.AnError)

	err := repo.MarkUsed(ctx, "token-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}
