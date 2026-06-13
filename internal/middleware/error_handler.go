package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ErrorHandler — единая точка обработки ошибок для всех ответов
func ErrorHandler(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			if rw.statusCode >= 400 {
				log.Warn("Error response",
					zap.Int("status", rw.statusCode),
					zap.String("path", strings.ReplaceAll(strings.ReplaceAll(r.URL.Path, "\n", ""), "\r", "")),
					zap.String("method", r.Method),
					zap.String("correlationId", GetCorrelationID(r.Context())),
				)
			}
		})
	}
}

// responseWriter — обёртка для перехвата статусного кода
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// JSONError отправляет стандартизированный JSON ответ с ошибкой
func JSONError(w http.ResponseWriter, r *http.Request, code int, message string) {
	if code == http.StatusForbidden {
		code = http.StatusNotFound
		message = "Не найдено"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	body := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    http.StatusText(code),
			"message": message,
		},
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"correlationId": GetCorrelationID(r.Context()),
	}

	_ = json.NewEncoder(w).Encode(body)
}

// GetRequestID deprecated: compatibility wrapper around GetCorrelationID(ctx)
func GetRequestID(ctx interface{}) string {
	if ctx == nil {
		return ""
	}
	if ctxValue, ok := ctx.(context.Context); ok {
		return GetCorrelationID(ctxValue)
	}
	return ""
}
