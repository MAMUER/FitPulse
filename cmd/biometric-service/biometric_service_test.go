package main

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	pb "github.com/MAMUER/project/api/gen/biometric"
	"github.com/MAMUER/project/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestBiometricServer_AddRecord_InvalidRequest(t *testing.T) {
	db, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	zapLog, _ := zap.NewDevelopment()
	log := &logger.Logger{Logger: zapLog}
	s := &biometricServer{
		db:          db,
		log:         log,
		rabbitQueue: nil,
	}

	tests := []struct {
		name     string
		req      *pb.AddRecordRequest
		wantCode codes.Code
	}{
		{
			name: "empty user_id",
			req: &pb.AddRecordRequest{
				UserId:     "",
				MetricType: "heart_rate",
				Value:      75.0,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "negative value",
			req: &pb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      -10.0,
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.AddRecord(context.Background(), tt.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, tt.wantCode, st.Code())
		})
	}
}
