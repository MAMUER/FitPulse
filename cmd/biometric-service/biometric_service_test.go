package main

import (
	"context"
	"database/sql"
	"testing"

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

func setupTestDB(t *testing.T, userExists bool, insertOK bool) (*sql.DB, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	if userExists {
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1\)`).WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		if insertOK {
			mock.ExpectExec(`INSERT INTO biometric_data`).WillReturnResult(sqlmock.NewResult(1, 1))
		}
	} else {
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1\)`).WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	}

	cleanup := func() {
		_ = db.Close()
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Logf("Mock expectations not met: %v", err)
		}
	}

	return db, cleanup
}

func newTestServer(db *sql.DB, log *zap.Logger) *biometricServer {
	return &biometricServer{db: db, log: &logger.Logger{Logger: log}, rabbitQueue: nil}
}

func TestBiometricServer_AddRecord_InvalidRequest(t *testing.T) {
	t.Run("empty user_id", func(t *testing.T) {
		db, cleanup := setupTestDB(t, false, false)
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
		db, cleanup := setupTestDB(t, false, false)
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
		db, cleanup := setupTestDB(t, false, false)
		defer cleanup()

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
		db, cleanup := setupTestDB(t, false, false)
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
		db, cleanup := setupTestDB(t, false, false)
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
	db, cleanup := setupTestDB(t, true, true)
	defer cleanup()

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
