// Package metrics provides Prometheus metrics collection and RPC instrumentation.
package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	rpcRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests by service, method and status",
		},
		[]string{"service", "method", "status"},
	)

	rpcDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "Histogram of gRPC request durations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method"},
	)

	rpcErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_errors_total",
			Help: "Total number of gRPC errors by service, method and error code",
		},
		[]string{"service", "method", "error_code"},
	)
)

// UnaryServerInterceptor возвращает interceptor для сбора метрик gRPC
func UnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		// Извлекаем статус для метрик
		statusCode := "ok"
		errorCode := ""
		if err != nil {
			st, _ := status.FromError(err)
			statusCode = "error"
			errorCode = st.Code().String()
			rpcErrorsTotal.WithLabelValues(serviceName, info.FullMethod, errorCode).Inc()
		}

		rpcRequestsTotal.WithLabelValues(serviceName, info.FullMethod, statusCode).Inc()
		rpcDurationSeconds.WithLabelValues(serviceName, info.FullMethod).Observe(duration.Seconds())

		return resp, err
	}
}

// UnaryClientInterceptor возвращает interceptor для клиентских вызовов
func UnaryClientInterceptor(serviceName string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(start)

		statusCode := "ok"
		errorCode := ""
		if err != nil {
			st, _ := status.FromError(err)
			statusCode = "error"
			errorCode = st.Code().String()
			rpcErrorsTotal.WithLabelValues(serviceName, method, errorCode).Inc()
		}

		rpcRequestsTotal.WithLabelValues(serviceName, method, statusCode).Inc()
		rpcDurationSeconds.WithLabelValues(serviceName, method).Observe(duration.Seconds())

		return err
	}
}
