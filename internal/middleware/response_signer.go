package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/MAMUER/Project/internal/auth"
	"go.uber.org/zap"
)

// responseRecorder перехватывает тело ответа для последующей подписи
type responseRecorder struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	written    bool
	headers    http.Header
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
		headers:        make(http.Header),
	}
}

func (r *responseRecorder) Header() http.Header {
	return r.headers
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(code int) {
	if !r.written {
		r.statusCode = code
		r.written = true
	}
}

// replay отправляет перехваченный ответ на оригинальный ResponseWriter
func (r *responseRecorder) replay(signature, algorithm string) {
	// Копируем заголовки
	for k, v := range r.headers {
		r.ResponseWriter.Header()[k] = v
	}
	// Добавляем подпись
	if signature != "" {
		r.ResponseWriter.Header().Set("X-Response-Signature", signature)
		r.ResponseWriter.Header().Set("X-Signature-Algorithm", algorithm)
	}
	r.ResponseWriter.WriteHeader(r.statusCode)
	_, _ = r.body.WriteTo(r.ResponseWriter)
}

// SignCriticalResponses подписывает JSON ответы для критических эндпоинтов
// Требование #11: Сервер serializes JSON first, then signs the exact bytes sent to client
func SignCriticalResponses(secret string, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := newResponseRecorder(w)
			next.ServeHTTP(recorder, r)

			// Подписываем только успешные JSON ответы
			if recorder.statusCode >= 200 && recorder.statusCode < 300 {
				contentType := recorder.headers.Get("Content-Type")
				if contentType == "application/json" || (contentType == "" && recorder.body.Len() > 0) {
					// Подписываем ТОЧНО ТЕ БАЙТЫ, которые будут отправлены клиенту
					signature, err := auth.SignResponse(recorder.body.Bytes(), secret)
					if err != nil {
						log.Warn("Failed to sign response", zap.Error(err))
						recorder.replay("", "")
						return
					}
					recorder.replay(signature, "HMAC-SHA256")
					return
				}
			}

			recorder.replay("", "")
		})
	}
}

// SignAndSendJSON — helper для хендлеров: сериализует, подписывает, отправляет
// Требование #11: Гарантирует, что подпись соответствует отправленным байтам
func SignAndSendJSON(w http.ResponseWriter, data interface{}, secret string, log *zap.Logger) error {
	// 1. Сериализуем в байты
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		log.Error("Failed to marshal JSON for signing", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return err
	}

	// 2. Подписываем байты
	signature, err := auth.SignResponse(jsonBytes, secret)
	if err != nil {
		log.Warn("Failed to sign response", zap.Error(err))
		// Отправляем без подписи — не критично
	} else {
		w.Header().Set("X-Response-Signature", signature)
		w.Header().Set("X-Signature-Algorithm", "HMAC-SHA256")
	}

	// 3. Отправляем ТЕ ЖЕ байты, что были подписаны
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(jsonBytes)
	return err
}
