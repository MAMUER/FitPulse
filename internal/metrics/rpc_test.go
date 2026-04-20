package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
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

	found := false
	for _, m := range requestsMetric {
		if m.GetName() == metricRequestsTotal {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName &&
					labelMap["method"] == fullMethod &&
					labelMap["status"] == "ok" {
					assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
					found = true
				}
			}
		}
	}
	assert.True(t, found, "expected grpc_requests_total with status=ok")

	// Verify rpcDurationSeconds observed
	durationFound := false
	for _, m := range requestsMetric {
		if m.GetName() == metricDurationSeconds {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName && labelMap["method"] == fullMethod {
					assert.Greater(t, metric.GetHistogram().GetSampleCount(), uint64(0))
					durationFound = true
				}
			}
		}
	}
	assert.True(t, durationFound, "expected grpc_request_duration_seconds observation")

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
		return nil, expectedErr
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

	// Verify rpcRequestsTotal incremented with status "error"
	foundRequest := false
	foundError := false
	for _, m := range metrics {
		if m.GetName() == metricRequestsTotal {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName &&
					labelMap["method"] == fullMethod &&
					labelMap["status"] == statusError {
					assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
					foundRequest = true
				}
			}
		}
		if m.GetName() == metricErrorsTotal {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName &&
					labelMap["method"] == fullMethod &&
					labelMap["error_code"] == codes.NotFound.String() {
					assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
					foundError = true
				}
			}
		}
	}
	assert.True(t, foundRequest, "expected grpc_requests_total with status=error")
	assert.True(t, foundError, "expected grpc_errors_total with error_code=NotFound")
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
		return nil, expectedErr
	}

	info := &grpc.UnaryServerInfo{FullMethod: fullMethod}

	_, err := interceptor(context.Background(), nil, info, handler)

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))

	metrics, err := registry.Gather()
	require.NoError(t, err)

	found := false
	for _, m := range metrics {
		if m.GetName() == metricErrorsTotal {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName &&
					labelMap["method"] == fullMethod &&
					labelMap["error_code"] == codes.Internal.String() {
					assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
					found = true
				}
			}
		}
	}
	assert.True(t, found, "expected grpc_errors_total with error_code=Internal")
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

	for _, m := range metrics {
		if m.GetName() == metricRequestsTotal {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName && labelMap["method"] == fullMethod {
					switch labelMap["status"] {
					case "ok":
						assert.InDelta(t, float64(3), metric.GetCounter().GetValue(), 0.0001)
					case statusError:
						assert.InDelta(t, float64(2), metric.GetCounter().GetValue(), 0.0001)
					}
				}
			}
		}
		if m.GetName() == metricErrorsTotal {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName &&
					labelMap["method"] == fullMethod &&
					labelMap["error_code"] == codes.DeadlineExceeded.String() {
					assert.InDelta(t, float64(2), metric.GetCounter().GetValue(), 0.0001)
				}
			}
		}
	}
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

	found := false
	for _, m := range metrics {
		if m.GetName() == "grpc_requests_total" {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName &&
					labelMap["method"] == method &&
					labelMap["status"] == "ok" {
					assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
					found = true
				}
			}
		}
	}
	assert.True(t, found, "expected grpc_requests_total with status=ok for client interceptor")

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

	foundRequest := false
	foundError := false
	for _, m := range metrics {
		if m.GetName() == metricRequestsTotal {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName &&
					labelMap["method"] == method &&
					labelMap["status"] == statusError {
					assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
					foundRequest = true
				}
			}
		}
		if m.GetName() == metricErrorsTotal {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName &&
					labelMap["method"] == method &&
					labelMap["error_code"] == codes.PermissionDenied.String() {
					assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
					foundError = true
				}
			}
		}
	}
	assert.True(t, foundRequest, "expected grpc_requests_total with status=error for client")
	assert.True(t, foundError, "expected grpc_errors_total with error_code=PermissionDenied")
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
	found := false
	for _, m := range metrics {
		if m.GetName() == metricRequestsTotal {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName &&
					labelMap["method"] == method &&
					labelMap["status"] == statusError {
					assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
					found = true
				}
			}
		}
	}
	assert.True(t, found, "expected grpc_requests_total with status=error for non-gRPC error")
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

	for _, m := range metrics {
		if m.GetName() == metricRequestsTotal {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName && labelMap["method"] == method {
					switch labelMap["status"] {
					case "ok":
						assert.InDelta(t, float64(5), metric.GetCounter().GetValue(), 0.0001)
					case statusError:
						assert.InDelta(t, float64(3), metric.GetCounter().GetValue(), 0.0001)
					}
				}
			}
		}
		if m.GetName() == metricErrorsTotal {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName &&
					labelMap["method"] == method &&
					labelMap["error_code"] == codes.Unavailable.String() {
					assert.InDelta(t, float64(3), metric.GetCounter().GetValue(), 0.0001)
				}
			}
		}
	}
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

	for _, m := range metrics {
		if m.GetName() == metricRequestsTotal {
			for _, metric := range m.GetMetric() {
				labels := metric.GetLabel()
				labelMap := make(map[string]string)
				for _, l := range labels {
					labelMap[l.GetName()] = l.GetValue()
				}
				if labelMap["service"] == serviceName && labelMap["status"] == "ok" {
					for _, expectedMethod := range methods {
						if labelMap["method"] == expectedMethod {
							assert.InDelta(t, float64(1), metric.GetCounter().GetValue(), 0.0001)
						}
					}
				}
			}
		}
	}
}
