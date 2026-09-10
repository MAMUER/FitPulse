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
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
)

func setupBiometricRepo(t *testing.T) (port.BiometricRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return NewBiometricRepository(db), mock
}

func TestBiometricRepository_Create_Success(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()
	record := &entity.BiometricRecord{
		ID:         "bio-1",
		UserID:     "user-1",
		MetricType: "heart_rate",
		Value:      72.5,
		Timestamp:  time.Now(),
		DeviceType: "watch",
		Source:     "test",
	}

	rows := sqlmock.NewRows([]string{"id"}).AddRow("bio-1")
	mock.ExpectQuery("INSERT INTO biometric_data").
		WithArgs(record.ID, record.UserID, record.MetricType, record.Value, record.Timestamp, record.DeviceType, record.Source).
		WillReturnRows(rows)

	result, err := repo.Create(ctx, record)
	require.NoError(t, err)
	assert.Equal(t, "bio-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_Create_QueryError(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("INSERT INTO biometric_data").
		WillReturnError(assert.AnError)

	result, err := repo.Create(ctx, &entity.BiometricRecord{UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_BatchCreate_Success(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()
	now := time.Now()
	records := []*entity.BiometricRecord{
		{ID: "bio-1", UserID: "user-1", MetricType: "hr", Value: 70, Timestamp: now, DeviceType: "w1", Source: "s1"},
		{ID: "bio-2", UserID: "user-1", MetricType: "hr", Value: 71, Timestamp: now, DeviceType: "w2", Source: "s2"},
	}

	mock.ExpectBegin()
	for _, rec := range records {
		mock.ExpectExec("INSERT INTO biometric_data").
			WithArgs(rec.ID, rec.UserID, rec.MetricType, rec.Value, rec.Timestamp, rec.DeviceType, rec.Source).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

	inserted, err := repo.BatchCreate(ctx, records)
	require.NoError(t, err)
	assert.Equal(t, 2, inserted)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_BatchCreate_BeginTxError(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()

	mock.ExpectBegin().WillReturnError(assert.AnError)

	inserted, err := repo.BatchCreate(ctx, []*entity.BiometricRecord{{UserID: "user-1"}})
	require.Error(t, err)
	assert.Equal(t, 0, inserted)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_BatchCreate_ExecError(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO biometric_data").
		WillReturnError(assert.AnError)

	inserted, err := repo.BatchCreate(ctx, []*entity.BiometricRecord{{UserID: "user-1"}})
	require.Error(t, err)
	assert.Equal(t, 0, inserted)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_BatchCreate_CommitError(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO biometric_data").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(assert.AnError)

	inserted, err := repo.BatchCreate(ctx, []*entity.BiometricRecord{{UserID: "user-1"}})
	require.Error(t, err)
	assert.Equal(t, 0, inserted)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetByUserID_MetricTypeAndOffset(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type", "source", "created_at"}).
		AddRow("bio-1", "user-1", "hr", 72.0, now, "watch", "test", now)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1", "hr", 10, 5).
		WillReturnRows(rows)

	result, err := repo.GetByUserID(ctx, "user-1", "hr", 10, 5)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "bio-1", result[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetByUserID_MetricTypeOnly(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type", "source", "created_at"}).
		AddRow("bio-1", "user-1", "hr", 72.0, now, "watch", "test", now)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1", "hr", 10).
		WillReturnRows(rows)

	result, err := repo.GetByUserID(ctx, "user-1", "hr", 10, 0)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetByUserID_OffsetOnly(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type", "source", "created_at"}).
		AddRow("bio-1", "user-1", "hr", 72.0, now, "watch", "test", now)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1", 10, 5).
		WillReturnRows(rows)

	result, err := repo.GetByUserID(ctx, "user-1", "", 10, 5)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetByUserID_NoFilters(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type", "source", "created_at"}).
		AddRow("bio-1", "user-1", "hr", 72.0, now, "watch", "test", now)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1", 10).
		WillReturnRows(rows)

	result, err := repo.GetByUserID(ctx, "user-1", "", 10, 0)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetByUserID_QueryError(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").WillReturnError(assert.AnError)

	result, err := repo.GetByUserID(ctx, "user-1", "hr", 10, 0)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetByUserID_ScanError(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type", "source", "created_at"}).
		AddRow("bio-1", nil, nil, nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id").
		WillReturnRows(rows)

	result, err := repo.GetByUserID(ctx, "user-1", "hr", 10, 0)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetByUserID_RowsError(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type", "source", "created_at"}).
		AddRow("bio-1", "user-1", "hr", 72.0, now, "watch", "test", now)
	rows.CloseError(assert.AnError)

	mock.ExpectQuery("SELECT id").
		WillReturnRows(rows)

	result, err := repo.GetByUserID(ctx, "user-1", "hr", 10, 0)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_Delete_Success(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM biometric_data").
		WithArgs("bio-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, "bio-1")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_Delete_Error(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM biometric_data").
		WithArgs("bio-1").
		WillReturnError(assert.AnError)

	err := repo.Delete(ctx, "bio-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetLatest_Success(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type", "source", "created_at"}).
		AddRow("bio-1", "user-1", "hr", 72.0, now, "watch", "test", now)

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1", "hr").
		WillReturnRows(rows)

	result, err := repo.GetLatest(ctx, "user-1", "hr")
	require.NoError(t, err)
	assert.Equal(t, "bio-1", result.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetLatest_NotFound(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").
		WithArgs("user-1", "hr").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetLatest(ctx, "user-1", "hr")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetLatest_QueryError(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id").
		WillReturnError(assert.AnError)

	result, err := repo.GetLatest(ctx, "user-1", "hr")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_Update_Success(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()
	now := time.Now()
	record := &entity.BiometricRecord{
		ID:         "bio-1",
		UserID:     "user-1",
		MetricType: "hr",
		Value:      80.0,
		Timestamp:  now,
		DeviceType: "watch",
		Source:     "test",
	}

	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type", "source", "created_at"}).
		AddRow("bio-1", "user-1", "hr", 80.0, now, "watch", "test", now)

	mock.ExpectQuery("UPDATE biometric_data").
		WithArgs(record.Value, record.Timestamp, record.DeviceType, record.ID).
		WillReturnRows(rows)

	result, err := repo.Update(ctx, record)
	require.NoError(t, err)
	assert.Equal(t, "bio-1", result.ID)
	assert.Equal(t, 80.0, result.Value)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_Update_NotFound(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("UPDATE biometric_data").
		WillReturnError(sql.ErrNoRows)

	result, err := repo.Update(ctx, &entity.BiometricRecord{ID: "missing"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_Update_Error(t *testing.T) {
	repo, mock := setupBiometricRepo(t)
	ctx := context.Background()

	mock.ExpectQuery("UPDATE biometric_data").
		WillReturnError(assert.AnError)

	result, err := repo.Update(ctx, &entity.BiometricRecord{ID: "bio-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
	require.NoError(t, mock.ExpectationsWereMet())
}
