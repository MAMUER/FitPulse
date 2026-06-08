package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const correlationIDUnknown = "unknown"

// CorrelationIDHTTP добавляет/проксирует X-Correlation-ID в HTTP-запросах
func CorrelationIDHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cid := r.Header.Get("X-Correlation-ID")
		if cid == "" {
			cid = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), CorrelationIDKey, cid)
		w.Header().Set("X-Correlation-ID", cid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CorrelationIDGRPC проксирует correlation ID между gRPC-сервисами
func CorrelationIDGRPC() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		cid := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("x-correlation-id"); len(vals) > 0 {
				cid = vals[0]
			}
		}
		if cid == "" {
			cid = uuid.New().String()
		}

		// Сохраняем в контекст для логов
		ctx = context.WithValue(ctx, CorrelationIDKey, cid)
		// Передаём дальше по цепочке вызовов
		ctx = metadata.AppendToOutgoingContext(ctx, "x-correlation-id", cid)

		return handler(ctx, req)
	}
}

// CorrelationIDGRPCClient injects correlation ID from context into outgoing gRPC metadata
func CorrelationIDGRPCClient() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		cid := GetCorrelationID(ctx)
		if cid != "" && cid != correlationIDUnknown {
			ctx = metadata.AppendToOutgoingContext(ctx, "x-correlation-id", cid)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// GetCorrelationID безопасно извлекает ID из контекста (для логов)
func GetCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return correlationIDUnknown
	}
	if cid, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return cid
	}
	return correlationIDUnknown
}

// GetUserID безопасно извлекает ID пользователя из контекста
func GetUserID(ctx context.Context) string {
	if uid, ok := ctx.Value(UserIDKey).(string); ok {
		return uid
	}
	return "anonymous"
}
