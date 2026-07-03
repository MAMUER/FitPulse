package metrics

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	metricRequestsTotal   = "grpc_requests_total"
	metricErrorsTotal     = "grpc_errors_total"
	metricDurationSeconds = "grpc_request_duration_seconds"
	statusError           = "error"
)

// resetMetrics recreates the metric vectors with a new registry to isolate test cases.
func resetMetrics(t *testing.T) *prometheus.Registry {
	t.Helper()
	registry := prometheus.NewRegistry()

	rpcRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests by service, method and status",
		},
		[]string{"service", "method", "status"},
	)
	registry.MustRegister(rpcRequestsTotal)

	rpcDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "Histogram of gRPC request durations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method"},
	)
	registry.MustRegister(rpcDurationSeconds)

	rpcErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_errors_total",
			Help: "Total number of gRPC errors by service, method and error code",
		},
		[]string{"service", "method", "error_code"},
	)
	registry.MustRegister(rpcErrorsTotal)

	return registry
}

// assertMetricMatches finds the first metric in the provided families whose name
// matches expectedName and whose labels match expectedLabels, and fails the test otherwise.
func assertMetricMatches(t *testing.T, metrics []*dto.MetricFamily, expectedName string, expectedLabels map[string]string) *dto.Metric {
	t.Helper()
	for _, m := range metrics {
		if m.GetName() == expectedName {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				match := true
				for k, v := range expectedLabels {
					if labelMap[k] != v {
						match = false
						break
					}
				}
				if match {
					return metric
				}
			}
		}
	}
	require.Fail(t, "metric not found", "expected metric %q with labels %v", expectedName, expectedLabels)
	return nil
}

// ---------------------------------------------------------------------------
// UnaryServerInterceptor tests
// ---------------------------------------------------------------------------

func TestUnaryServerInterceptor_Success(t *testing.T) {
	registry := resetMetrics(t)

	const (
		serviceName = "test-service"
		fullMethod  = "/api.v1.UserService/GetUser"
	)

	interceptor := UnaryServerInterceptor(serviceName)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return map[string]string{"id": "1"}, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: fullMethod,
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"id": "1"}, resp)

	// Verify rpcRequestsTotal incremented with status "ok"
	count := testutil.CollectAndCount(registry, metricRequestsTotal)
	assert.Equal(t, 1, count, "expected exactly one grpc_requests_total series")

	requestsMetric, err := registry.Gather()
	require.NoError(t, err)

	metric := assertMetricMatches(t, requestsMetric, metricRequestsTotal, map[string]string{
		"service": serviceName,
		"method":  fullMethod,
		"status":  "ok",
	})
	assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)

	// Verify rpcDurationSeconds observed
	durationMetric := assertMetricMatches(t, requestsMetric, metricDurationSeconds, map[string]string{
		"service": serviceName,
		"method":  fullMethod,
	})
	assert.Greater(t, durationMetric.GetHistogram().GetSampleCount(), uint64(0))

	// Verify no errors recorded
	errCount := testutil.CollectAndCount(registry, metricErrorsTotal)
	assert.Equal(t, 0, errCount, "expected no grpc_errors_total on success")
}

func TestUnaryServerInterceptor_Error(t *testing.T) {
	registry := resetMetrics(t)

	const (
		serviceName = "test-service"
		fullMethod  = "/api.v1.UserService/GetUser"
	)

	interceptor := UnaryServerInterceptor(serviceName)
	expectedErr := status.Error(codes.NotFound, "user not found")

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, fmt.Errorf("test handler: %w", expectedErr)
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: fullMethod,
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	metrics, err := registry.Gather()
	require.NoError(t, err)

	requestMetric := assertMetricMatches(t, metrics, metricRequestsTotal, map[string]string{
		"service": serviceName,
		"method":  fullMethod,
		"status":  statusError,
	})
	assert.InDelta(t, float64(1), requestMetric.GetCounter().GetValue(), 0.0001)

	errorMetric := assertMetricMatches(t, metrics, metricErrorsTotal, map[string]string{
		"service":    serviceName,
		"method":     fullMethod,
		"error_code": codes.NotFound.String(),
	})
	assert.InDelta(t, float64(1), errorMetric.GetCounter().GetValue(), 0.0001)
}

func TestUnaryServerInterceptor_InternalError(t *testing.T) {
	registry := resetMetrics(t)

	const (
		serviceName = "test-service"
		fullMethod  = "/api.v1.PaymentService/ProcessPayment"
	)

	interceptor := UnaryServerInterceptor(serviceName)
	expectedErr := status.Error(codes.Internal, "database unavailable")

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, fmt.Errorf("test handler: %w", expectedErr)
	}

	info := &grpc.UnaryServerInfo{FullMethod: fullMethod}

	_, err := interceptor(context.Background(), nil, info, handler)

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))

	metrics, err := registry.Gather()
	require.NoError(t, err)

	errorMetric := assertMetricMatches(t, metrics, metricErrorsTotal, map[string]string{
		"service":    serviceName,
		"method":     fullMethod,
		"error_code": codes.Internal.String(),
	})
	assert.InDelta(t, float64(1), errorMetric.GetCounter().GetValue(), 0.0001)
}

func TestUnaryServerInterceptor_MultipleCalls(t *testing.T) {
	registry := resetMetrics(t)

	const (
		serviceName = "test-service"
		fullMethod  = "/api.v1.UserService/ListUsers"
	)

	interceptor := UnaryServerInterceptor(serviceName)
	info := &grpc.UnaryServerInfo{FullMethod: fullMethod}

	successHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return []string{"user1", "user2"}, nil
	}

	errorHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, status.Error(codes.DeadlineExceeded, "timeout")
	}

	// 3 successful calls
	for i := 0; i < 3; i++ {
		_, err := interceptor(context.Background(), nil, info, successHandler)
		require.NoError(t, err)
	}

	// 2 error calls
	for i := 0; i < 2; i++ {
		_, err := interceptor(context.Background(), nil, info, errorHandler)
		require.Error(t, err)
	}

	metrics, err := registry.Gather()
	require.NoError(t, err)

	okMetric := assertMetricMatches(t, metrics, metricRequestsTotal, map[string]string{
		"service": serviceName,
		"method":  fullMethod,
		"status":  "ok",
	})
	assert.InDelta(t, float64(3), okMetric.GetCounter().GetValue(), 0.0001)

	errMetric := assertMetricMatches(t, metrics, metricRequestsTotal, map[string]string{
		"service": serviceName,
		"method":  fullMethod,
		"status":  statusError,
	})
	assert.InDelta(t, float64(2), errMetric.GetCounter().GetValue(), 0.0001)

	dlMetric := assertMetricMatches(t, metrics, metricErrorsTotal, map[string]string{
		"service":    serviceName,
		"method":     fullMethod,
		"error_code": codes.DeadlineExceeded.String(),
	})
	assert.InDelta(t, float64(2), dlMetric.GetCounter().GetValue(), 0.0001)
}

func TestUnaryServerInterceptor_DurationIsRecorded(t *testing.T) {
	resetMetrics(t)

	interceptor := UnaryServerInterceptor("svc")

	slowHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		time.Sleep(10 * time.Millisecond)
		return "done", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/slow/Method"}

	_, err := interceptor(context.Background(), nil, info, slowHandler)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// UnaryClientInterceptor tests
// ---------------------------------------------------------------------------

func TestUnaryClientInterceptor_Success(t *testing.T) {
	registry := resetMetrics(t)

	const (
		serviceName = "client-service"
		method      = "/api.v1.OrderService/CreateOrder"
	)

	interceptor := UnaryClientInterceptor(serviceName)

	invoker := func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		return nil
	}

	err := interceptor(
		context.Background(),
		method,
		nil, nil,
		nil,
		invoker,
	)

	require.NoError(t, err)

	metrics, err := registry.Gather()
	require.NoError(t, err)

	metric := assertMetricMatches(t, metrics, metricRequestsTotal, map[string]string{
		"service": serviceName,
		"method":  method,
		"status":  "ok",
	})
	assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)

	errCount := testutil.CollectAndCount(registry, metricErrorsTotal)
	assert.Equal(t, 0, errCount, "expected no grpc_errors_total on client success")
}

func TestUnaryClientInterceptor_Error(t *testing.T) {
	registry := resetMetrics(t)

	const (
		serviceName = "client-service"
		method      = "/api.v1.OrderService/CreateOrder"
	)

	interceptor := UnaryClientInterceptor(serviceName)
	expectedErr := status.Error(codes.PermissionDenied, "access denied")

	invoker := func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		return expectedErr
	}

	err := interceptor(
		context.Background(),
		method,
		nil, nil,
		nil,
		invoker,
	)

	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	metrics, err := registry.Gather()
	require.NoError(t, err)

	requestMetric := assertMetricMatches(t, metrics, metricRequestsTotal, map[string]string{
		"service": serviceName,
		"method":  method,
		"status":  statusError,
	})
	assert.InDelta(t, float64(1), requestMetric.GetCounter().GetValue(), 0.0001)

	errorMetric := assertMetricMatches(t, metrics, metricErrorsTotal, map[string]string{
		"service":    serviceName,
		"method":     method,
		"error_code": codes.PermissionDenied.String(),
	})
	assert.InDelta(t, float64(1), errorMetric.GetCounter().GetValue(), 0.0001)
}

func TestUnaryClientInterceptor_NonStatusError(t *testing.T) {
	registry := resetMetrics(t)

	const (
		serviceName = "client-service"
		method      = "/api.v1.FallbackService/Ping"
	)

	interceptor := UnaryClientInterceptor(serviceName)
	expectedErr := errors.New("network unreachable")

	invoker := func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		return expectedErr
	}

	err := interceptor(
		context.Background(),
		method,
		nil, nil,
		nil,
		invoker,
	)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)

	metrics, err := registry.Gather()
	require.NoError(t, err)

	// Should still record request as error, but error_code will be "OK" (unknown error)
	metric := assertMetricMatches(t, metrics, metricRequestsTotal, map[string]string{
		"service": serviceName,
		"method":  method,
		"status":  statusError,
	})
	assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
}

func TestUnaryClientInterceptor_MultipleCalls(t *testing.T) {
	registry := resetMetrics(t)

	const (
		serviceName = "client-service"
		method      = "/api.v1.UserService/GetProfile"
	)

	interceptor := UnaryClientInterceptor(serviceName)

	successInvoker := func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		return nil
	}

	errorInvoker := func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		return status.Error(codes.Unavailable, "service unavailable")
	}

	// 5 successful calls
	for i := 0; i < 5; i++ {
		err := interceptor(context.Background(), method, nil, nil, nil, successInvoker)
		require.NoError(t, err)
	}

	// 3 error calls
	for i := 0; i < 3; i++ {
		err := interceptor(context.Background(), method, nil, nil, nil, errorInvoker)
		require.Error(t, err)
	}

	metrics, err := registry.Gather()
	require.NoError(t, err)

	okMetric := assertMetricMatches(t, metrics, metricRequestsTotal, map[string]string{
		"service": serviceName,
		"method":  method,
		"status":  "ok",
	})
	assert.InDelta(t, float64(5), okMetric.GetCounter().GetValue(), 0.0001)

	errMetric := assertMetricMatches(t, metrics, metricRequestsTotal, map[string]string{
		"service": serviceName,
		"method":  method,
		"status":  statusError,
	})
	assert.InDelta(t, float64(3), errMetric.GetCounter().GetValue(), 0.0001)

	unavailMetric := assertMetricMatches(t, metrics, metricErrorsTotal, map[string]string{
		"service":    serviceName,
		"method":     method,
		"error_code": codes.Unavailable.String(),
	})
	assert.InDelta(t, float64(3), unavailMetric.GetCounter().GetValue(), 0.0001)
}

func TestUnaryClientInterceptor_DifferentMethods(t *testing.T) {
	registry := resetMetrics(t)

	const serviceName = "client-service"

	interceptor := UnaryClientInterceptor(serviceName)
	methods := []string{
		"/api.v1.UserService/GetUser",
		"/api.v1.UserService/UpdateUser",
		"/api.v1.UserService/DeleteUser",
	}

	for _, method := range methods {
		invoker := func(
			ctx context.Context,
			method string,
			req, reply interface{},
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			return nil
		}

		err := interceptor(context.Background(), method, nil, nil, nil, invoker)
		require.NoError(t, err)
	}

	metrics, err := registry.Gather()
	require.NoError(t, err)

	for _, expectedMethod := range methods {
		metric := assertMetricMatches(t, metrics, metricRequestsTotal, map[string]string{
			"service": serviceName,
			"method":  expectedMethod,
			"status":  "ok",
		})
		assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
	}
}
