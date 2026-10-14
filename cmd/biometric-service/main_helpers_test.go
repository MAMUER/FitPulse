package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/MAMUER/project/api/gen/biometric"
)

func TestValidateBatchRequest(t *testing.T) {
	records := []*pb.AddRecordRequest{{UserId: "u1"}}

	require.NoError(t, validateBatchRequest(&pb.BatchAddRecordsRequest{UserId: "u1", Records: records}))
	require.NoError(t, validateBatchRequest(&pb.BatchAddRecordsRequest{UserId: "u1", Records: records}))

	err := validateBatchRequest(&pb.BatchAddRecordsRequest{UserId: "", Records: records})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "user_id is required")

	err = validateBatchRequest(&pb.BatchAddRecordsRequest{UserId: "u1"})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "records cannot be empty")
}

func TestValidateRecords(t *testing.T) {
	valid := []*pb.AddRecordRequest{
		{UserId: "u1", MetricType: "heart_rate", Value: 72},
	}
	require.NoError(t, validateRecords(context.Background(), valid))

	err := validateRecords(context.Background(), []*pb.AddRecordRequest{
		{UserId: "u1", MetricType: "heart_rate", Value: 999},
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "record[0]")

	// Отменённый контекст прерывает валидацию.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = validateRecords(ctx, valid)
	assert.Equal(t, codes.Canceled, status.Code(err))
}
