package main

import (
	"context"
	"os/exec"
	"strconv"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/MAMUER/project/api/gen/biometric"
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/queue"
	"github.com/MAMUER/project/internal/testcontainers"
)

func TestBiometricServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping integration tests")
	}

	ctx := context.Background()
	infra := testcontainers.StartInfrastructure(t)
	require.NotNil(t, infra)

	cfg := db.Config{
		Host:     infra.PostgresHost,
		Port:     strconv.Itoa(infra.PostgresPort),
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	database, err := db.NewConnection(cfg)
	require.NoError(t, err)
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			t.Logf("Failed to close database: %v", closeErr)
		}
	}()

	_, err = database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			role VARCHAR(50) NOT NULL DEFAULT 'client',
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS biometric_data (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			metric_type VARCHAR(50) NOT NULL,
			value DOUBLE PRECISION NOT NULL CHECK (value >= 0),
			timestamp TIMESTAMPTZ NOT NULL,
			device_type VARCHAR(50),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE (user_id, metric_type, timestamp, device_type)
		);
		CREATE INDEX IF NOT EXISTS idx_biometric_user_metric_time ON biometric_data(user_id, metric_type, timestamp);
	`)
	require.NoError(t, err)

	var userID string
	err = database.QueryRowContext(ctx, `INSERT INTO users (email, role) VALUES ($1, $2) RETURNING id`, "test@example.com", "client").Scan(&userID)
	require.NoError(t, err)

	zapLog, _ := zap.NewDevelopment()
	log := &logger.Logger{Logger: zapLog}

	rabbitURL := "amqp://testuser:testpass@" + infra.RabbitMQHost + ":" + strconv.Itoa(infra.RabbitMQPort) + "/"
	rabbitQueue, err := queue.NewPublisher(rabbitURL, "biometric_events", zapLog)
	if err != nil {
		t.Logf("RabbitMQ not available, tests will skip queue checks: %v", err)
	}
	var cleanUpQueue func()
	if rabbitQueue != nil {
		cleanUpQueue = func() { _ = rabbitQueue.Close() }
		defer cleanUpQueue()
	}

	server := &biometricServer{
		db:          database,
		log:         log,
		rabbitQueue: rabbitQueue,
	}

	t.Run("AddRecord_Success", func(t *testing.T) {
		resp, err := server.AddRecord(ctx, &pb.AddRecordRequest{
			UserId:     userID,
			MetricType: "heart_rate",
			Value:      75.0,
			Timestamp:  timestamppb.New(time.Now()),
			DeviceType: "test-device",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Id)
	})

	t.Run("AddRecord_InvalidMetric", func(t *testing.T) {
		_, err := server.AddRecord(ctx, &pb.AddRecordRequest{
			UserId:     userID,
			MetricType: "heart_rate",
			Value:      300.0,
			Timestamp:  timestamppb.New(time.Now()),
			DeviceType: "test-device",
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("AddRecord_UserNotFound", func(t *testing.T) {
		_, err := server.AddRecord(ctx, &pb.AddRecordRequest{
			UserId:     "00000000-0000-0000-0000-000000000000",
			MetricType: "heart_rate",
			Value:      75.0,
			Timestamp:  timestamppb.New(time.Now()),
			DeviceType: "test-device",
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	t.Run("AddRecord_Duplicate", func(t *testing.T) {
		ts := time.Now().Truncate(time.Second)
		_, err := server.AddRecord(ctx, &pb.AddRecordRequest{
			UserId:     userID,
			MetricType: "heart_rate",
			Value:      75.0,
			Timestamp:  timestamppb.New(ts),
			DeviceType: "test-device",
		})
		require.NoError(t, err)

		resp, err := server.AddRecord(ctx, &pb.AddRecordRequest{
			UserId:     userID,
			MetricType: "heart_rate",
			Value:      80.0,
			Timestamp:  timestamppb.New(ts),
			DeviceType: "test-device",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Id)
	})

	t.Run("BatchAddRecords_Success", func(t *testing.T) {
		resp, err := server.BatchAddRecords(ctx, &pb.BatchAddRecordsRequest{
			UserId: userID,
			Records: []*pb.AddRecordRequest{
				{MetricType: "spo2", Value: 98.0, Timestamp: timestamppb.New(time.Now()), DeviceType: "test-device"},
				{MetricType: "spo2", Value: 97.5, Timestamp: timestamppb.New(time.Now()), DeviceType: "test-device"},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, int32(2), resp.Count)
	})

	t.Run("GetRecords_Success", func(t *testing.T) {
		resp, err := server.GetRecords(ctx, &pb.GetRecordsRequest{
			UserId:     userID,
			MetricType: "heart_rate",
			Limit:      100,
			Offset:     0,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Records)
	})

	t.Run("GetLatest_Success", func(t *testing.T) {
		record, err := server.GetLatest(ctx, &pb.GetLatestRequest{
			UserId:     userID,
			MetricType: "heart_rate",
		})
		require.NoError(t, err)
		assert.Equal(t, userID, record.UserId)
		assert.Equal(t, "heart_rate", record.MetricType)
	})

	t.Run("UpdateRecord_Success", func(t *testing.T) {
		record, err := server.GetLatest(ctx, &pb.GetLatestRequest{
			UserId:     userID,
			MetricType: "heart_rate",
		})
		require.NoError(t, err)

		updated, err := server.UpdateRecord(ctx, &pb.UpdateRecordRequest{
			Id:        record.Id,
			Value:     80.0,
			Timestamp: timestamppb.New(time.Now()),
		})
		require.NoError(t, err)
		assert.Equal(t, 80.0, updated.Value)
	})

	t.Run("DeleteRecord_Success", func(t *testing.T) {
		record, err := server.GetLatest(ctx, &pb.GetLatestRequest{
			UserId:     userID,
			MetricType: "heart_rate",
		})
		require.NoError(t, err)

		delResp, err := server.DeleteRecord(ctx, &pb.DeleteRecordRequest{
			Id: record.Id,
		})
		require.NoError(t, err)
		assert.True(t, delResp.Deleted)
	})

	t.Run("DeleteRecord_NotFound", func(t *testing.T) {
		_, err := server.DeleteRecord(ctx, &pb.DeleteRecordRequest{
			Id: "00000000-0000-0000-0000-000000000000",
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	t.Run("AddRecord_NilRequest", func(t *testing.T) {
		_, err := server.AddRecord(ctx, nil)
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("BatchAddRecords_EmptyRecords", func(t *testing.T) {
		_, err := server.BatchAddRecords(ctx, &pb.BatchAddRecordsRequest{
			UserId:  userID,
			Records: []*pb.AddRecordRequest{},
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func isDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	return err == nil
}
