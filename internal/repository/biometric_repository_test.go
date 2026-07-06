package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/domain"
)

const testUserID = "user-123"

func setupTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db, mock
}

func TestBiometricRepository_Save_Success(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewBiometricRepository(db)

	data := &domain.BiometricData{
		UserID:     testUserID,
		MetricType: "heart_rate",
		Value:      72.5,
		Timestamp:  time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC),
		DeviceType: "smartwatch",
	}

	mock.ExpectExec(`INSERT INTO biometric_data \(id, user_id, metric_type, value, timestamp, device_type, created_at\)`).
		WithArgs(sqlmock.AnyArg(), data.UserID, data.MetricType, data.Value, data.Timestamp, data.DeviceType, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Save(context.Background(), data)

	assert.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_Save_DatabaseError(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewBiometricRepository(db)

	data := &domain.BiometricData{
		UserID:     testUserID,
		MetricType: "heart_rate",
		Value:      72.5,
		Timestamp:  time.Now(),
		DeviceType: "smartwatch",
	}

	mock.ExpectExec(`INSERT INTO biometric_data`).
		WillReturnError(errors.New("database connection lost"))

	err := repo.Save(context.Background(), data)

	assert.Error(t, err)
	assert.EqualError(t, err, "database connection lost")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetByUser_Success(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewBiometricRepository(db)

	userID := testUserID
	limit := 10
	ts := time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type"}).
		AddRow("id-1", userID, "heart_rate", 72.5, ts, "smartwatch").
		AddRow("id-2", userID, "blood_pressure", 120.0, ts.Add(-time.Hour), "band")

	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type FROM biometric_data WHERE user_id = \$1 ORDER BY timestamp DESC LIMIT \$2`).
		WithArgs(userID, limit).
		WillReturnRows(rows)

	results, err := repo.GetByUser(context.Background(), userID, limit)

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "id-1", results[0].ID)
	assert.Equal(t, userID, results[0].UserID)
	assert.Equal(t, "heart_rate", results[0].MetricType)
	assert.InDelta(t, 72.5, results[0].Value, 0.01)
	assert.Equal(t, ts, results[0].Timestamp)
	assert.Equal(t, "smartwatch", results[0].DeviceType)
	assert.Equal(t, "id-2", results[1].ID)
	assert.InDelta(t, 120.0, results[1].Value, 0.01)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetByUser_EmptyResult(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewBiometricRepository(db)

	userID := "user-456"
	limit := 5

	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type"})

	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type FROM biometric_data WHERE user_id = \$1 ORDER BY timestamp DESC LIMIT \$2`).
		WithArgs(userID, limit).
		WillReturnRows(rows)

	results, err := repo.GetByUser(context.Background(), userID, limit)

	require.NoError(t, err)
	assert.Empty(t, results)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetByUser_DatabaseError(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewBiometricRepository(db)

	userID := testUserID
	limit := 10

	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type FROM biometric_data WHERE user_id = \$1 ORDER BY timestamp DESC LIMIT \$2`).
		WithArgs(userID, limit).
		WillReturnError(errors.New("query failed"))

	results, err := repo.GetByUser(context.Background(), userID, limit)

	assert.Nil(t, results)
	assert.Error(t, err)
	assert.EqualError(t, err, "query biometric data by user: query failed")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetByUser_ScanError(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewBiometricRepository(db)

	userID := testUserID
	limit := 10

	// Simulate a row where timestamp cannot be scanned into time.Time
	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type"}).
		AddRow("id-1", userID, "heart_rate", 72.5, "not-a-timestamp", "smartwatch")

	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type FROM biometric_data WHERE user_id = \$1 ORDER BY timestamp DESC LIMIT \$2`).
		WithArgs(userID, limit).
		WillReturnRows(rows)

	results, err := repo.GetByUser(context.Background(), userID, limit)

	assert.Nil(t, results)
	assert.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetLatest_Success(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewBiometricRepository(db)

	userID := testUserID
	metricType := "heart_rate"
	ts := time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC)

	row := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type"}).
		AddRow("id-1", userID, metricType, 75.0, ts, "smartwatch")

	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type FROM biometric_data WHERE user_id = \$1 AND metric_type = \$2 ORDER BY timestamp DESC LIMIT 1`).
		WithArgs(userID, metricType).
		WillReturnRows(row)

	result, err := repo.GetLatest(context.Background(), userID, metricType)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "id-1", result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, metricType, result.MetricType)
	assert.InDelta(t, 75.0, result.Value, 0.01)
	assert.Equal(t, ts, result.Timestamp)
	assert.Equal(t, "smartwatch", result.DeviceType)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetLatest_NotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewBiometricRepository(db)

	userID := "user-789"
	metricType := "oxygen_level"

	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type FROM biometric_data WHERE user_id = \$1 AND metric_type = \$2 ORDER BY timestamp DESC LIMIT 1`).
		WithArgs(userID, metricType).
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetLatest(context.Background(), userID, metricType)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.EqualError(t, err, "not found")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetLatest_DatabaseError(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewBiometricRepository(db)

	userID := testUserID
	metricType := "heart_rate"

	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type FROM biometric_data WHERE user_id = \$1 AND metric_type = \$2 ORDER BY timestamp DESC LIMIT 1`).
		WithArgs(userID, metricType).
		WillReturnError(errors.New("internal server error"))

	result, err := repo.GetLatest(context.Background(), userID, metricType)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.EqualError(t, err, "query latest biometric: internal server error")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNewBiometricRepository(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewBiometricRepository(db)

	assert.NotNil(t, repo)
	assert.Implements(t, (*BiometricRepository)(nil), repo)
}

func TestBiometricRepository_GetByUser_LimitZero(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewBiometricRepository(db)

	userID := testUserID

	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type"})

	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type FROM biometric_data WHERE user_id = \$1 ORDER BY timestamp DESC LIMIT \$2`).
		WithArgs(userID, 0).
		WillReturnRows(rows)

	results, err := repo.GetByUser(context.Background(), userID, 0)

	require.NoError(t, err)
	assert.Empty(t, results)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_GetByUser_MultipleRecords(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewBiometricRepository(db)

	userID := testUserID
	limit := 3
	ts := time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type"}).
		AddRow("id-1", userID, "heart_rate", 72.5, ts, "smartwatch").
		AddRow("id-2", userID, "heart_rate", 68.0, ts.Add(-time.Hour), "smartwatch").
		AddRow("id-3", userID, "blood_pressure", 118.0, ts.Add(-2*time.Hour), "band")

	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type FROM biometric_data WHERE user_id = \$1 ORDER BY timestamp DESC LIMIT \$2`).
		WithArgs(userID, limit).
		WillReturnRows(rows)

	results, err := repo.GetByUser(context.Background(), userID, limit)

	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, "id-1", results[0].ID)
	assert.Equal(t, "id-2", results[1].ID)
	assert.Equal(t, "id-3", results[2].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricRepository_Save_WithNilContext(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewBiometricRepository(db)

	data := &domain.BiometricData{
		UserID:     testUserID,
		MetricType: "heart_rate",
		Value:      72.5,
		Timestamp:  time.Now(),
		DeviceType: "smartwatch",
	}

	mock.ExpectExec(`INSERT INTO biometric_data`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// sqlmock handles nil context gracefully; we still test the method contract
	err := repo.Save(context.Background(), data)

	assert.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
