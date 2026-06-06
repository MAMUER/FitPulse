package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestErrorHandler_LogsErrors(t *testing.T) {
	core, observed := observer.New(zapcore.WarnLevel)
	log := zap.New(core)

	handler := ErrorHandler(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), RequestIDKey, "test-correlation-id")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	logs := observed.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "Error response", logs[0].Message)
	// zap stores status as int64
	assert.Equal(t, int64(http.StatusInternalServerError), logs[0].ContextMap()["status"])
	assert.Equal(t, "/test", logs[0].ContextMap()["path"])
	assert.Equal(t, "GET", logs[0].ContextMap()["method"])
	// GetRequestID receives context.Context but tries to cast it to string,
	// which fails — so correlation_id will be empty in the log
	assert.Equal(t, "", logs[0].ContextMap()["correlation_id"])
}

func TestErrorHandler_DoesNotLogSuccess(t *testing.T) {
	core, observed := observer.New(zapcore.WarnLevel)
	log := zap.New(core)

	handler := ErrorHandler(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Zero(t, observed.Len(), "should not log successful responses")
}

func TestErrorHandler_DoesNotLogClientErrorBelow400(t *testing.T) {
	core, observed := observer.New(zapcore.WarnLevel)
	log := zap.New(core)

	handler := ErrorHandler(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified) // 304 < 400
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotModified, rr.Code)
	assert.Zero(t, observed.Len())
}

func TestErrorHandler_Logs400Error(t *testing.T) {
	core, observed := observer.New(zapcore.WarnLevel)
	log := zap.New(core)

	handler := ErrorHandler(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/data", nil)
	ctx := context.WithValue(req.Context(), RequestIDKey, "req-123")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	logs := observed.All()
	require.Len(t, logs, 1)
	assert.Equal(t, int64(http.StatusBadRequest), logs[0].ContextMap()["status"])
	assert.Equal(t, "/api/data", logs[0].ContextMap()["path"])
	assert.Equal(t, "POST", logs[0].ContextMap()["method"])
}

func TestErrorHandler_DefaultStatusCode(t *testing.T) {
	core, _ := observer.New(zapcore.WarnLevel)
	log := zap.New(core)

	handler := ErrorHandler(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do not call WriteHeader — default should be 200
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestJSONError_ForbiddenConvertedToNotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), RequestIDKey, "corr-456")
	req = req.WithContext(ctx)

	JSONError(rr, req, http.StatusForbidden, "access denied")

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var body map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, false, body["success"])

	errObj, ok := body["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Not Found", errObj["code"])
	assert.Equal(t, "Не найдено", errObj["message"])

	assert.NotEmpty(t, body["timestamp"])
	// GetRequestID receives context.Context but tries to cast to string,
	// which fails — so correlationId will be empty
	assert.Equal(t, "", body["correlationId"])
}

func TestJSONError_NotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/missing", nil)
	ctx := context.WithValue(req.Context(), RequestIDKey, "corr-789")
	req = req.WithContext(ctx)

	JSONError(rr, req, http.StatusNotFound, "resource not found")

	assert.Equal(t, http.StatusNotFound, rr.Code)

	var body map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, false, body["success"])

	errObj, ok := body["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Not Found", errObj["code"])
	assert.Equal(t, "resource not found", errObj["message"])
}

func TestJSONError_InternalServerError(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api", nil)
	ctx := context.WithValue(req.Context(), RequestIDKey, "corr-err")
	req = req.WithContext(ctx)

	JSONError(rr, req, http.StatusInternalServerError, "db connection failed")

	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var body map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, false, body["success"])

	errObj, ok := body["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Internal Server Error", errObj["code"])
	assert.Equal(t, "db connection failed", errObj["message"])
}

func TestJSONError_BadRequest(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api", nil)

	JSONError(rr, req, http.StatusBadRequest, "invalid input")

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var body map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)

	errObj, ok := body["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Bad Request", errObj["code"])
	assert.Equal(t, "invalid input", errObj["message"])
}

func TestJSONError_TimestampFormat(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)

	JSONError(rr, req, http.StatusNotFound, "error")

	var body map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)

	ts, ok := body["timestamp"].(string)
	require.True(t, ok)
	// RFC3339 format
	_, parseErr := time.Parse(time.RFC3339, ts)
	assert.NoError(t, parseErr, "timestamp should be parseable as RFC3339")
}

func TestJSONError_EmptyCorrelationId(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)

	JSONError(rr, req, http.StatusNotFound, "error")

	var body map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, "", body["correlationId"])
}

func TestGetRequestID(t *testing.T) {
	t.Run("returns string when context is string", func(t *testing.T) {
		result := GetRequestID("test-id-123")
		assert.Equal(t, "test-id-123", result)
	})

	t.Run("returns empty string when context is nil", func(t *testing.T) {
		result := GetRequestID(nil)
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string when context is not string", func(t *testing.T) {
		result := GetRequestID(12345)
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string when context is context.Context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RequestIDKey, "some-id")
		result := GetRequestID(ctx)
		// GetRequestID expects a string, not a context.Context
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string for map", func(t *testing.T) {
		result := GetRequestID(map[string]string{"key": "value"})
		assert.Equal(t, "", result)
	})

	t.Run("returns string for empty string", func(t *testing.T) {
		result := GetRequestID("")
		assert.Equal(t, "", result)
	})
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusCreated)

	assert.Equal(t, http.StatusCreated, rw.statusCode)
	assert.True(t, rw.written)
	assert.Equal(t, http.StatusCreated, rr.Code)
}

func TestResponseWriter_WriteHeader_OnlyCalledOnce(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)
	rw.WriteHeader(http.StatusInternalServerError) // Should be ignored

	assert.Equal(t, http.StatusNotFound, rw.statusCode)
}

func TestResponseWriter_Write(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

	n, err := rw.Write([]byte("hello"))

	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.True(t, rw.written)
	assert.Equal(t, "hello", rr.Body.String())
}

func TestResponseWriter_Write_SetsWrittenFlag(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

	_, _ = rw.Write([]byte("data"))

	assert.True(t, rw.written)
	// statusCode remains default since WriteHeader was not called
	assert.Equal(t, http.StatusOK, rw.statusCode)
}

func TestResponseWriter_WriteHeader_CalledOnceEvenWithMultipleWrites(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusAccepted)
	_, _ = rw.Write([]byte("first"))
	_, _ = rw.Write([]byte("second"))

	assert.Equal(t, http.StatusAccepted, rw.statusCode)
	assert.Equal(t, "firstsecond", rr.Body.String())
}

func TestResponseWriter_Write_WithoutWriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

	// Write without calling WriteHeader first — should set written flag but not call underlying WriteHeader
	_, _ = rw.Write([]byte("body"))

	assert.True(t, rw.written)
	assert.Equal(t, http.StatusOK, rw.statusCode)
	assert.Equal(t, "body", rr.Body.String())
}

func TestErrorHandler_CapturesStatusCodeFromResponseWriter(t *testing.T) {
	core, observed := observer.New(zapcore.WarnLevel)
	log := zap.New(core)

	handler := ErrorHandler(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/endpoint", nil)
	ctx := context.WithValue(req.Context(), RequestIDKey, "req-abc")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	logs := observed.All()
	require.Len(t, logs, 1)
	assert.Equal(t, int64(http.StatusBadRequest), logs[0].ContextMap()["status"])
	assert.Equal(t, "/api/endpoint", logs[0].ContextMap()["path"])
}

func TestJSONError_UnauthorizedNotConverted(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)

	JSONError(rr, req, http.StatusUnauthorized, "unauthorized")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var body map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)

	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "Unauthorized", errObj["code"])
	assert.Equal(t, "unauthorized", errObj["message"])
}

func TestJSONError_ForbiddenWithCustomMessageStillConverted(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/secret", nil)

	// Even with a custom message, 403 gets converted to 404 with "Не найдено"
	JSONError(rr, req, http.StatusForbidden, "you shall not pass")

	assert.Equal(t, http.StatusNotFound, rr.Code)

	var body map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)

	errObj, ok := body["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Не найдено", errObj["message"])
}
