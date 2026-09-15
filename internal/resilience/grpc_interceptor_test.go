package resilience

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type fakeLogger struct {
	lastWarn string
	lastInfo string
}

func (f *fakeLogger) Warn(msg string, fields ...zap.Field) {
	f.lastWarn = msg
}

func (f *fakeLogger) Info(msg string, fields ...zap.Field) {
	f.lastInfo = msg
}

func TestExtractService(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected string
	}{
		{
			name:     "extracts service from full method",
			method:   "/user.UserService/GetUser",
			expected: "user.UserService",
		},
		{
			name:     "returns unknown for empty method",
			method:   "",
			expected: "unknown",
		},
		{
			name:     "returns trimmed method for single part",
			method:   "CustomMethod",
			expected: "CustomMethod",
		},
		{
			name:     "returns second part for two part method",
			method:   "/service-name/method",
			expected: "service-name",
		},
		{
			name:     "trims whitespace from service name",
			method:   "/  service.with.spaces  /method",
			expected: "service.with.spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractService(tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadCircuitBreaker(t *testing.T) {
	t.Run("creates new circuit breaker when not exists", func(t *testing.T) {
		breakers := &sync.Map{}
		cb := loadCircuitBreaker(breakers, "test-service")
		assert.NotNil(t, cb)
		assert.Equal(t, "test-service", cb.name)
	})

	t.Run("returns existing circuit breaker", func(t *testing.T) {
		breakers := &sync.Map{}
		cb1 := loadCircuitBreaker(breakers, "test-service")
		cb2 := loadCircuitBreaker(breakers, "test-service")
		assert.Equal(t, cb1, cb2)
		assert.Equal(t, "test-service", cb2.name)
	})

	t.Run("creates separate breakers for different services", func(t *testing.T) {
		breakers := &sync.Map{}
		cb1 := loadCircuitBreaker(breakers, "service-a")
		cb2 := loadCircuitBreaker(breakers, "service-b")
		assert.NotEqual(t, cb1, cb2)
		assert.Equal(t, "service-a", cb1.name)
		assert.Equal(t, "service-b", cb2.name)
	})
}

func TestUnaryClientInterceptor(t *testing.T) {
	t.Run("successful call passes through", func(t *testing.T) {
		logger := &fakeLogger{}
		interceptor := UnaryClientInterceptor(logger)

		err := interceptor(context.Background(), "/test.Service/Method", nil, nil, nil, func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return nil
		})
		assert.NoError(t, err)
		assert.Empty(t, logger.lastWarn)
	})

	t.Run("failed call returns error without logging", func(t *testing.T) {
		logger := &fakeLogger{}
		interceptor := UnaryClientInterceptor(logger)
		expectedErr := errors.New("call failed")

		err := interceptor(context.Background(), "/test.Service/Method", nil, nil, nil, func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return expectedErr
		})
		assert.Equal(t, expectedErr, err)
		assert.Empty(t, logger.lastWarn)
	})

	t.Run("circuit open error is logged", func(t *testing.T) {
		logger := &fakeLogger{}
		interceptor := UnaryClientInterceptor(logger)

		failingInvoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return errors.New("service error")
		}

		for i := 0; i < 5; i++ {
			_ = interceptor(context.Background(), "/test.Service/Method", nil, nil, nil, failingInvoker)
		}

		err := interceptor(context.Background(), "/test.Service/Method", nil, nil, nil, func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return nil
		})
		assert.Error(t, err)
		assert.Contains(t, logger.lastWarn, "gRPC circuit breaker open")
	})
}
