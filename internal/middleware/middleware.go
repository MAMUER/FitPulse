// internal/middleware/middleware.go
package middleware

import (
	"context"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/MAMUER/Project/internal/auth"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestID добавляет уникальный идентификатор запроса
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthMiddleware проверяет JWT токен и добавляет пользователя в контекст
// Требование #4: Заменяем 403 → 404 Not Found
// Требование #8: Middleware блокирует запрос до вызова обработчика
// Требование #7: return после http.Error прекращает выполнение
func AuthMiddleware(secret string, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Debug("Missing authorization header", zap.String("path", r.URL.Path))
				http.Error(w, "not found", http.StatusNotFound)
				return // Требование #7: немедленное завершение
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				log.Debug("Invalid authorization format", zap.String("header", authHeader))
				http.Error(w, "not found", http.StatusNotFound)
				return // Требование #7
			}
			token := parts[1]
			claims, err := auth.ValidateJWT(token, secret)
			if err != nil {
				log.Debug("Invalid token", zap.Error(err), zap.String("path", r.URL.Path))
				http.Error(w, "not found", http.StatusNotFound)
				return // Требование #7
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole проверяет роль пользователя
// Требование #4: Возвращает 404 вместо 403
// Требование #7: return после ошибки
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(RoleKey).(string)
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return // Требование #7
			}
			for _, allowed := range allowedRoles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			// Требование #4: 404 вместо 403
			http.Error(w, "not found", http.StatusNotFound)
		})
	}
}

// LoggingMiddleware логирует запросы с корреляционным ID
func LoggingMiddleware(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			cid := GetCorrelationID(r.Context())

			next.ServeHTTP(w, r)

			log.Info("HTTP request",
				zap.String("correlation_id", cid),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

// RecoveryMiddleware перехватывает паники
// Требование #7: Блокирует генерацию контента после паники
func RecoveryMiddleware(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("Panic recovered",
						zap.Any("panic", rec),
						zap.String("path", r.URL.Path),
						zap.String("stack", string(debug.Stack())),
					)
					w.Header().Set("Content-Type", "text/plain")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte("Internal Server Error"))
					// Требование #7: после паники — возврат, без дальнейшего выполнения
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersMiddleware добавляет заголовки безопасности
// Требование #12: Content Security Policy
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none';")

		// HSTS
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}
