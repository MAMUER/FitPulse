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

func setupExternalSecretsRepo(t *testing.T) (*externalSecretsRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewExternalSecretsRepository(db).(*externalSecretsRepository), mock
}

func TestExternalSecretsRepository_GetByName_Success(t *testing.T) {
	repo, mock := setupExternalSecretsRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "value", "created_at", "updated_at"}).
		AddRow("sec-1", "fitpulse/production/app-secrets/JWT_PRIVATE_KEY_PEM", "secret-value", now, now)

	mock.ExpectQuery("SELECT id").
		WithArgs("fitpulse/production/app-secrets/JWT_PRIVATE_KEY_PEM").
		WillReturnRows(rows)

	result, err := repo.GetByName(ctx, "fitpulse/production/app-secrets/JWT_PRIVATE_KEY_PEM")

	require.NoError(t, err)
	assert.Equal(t, "sec-1", result.ID)
	assert.Equal(t, "fitpulse/production/app-secrets/JWT_PRIVATE_KEY_PEM", result.Name)
	assert.Equal(t, "secret-value", result.Value)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExternalSecretsRepository_GetByName_NotFound(t *testing.T) {
	repo, mock := setupExternalSecretsRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByName(ctx, "missing")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "NOT_FOUND")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExternalSecretsRepository_List_Success(t *testing.T) {
	repo, mock := setupExternalSecretsRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "value", "created_at", "updated_at"}).
		AddRow("sec-1", "secret-a", "value-a", now, now).
		AddRow("sec-2", "secret-b", "value-b", now, now)

	mock.ExpectQuery("SELECT id").
		WillReturnRows(rows)

	result, err := repo.List(ctx)

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "secret-a", result[0].Name)
	assert.Equal(t, "secret-b", result[1].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExternalSecretsRepository_Upsert_Create(t *testing.T) {
	repo, mock := setupExternalSecretsRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO external_secrets").
		WithArgs("new-secret", "new-value").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Upsert(ctx, &port.ExternalSecret{Name: "new-secret", Value: "new-value"})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExternalSecretsRepository_Upsert_Update(t *testing.T) {
	repo, mock := setupExternalSecretsRepo(t)
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO external_secrets").
		WithArgs("existing-secret", "updated-value").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Upsert(ctx, &port.ExternalSecret{Name: "existing-secret", Value: "updated-value"})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExternalSecretsRepository_Delete_Success(t *testing.T) {
	repo, mock := setupExternalSecretsRepo(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM external_secrets").
		WithArgs("to-delete").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(ctx, "to-delete")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
