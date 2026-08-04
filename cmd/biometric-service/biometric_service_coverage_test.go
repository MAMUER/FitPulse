package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/MAMUER/project/api/gen/biometric"
)

func TestBiometricServer_BatchAddRecords_Empty(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	log, _ := zap.NewDevelopment()
	s := newTestServer(db, log)

	_, err := s.BatchAddRecords(context.Background(), &pb.BatchAddRecordsRequest{
		UserId: "user-123",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBiometricServer_UpdateRecord_Validation(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		db, _, cleanup := setupTestDB(t)
		defer cleanup()

		log, _ := zap.NewDevelopment()
		s := newTestServer(db, log)

		_, err := s.UpdateRecord(context.Background(), &pb.UpdateRecordRequest{
			Id:    "",
			Value: 75.0,
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

		_, err := s.UpdateRecord(context.Background(), &pb.UpdateRecordRequest{
			Id:    "record-1",
			Value: -10.0,
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestBiometricServer_DeleteRecord_EmptyID(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	log, _ := zap.NewDevelopment()
	s := newTestServer(db, log)

	_, err := s.DeleteRecord(context.Background(), &pb.DeleteRecordRequest{
		Id: "",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBuildGetRecordsQuery(t *testing.T) {
	server := &biometricServer{}

	t.Run("no date range", func(t *testing.T) {
		req := &pb.GetRecordsRequest{
			UserId:     "user-123",
			MetricType: "heart_rate",
			Limit:      50,
			Offset:     0,
		}
		q := server.buildGetRecordsQuery(req)
		assert.Contains(t, q.query, "WHERE user_id = $1 AND metric_type = $2")
		assert.Contains(t, q.query, "ORDER BY timestamp DESC")
	})

	t.Run("both dates", func(t *testing.T) {
		req := &pb.GetRecordsRequest{
			UserId:     "user-123",
			MetricType: "heart_rate",
			From:       &timestamppb.Timestamp{Seconds: 1700000000},
			To:         &timestamppb.Timestamp{Seconds: 1700003600},
			Limit:      100,
			Offset:     0,
		}
		q := server.buildGetRecordsQuery(req)
		assert.Contains(t, q.query, "timestamp >= $3 AND timestamp <= $4")
	})
}

func TestSafeIntToInt32_AllCases(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int32
	}{
		{"positive value", 100, 100},
		{"zero", 0, 0},
		{"negative value", -100, -100},
		{"max int32", 2147483647, 2147483647},
		{"min int32", -2147483648, -2147483648},
		{"overflow positive", 2147483648, 2147483647},
		{"overflow negative", -2147483649, -2147483648},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeIntToInt32(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

