// internal/middleware/correlation.go
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// CorrelationID добавляет X-Correlation-ID в контекст запроса
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cid := r.Header.Get("X-Correlation-ID")
		if cid == "" {
			cid = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), CorrelationIDKey, cid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCorrelationID извлекает корреляционный ID из контекста
func GetCorrelationID(ctx context.Context) string {
	if cid, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return cid
	}
	return ""
}

// GetUserID извлекает ID пользователя из контекста
func GetUserID(ctx context.Context) string {
	if uid, ok := ctx.Value(UserIDKey).(string); ok {
		return uid
	}
	return ""
}

// WithCorrelationID добавляет корреляционный ID в контекст (для тестов)
func WithCorrelationID(ctx context.Context, cid string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, cid)
}
