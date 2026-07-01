package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	handler := RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:12345"

	for i := 0; i < 50; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	time.Sleep(1 * time.Second)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	healthReq := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)
	healthReq.RemoteAddr = "192.0.2.1:12345"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, healthReq)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRateLimitDifferentIPs(t *testing.T) {
	handler := RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Два разных IP — каждый имеет свой лимит
	for _, ip := range []string{"192.0.2.1:1", "192.0.2.2:1"} {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
		req.RemoteAddr = ip
		for i := 0; i < 15; i++ {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		}
	}
}

func TestRateLimitHealthAlwaysAllowed(t *testing.T) {
	handler := RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	for i := 0; i < 100; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "health should never be rate limited")
	}
}
