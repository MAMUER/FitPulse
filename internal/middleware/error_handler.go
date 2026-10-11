package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
)

// errorHandlerLog используется функциями без достарёпного логгера: к моменту
// записи ответа заголовки уже отправлены, поэтому остаётся только залогировать.
var errorHandlerLog = logger.New("middleware")

// ErrorHandler — единая точка обработки ошибок для всех ответов
func ErrorHandler(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			if rw.statusCode >= 400 {
				log.Warn("Error response",
					zap.Int("status", rw.statusCode),
					zap.String("path", sanitizeLogValue(r.URL.Path)),
					zap.String("method", sanitizeLogValue(r.Method)),
					zap.String("correlationId", sanitizeLogValue(GetCorrelationID(r.Context()))),
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
	n, err := rw.ResponseWriter.Write(b)
	if err != nil {
		return n, fmt.Errorf("write response: %w", err)
	}
	return n, nil
}

// JSONError отправляет стандартизированный JSON ответ с ошибкой
func JSONError(w http.ResponseWriter, r *http.Request, code int, message string) {
	if code == http.StatusForbidden {
		code = http.StatusNotFound
		message = msgNotFound
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

	if err := json.NewEncoder(w).Encode(body); err != nil {
		errorHandlerLog.Warn("Failed to encode JSON error response", zap.Error(err))
	}
}

// GetRequestID extracts correlation ID from context.
func GetRequestID(ctx context.Context) string {
	return GetCorrelationID(ctx)
}
