// Package resilience provides gRPC client interceptors for circuit breaker protection.
package resilience

import (
	"context"
	"errors"
	"strings"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// CircuitBreakerLogger is a minimal logger interface for circuit breaker interceptor.
type CircuitBreakerLogger interface {
	Warn(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
}

// UnaryClientInterceptor returns a gRPC unary client interceptor that wraps calls
// with a per-service circuit breaker. Service name is extracted from the full method
// (e.g. "/user.UserService/GetUserClaims" -> "user.UserService").
func UnaryClientInterceptor(log CircuitBreakerLogger) grpc.UnaryClientInterceptor {
	breakers := &sync.Map{}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		service := extractService(method)
		cb := loadCircuitBreaker(breakers, service)
		err := cb.Execute(func() error {
			return invoker(ctx, method, req, reply, cc, opts...)
		})
		if err != nil {
			if errors.Is(err, ErrCircuitOpen) {
				log.Warn("gRPC circuit breaker open",
					zap.String("service", service),
					zap.String("method", method),
				)
			}
			return err
		}
		return nil
	}
}

func extractService(method string) string {
	if method == "" {
		return "unknown"
	}
	parts := strings.Split(method, "/")
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(method)
}

func loadCircuitBreaker(breakers *sync.Map, name string) *CircuitBreaker {
	if v, ok := breakers.Load(name); ok {
		return v.(*CircuitBreaker)
	}
	cb := NewCircuitBreaker(name, DefaultConfig())
	breakers.Store(name, cb)
	return cb
}
