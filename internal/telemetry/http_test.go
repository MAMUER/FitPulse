package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"

	"github.com/MAMUER/project/internal/logger"
)

func TestHTTPMiddleware_PassesThrough(t *testing.T) {
	log := logger.New("test")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("accepted"))
	})

	handler := HTTPMiddleware(log)(next)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/data", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	assert.Equal(t, "value", rr.Header().Get("X-Custom"))
	assert.Equal(t, "accepted", rr.Body.String())
}

func TestHTTPMiddleware_NoSpan(t *testing.T) {
	log := logger.New("test")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := HTTPMiddleware(log)(next)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("X-Trace-ID"))
}

func TestHTTPMiddleware_WithValidSpan(t *testing.T) {
	log := logger.New("test")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := HTTPMiddleware(log)(next)

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{1},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
	req := httptest.NewRequestWithContext(ctx, "GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NotEmpty(t, rr.Header().Get("X-Trace-ID"))
}
