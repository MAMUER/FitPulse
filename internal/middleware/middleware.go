// Package middleware provides HTTP middleware for authentication, logging, and security.
package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/MAMUER/project/internal/auth"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
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
func AuthMiddleware(secret string, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Debug("Missing authorization header", zap.String("path", sanitizeLogValue(r.URL.Path)))
				http.Error(w, "Не найдено", http.StatusNotFound)
				return
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				log.Debug("Invalid authorization format")
				http.Error(w, "Не найдено", http.StatusNotFound)
				return
			}
			token := parts[1]
			claims, err := auth.ValidateJWT(token, secret)
			if err != nil {
				log.Debug("Invalid token", zap.Error(err), zap.String("path", sanitizeLogValue(r.URL.Path)))
				http.Error(w, "Не найдено", http.StatusNotFound)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole проверяет роль пользователя
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(RoleKey).(string)
			if !ok {
				http.Error(w, "Не найдено", http.StatusNotFound)
				return
			}
			for _, allowed := range allowedRoles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "Не найдено", http.StatusNotFound)
		})
	}
}

// LoggingMiddleware логирует запросы с корреляционным ID
func LoggingMiddleware(log *zap.Logger, requestDuration *prometheus.HistogramVec, requestTotal *prometheus.CounterVec, errorTotal *prometheus.CounterVec) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			cid := GetCorrelationID(r.Context())
			userID := GetUserID(r.Context())

			// Wrap response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			// Record metrics
			statusStr := fmt.Sprintf("%d", rw.statusCode)
			if requestDuration != nil {
				requestDuration.WithLabelValues("gateway", r.URL.Path, r.Method, statusStr).Observe(duration.Seconds())
			}
			if requestTotal != nil {
				requestTotal.WithLabelValues("gateway", r.URL.Path, r.Method).Inc()
			}

			if rw.statusCode >= 400 && errorTotal != nil {
				errorTotal.WithLabelValues("gateway", statusStr, r.URL.Path).Inc()
			}

			// Log in structured format
			log.Info("HTTP_REQUEST",
				zap.String("correlationId", sanitizeLogValue(cid)),
				zap.String("userId", userID),
				zap.String("action", "HTTP_REQUEST"),
				zap.Int64("durationMs", duration.Milliseconds()),
				zap.String("endpoint", sanitizeLogValue(r.URL.Path)),
				zap.String("method", sanitizeLogValue(r.Method)),
				zap.Int("statusCode", rw.statusCode),
				zap.String("userAgent", sanitizeLogValue(r.Header.Get("User-Agent"))),
				zap.String("ip", sanitizeLogValue(getClientIP(r))),
			)
		})
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP if multiple
		ips := strings.Split(xff, ",")
		return sanitizeLogValue(strings.TrimSpace(ips[0]))
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return sanitizeLogValue(strings.TrimSpace(xri))
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return sanitizeLogValue(r.RemoteAddr)
	}
	return sanitizeLogValue(ip)
}

func sanitizeLogValue(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\n", ""), "\r", "")
}

// RecoveryMiddleware перехватывает паники
func RecoveryMiddleware(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("Panic recovered",
						zap.Any("panic", rec),
						zap.String("path", sanitizeLogValue(r.URL.Path)),
						zap.String("stack", string(debug.Stack())),
					)
					w.Header().Set("Content-Type", "text/plain")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte("Internal Server Error"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersMiddleware добавляет заголовки безопасности
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
