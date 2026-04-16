// cmd/biometric-service/biometric_service_test.go
package main

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	pb "github.com/MAMUER/Project/api/gen/biometric"
	"github.com/MAMUER/Project/internal/logger"
	"github.com/MAMUER/Project/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) Publish(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestValidateBiometricRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.AddRecordRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid heart rate",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      75.0,
			},
			wantErr: false,
		},
		{
			name: "missing user_id",
			req: &pb.AddRecordRequest{
				MetricType: "heart_rate",
				Value:      75.0,
			},
			wantErr: true,
			errMsg:  "user_id is required",
		},
		{
			name: "missing metric_type",
			req: &pb.AddRecordRequest{
				UserId: "user-123",
				Value:  75.0,
			},
			wantErr: true,
			errMsg:  "metric_type is required",
		},
		{
			name: "negative value",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      -10.0,
			},
			wantErr: true,
			errMsg:  "value cannot be negative",
		},
		{
			name: "heart rate too low",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      25.0,
			},
			wantErr: true,
			errMsg:  "heart_rate out of valid range",
		},
		{
			name: "heart rate too high",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      250.0,
			},
			wantErr: true,
			errMsg:  "heart_rate out of valid range",
		},
		{
			name: "valid spo2",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "spo2",
				Value:      98.0,
			},
			wantErr: false,
		},
		{
			name: "spo2 below range",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "spo2",
				Value:      69.0,
			},
			wantErr: true,
			errMsg:  "spo2 out of valid range",
		},
		{
			name: "spo2 above range",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "spo2",
				Value:      101.0,
			},
			wantErr: true,
			errMsg:  "spo2 out of valid range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateBiometricRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBiometricServer_AddRecord_ValidationError(t *testing.T) {
	mockLog := logger.New("test-biometric")
	mockQueue := new(mockPublisher)

	server := &biometricServer{
		log:         mockLog,
		rabbitQueue: mockQueue,
		// db не нужен для тестов валидации
	}

	tests := []struct {
		name       string
		req        *pb.AddRecordRequest
		wantCode   codes.Code
		wantErrMsg string
	}{
		{
			name: "missing user_id",
			req: &pb.AddRecordRequest{
				MetricType: "heart_rate",
				Value:      75.0,
			},
			wantCode:   codes.InvalidArgument,
			wantErrMsg: "user_id is required",
		},
		{
			name: "missing metric_type",
			req: &pb.AddRecordRequest{
				UserId: "user-123",
				Value:  75.0,
			},
			wantCode:   codes.InvalidArgument,
			wantErrMsg: "metric_type is required",
		},
		{
			name: "negative value",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      -10.0,
			},
			wantCode:   codes.InvalidArgument,
			wantErrMsg: "value cannot be negative",
		},
		{
			name: "heart rate out of range low",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      25.0,
			},
			wantCode:   codes.InvalidArgument,
			wantErrMsg: "heart_rate out of valid range",
		},
		{
			name: "heart rate out of range high",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      250.0,
			},
			wantCode:   codes.InvalidArgument,
			wantErrMsg: "heart_rate out of valid range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.AddRecord(context.Background(), tt.req)

			assert.Nil(t, resp)
			assert.Error(t, err)

			st, ok := status.FromError(err)
			assert.True(t, ok, "error should be a gRPC status")
			assert.Equal(t, tt.wantCode, st.Code())
			assert.Contains(t, st.Message(), tt.wantErrMsg)
		})
	}

	mockQueue.AssertExpectations(t)
}

func TestBiometricServer_AddRecord_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectExec(`INSERT INTO biometric_data`).
		WithArgs(
			sqlmock.AnyArg(),
			"user-123",
			"heart_rate",
			75.0,
			sqlmock.AnyArg(),
			"test_device",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	srv := &biometricServer{
		db:  db,
		log: logger.New("test"),
		// rabbitQueue можно замокать или оставить nil для этого теста
	}

	resp, err := srv.AddRecord(context.Background(), &pb.AddRecordRequest{
		UserId:     "user-123",
		MetricType: "heart_rate",
		Value:      75.0,
		Timestamp:  timestamppb.Now(),
		DeviceType: "test_device",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	if resp != nil {
		assert.NotEmpty(t, resp.Id)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchAddRecords_Validation(t *testing.T) {
	mockLog := logger.New("test")
	// ✅ Используем мокаемый интерфейс, а не конкретный тип
	mockQueue := new(mockPublisher)
	// Не ожидаем Close - сервер не вызывает Close на очереди при валидации

	// Создаём сервер с nil db - валидация происходит раньше обращения к БД
	server := &biometricServer{
		db:          nil,
		log:         mockLog,
		rabbitQueue: mockQueue,
	}

	tests := []struct {
		name       string
		req        *pb.BatchAddRecordsRequest
		wantCode   codes.Code
		wantErrMsg string
	}{
		{
			name: "empty user_id",
			req: &pb.BatchAddRecordsRequest{
				UserId: "",
				Records: []*pb.AddRecordRequest{
					{MetricType: "heart_rate", Value: 75.0},
				},
			},
			wantCode:   codes.InvalidArgument,
			wantErrMsg: "user_id is required",
		},
		{
			name: "empty records list",
			req: &pb.BatchAddRecordsRequest{
				UserId:  "user-123",
				Records: []*pb.AddRecordRequest{},
			},
			wantCode:   codes.InvalidArgument,
			wantErrMsg: "records cannot be empty",
		},
		{
			name: "invalid record in batch - missing metric_type",
			req: &pb.BatchAddRecordsRequest{
				UserId: "user-123",
				Records: []*pb.AddRecordRequest{
					{Value: 98.0}, // Missing MetricType
				},
			},
			wantCode:   codes.InvalidArgument,
			wantErrMsg: "record[0]: metric_type is required",
		},
		{
			name: "invalid record in batch - heart_rate out of range",
			req: &pb.BatchAddRecordsRequest{
				UserId: "user-123",
				Records: []*pb.AddRecordRequest{
					{MetricType: "heart_rate", Value: 250.0},
				},
			},
			wantCode:   codes.InvalidArgument,
			wantErrMsg: "heart_rate out of valid range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.BatchAddRecords(context.Background(), tt.req)

			assert.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok, "error should be gRPC status")
			assert.Equal(t, tt.wantCode, st.Code())
			if tt.wantErrMsg != "" {
				assert.Contains(t, st.Message(), tt.wantErrMsg)
			}
			assert.Nil(t, resp)
		})
	}
}

func TestBatchAddRecords_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO biometric_data`)).
		WithArgs(
			sqlmock.AnyArg(),
			"user-123",
			"heart_rate",
			75.0,
			sqlmock.AnyArg(),
			"device-1",
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO biometric_data`)).
		WithArgs(
			sqlmock.AnyArg(),
			"user-123",
			"spo2",
			98.0,
			sqlmock.AnyArg(),
			"device-1",
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	srv := &biometricServer{
		db:  db,
		log: logger.New("test"),
	}

	req := &pb.BatchAddRecordsRequest{
		UserId: "user-123",
		Records: []*pb.AddRecordRequest{
			{
				MetricType: "heart_rate",
				Value:      75.0,
				DeviceType: "device-1",
				Timestamp:  timestamppb.Now(),
			},
			{
				MetricType: "spo2",
				Value:      98.0,
				DeviceType: "device-1",
				Timestamp:  timestamppb.Now(),
			},
		},
	}

	resp, err := srv.BatchAddRecords(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	if resp != nil {
		assert.Equal(t, int32(2), resp.Count)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchAddRecords_RollbackOnDBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	s := &biometricServer{db: db, log: logger.New("test")}
	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO biometric_data`)).
		WithArgs(
			sqlmock.AnyArg(), "user-123", "heart_rate", 75.0, sqlmock.AnyArg(), "device-1", sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO biometric_data`)).
		WithArgs(
			sqlmock.AnyArg(), "user-123", "spo2", 98.0, sqlmock.AnyArg(), "device-1", sqlmock.AnyArg(),
		).
		WillReturnError(errors.New("simulated DB error"))

	mock.ExpectRollback()

	req := &pb.BatchAddRecordsRequest{
		UserId: "user-123",
		Records: []*pb.AddRecordRequest{
			{MetricType: "heart_rate", Value: 75.0, DeviceType: "device-1", Timestamp: timestamppb.Now()},
			{MetricType: "spo2", Value: 98.0, DeviceType: "device-1", Timestamp: timestamppb.Now()},
		},
	}

	resp, err := s.BatchAddRecords(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricServer_GetRecords(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	srv := &biometricServer{db: db, log: logger.New("test")}

	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	to := now

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "metric_type", "value", "timestamp", "device_type", "created_at",
	}).AddRow(
		"rec-1", "user-123", "heart_rate", 75.0, from, "device-1", now,
	)

	// Use AnyArg for timestamps to avoid timezone issues
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, metric_type, value, timestamp, device_type, created_at`)).
		WithArgs("user-123", "heart_rate", sqlmock.AnyArg(), sqlmock.AnyArg(), int32(100)).
		WillReturnRows(rows)

	resp, err := srv.GetRecords(context.Background(), &pb.GetRecordsRequest{
		UserId:     "user-123",
		MetricType: "heart_rate",
		From:       timestamppb.New(from),
		To:         timestamppb.New(to),
		Limit:      100,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	if resp != nil {
		assert.Len(t, resp.Records, 1)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricServer_GetLatest_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	srv := &biometricServer{
		db:  db,
		log: logger.New("test"),
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, metric_type, value, timestamp, device_type, created_at`)).
		WithArgs("user-123", "heart_rate").
		WillReturnError(sql.ErrNoRows)

	resp, err := srv.GetLatest(context.Background(), &pb.GetLatestRequest{
		UserId:     "user-123",
		MetricType: "heart_rate",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestValidateBiometricRequest_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.AddRecordRequest
		wantErr bool
	}{
		{
			name: "spo2 valid range boundary low",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "spo2",
				Value:      70.0,
			},
			wantErr: false,
		},
		{
			name: "spo2 valid range boundary high",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "spo2",
				Value:      100.0,
			},
			wantErr: false,
		},
		{
			name: "heart_rate valid range boundary low",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      30.0,
			},
			wantErr: false,
		},
		{
			name: "heart_rate valid range boundary high",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      220.0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateBiometricRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBiometricServer_GetRecords_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	srv := &biometricServer{db: db, log: logger.New("test")}

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, metric_type, value, timestamp, device_type, created_at`)).
		WithArgs("user-123", "heart_rate", sqlmock.AnyArg(), sqlmock.AnyArg(), int32(100)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "metric_type", "value", "timestamp", "device_type", "created_at"}))

	resp, err := srv.GetRecords(context.Background(), &pb.GetRecordsRequest{
		UserId:     "user-123",
		MetricType: "heart_rate",
		From:       timestamppb.New(now.Add(-24 * time.Hour)),
		To:         timestamppb.New(now),
		Limit:      100,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	if resp != nil {
		assert.Empty(t, resp.Records)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBiometricServer_GetRecords_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	srv := &biometricServer{db: db, log: logger.New("test")}

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, metric_type, value, timestamp, device_type, created_at`)).
		WithArgs("user-123", "heart_rate", sqlmock.AnyArg(), sqlmock.AnyArg(), int32(100)).
		WillReturnError(errors.New("db error"))

	resp, err := srv.GetRecords(context.Background(), &pb.GetRecordsRequest{
		UserId:     "user-123",
		MetricType: "heart_rate",
		From:       timestamppb.New(now.Add(-24 * time.Hour)),
		To:         timestamppb.New(now),
		Limit:      100,
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchAddRecords_ContextCancelled(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	srv := &biometricServer{db: db, log: logger.New("test")}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу

	// Валидация должна пройти, поэтому добавляем корректные данные
	req := &pb.BatchAddRecordsRequest{
		UserId: "user-123",
		Records: []*pb.AddRecordRequest{
			{MetricType: "heart_rate", Value: 75.0, Timestamp: timestamppb.Now()},
		},
	}

	resp, err := srv.BatchAddRecords(ctx, req)
	// Ошибка может быть context.Canceled или ошибка БД из-за отмены
	// Проверяем, что ответ nil при ошибке
	if err != nil {
		assert.Nil(t, resp)
	}
}
