package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/MAMUER/project/api/gen/biometric"
	"github.com/MAMUER/project/internal/logger"
)

func setupTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	cleanup := func() {
		_ = db.Close()
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Logf("Mock expectations not met: %v", err)
		}
	}

	return db, mock, cleanup
}

func newTestServer(db *sql.DB, log *zap.Logger) *biometricServer {
	return &biometricServer{db: db, log: &logger.Logger{Logger: log}, rabbitQueue: nil}
}

func TestBiometricServer_AddRecord_InvalidRequest(t *testing.T) {
	t.Run("empty user_id", func(t *testing.T) {
		db, _, cleanup := setupTestDB(t)
		defer cleanup()

		log, _ := zap.NewDevelopment()
		s := newTestServer(db, log)

		_, err := s.AddRecord(context.Background(), &pb.AddRecordRequest{
			UserId:     "",
			MetricType: "heart_rate",
			Value:      75.0,
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("negative value", func(t *testing.T) {
		db, _, cleanup := setupTestDB(t)
		defer cleanup()

		log, _ := zap.NewDevelopment()
		s := newTestServer(db, log)

		_, err := s.AddRecord(context.Background(), &pb.AddRecordRequest{
			UserId:     "user-123",
			MetricType: "heart_rate",
			Value:      -10.0,
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("user not found", func(t *testing.T) {
		db, mock, cleanup := setupTestDB(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1\)`).WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		log, _ := zap.NewDevelopment()
		s := newTestServer(db, log)

		_, err := s.AddRecord(context.Background(), &pb.AddRecordRequest{
			UserId:     "00000000-0000-0000-0000-000000000000",
			MetricType: "heart_rate",
			Value:      75.0,
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	t.Run("heart_rate out of range", func(t *testing.T) {
		db, _, cleanup := setupTestDB(t)
		defer cleanup()

		log, _ := zap.NewDevelopment()
		s := newTestServer(db, log)

		_, err := s.AddRecord(context.Background(), &pb.AddRecordRequest{
			UserId:     "user-123",
			MetricType: "heart_rate",
			Value:      300.0,
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("spo2 out of range", func(t *testing.T) {
		db, _, cleanup := setupTestDB(t)
		defer cleanup()

		log, _ := zap.NewDevelopment()
		s := newTestServer(db, log)

		_, err := s.AddRecord(context.Background(), &pb.AddRecordRequest{
			UserId:     "user-123",
			MetricType: "spo2",
			Value:      60.0,
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestBiometricServer_AddRecord_Success(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1\)`).WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`INSERT INTO biometric_data`).WithArgs(
		sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
		sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("generated-record-id"))

	log, _ := zap.NewDevelopment()
	s := newTestServer(db, log)

	resp, err := s.AddRecord(context.Background(), &pb.AddRecordRequest{
		UserId:     "user-123",
		MetricType: "heart_rate",
		Value:      75.0,
		Timestamp:  &timestamppb.Timestamp{Seconds: 1700000000},
		DeviceType: "test",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Id)
}

func TestBiometricServer_GetRecords_Success(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type, created_at FROM biometric_data WHERE user_id = \$1 AND metric_type = \$2`).
		WithArgs("user-123", "heart_rate").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type", "created_at"}).
			AddRow("record-1", "user-123", "heart_rate", 75.0, time.Now(), "test", time.Now()))

	log, _ := zap.NewDevelopment()
	s := newTestServer(db, log)

	resp, err := s.GetRecords(context.Background(), &pb.GetRecordsRequest{
		UserId:     "user-123",
		MetricType: "heart_rate",
		Limit:      50,
		Offset:     0,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Records, 1)
	assert.Equal(t, "record-1", resp.Records[0].Id)
}

func TestBiometricServer_GetRecords_QueryError(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type, created_at FROM biometric_data WHERE user_id = \$1 AND metric_type = \$2`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(fmt.Errorf("query failed"))

	log, _ := zap.NewDevelopment()
	s := newTestServer(db, log)

	_, err := s.GetRecords(context.Background(), &pb.GetRecordsRequest{
		UserId:     "user-123",
		MetricType: "heart_rate",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestBiometricServer_GetLatest_Success(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type, created_at FROM biometric_data WHERE user_id = \$1 AND metric_type = \$2 ORDER BY timestamp DESC LIMIT 1`).
		WithArgs("user-123", "heart_rate").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type", "created_at"}).
			AddRow("record-1", "user-123", "heart_rate", 75.0, now, "test", now))

	log, _ := zap.NewDevelopment()
	s := newTestServer(db, log)

	resp, err := s.GetLatest(context.Background(), &pb.GetLatestRequest{
		UserId:     "user-123",
		MetricType: "heart_rate",
	})
	require.NoError(t, err)
	assert.Equal(t, "record-1", resp.Id)
	assert.Equal(t, "heart_rate", resp.MetricType)
}

func TestBiometricServer_GetLatest_NotFound(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, user_id, metric_type, value, timestamp, device_type, created_at FROM biometric_data WHERE user_id = \$1 AND metric_type = \$2 ORDER BY timestamp DESC LIMIT 1`).
		WithArgs("user-123", "heart_rate").
		WillReturnError(sql.ErrNoRows)

	log, _ := zap.NewDevelopment()
	s := newTestServer(db, log)

	_, err := s.GetLatest(context.Background(), &pb.GetLatestRequest{
		UserId:     "user-123",
		MetricType: "heart_rate",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestBiometricServer_BatchAddRecords_Success(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1\)`).WithArgs("user-123").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	for i := 0; i < 2; i++ {
		mock.ExpectExec(`INSERT INTO biometric_data`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(int64(i+1), 1))
	}
	mock.ExpectCommit()

	log, _ := zap.NewDevelopment()
	s := newTestServer(db, log)

	resp, err := s.BatchAddRecords(context.Background(), &pb.BatchAddRecordsRequest{
		UserId: "user-123",
		Records: []*pb.AddRecordRequest{
			{MetricType: "heart_rate", Value: 75.0, Timestamp: timestamppb.New(time.Now()), DeviceType: "test"},
			{MetricType: "spo2", Value: 98.0, Timestamp: timestamppb.New(time.Now()), DeviceType: "test"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, int32(2), resp.Count)
}

func TestBiometricServer_UpdateRecord_Success(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now()
	ts := timestamppb.New(now)
	mock.ExpectQuery(`UPDATE biometric_data SET value = \$1, timestamp = \$2, device_type = \$3 WHERE id = \$4 RETURNING id, user_id, metric_type, value, timestamp, device_type, created_at`).
		WithArgs(80.0, ts.AsTime(), "new-device", "record-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type", "created_at"}).
			AddRow("record-1", "user-123", "heart_rate", 80.0, ts.AsTime(), "new-device", ts.AsTime()))

	log, _ := zap.NewDevelopment()
	s := newTestServer(db, log)

	resp, err := s.UpdateRecord(context.Background(), &pb.UpdateRecordRequest{
		Id:         "record-1",
		Value:      80.0,
		DeviceType: "new-device",
		Timestamp:  ts,
	})
	require.NoError(t, err)
	assert.Equal(t, "record-1", resp.Id)
	assert.Equal(t, 80.0, resp.Value)
}

func TestBiometricServer_UpdateRecord_NotFound(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	mock.ExpectQuery(`UPDATE biometric_data SET value = \$1, timestamp = \$2, device_type = \$3 WHERE id = \$4 RETURNING id, user_id, metric_type, value, timestamp, device_type, created_at`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	log, _ := zap.NewDevelopment()
	s := newTestServer(db, log)

	_, err := s.UpdateRecord(context.Background(), &pb.UpdateRecordRequest{
		Id:    "missing",
		Value: 80.0,
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestBiometricServer_DeleteRecord_Success(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM biometric_data WHERE id = \$1`).
		WithArgs("record-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	log, _ := zap.NewDevelopment()
	s := newTestServer(db, log)

	_, err := s.DeleteRecord(context.Background(), &pb.DeleteRecordRequest{
		Id: "record-1",
	})
	require.NoError(t, err)
}

func TestBiometricServer_DeleteRecord_NotFound(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	mock.ExpectExec(`DELETE FROM biometric_data WHERE id = \$1`).
		WithArgs("missing").
		WillReturnResult(sqlmock.NewResult(1, 0))

	log, _ := zap.NewDevelopment()
	s := newTestServer(db, log)

	_, err := s.DeleteRecord(context.Background(), &pb.DeleteRecordRequest{
		Id: "missing",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}
