package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRateLimit(t *testing.T) {
	resetRateLimiters()
	log := zap.NewNop()
	handler := RateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	resetRateLimiters()
	log := zap.NewNop()
	handler := RateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	resetRateLimiters()
	log := zap.NewNop()
	handler := RateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestRateLimitRefreshPathAlwaysAllowed(t *testing.T) {
	resetRateLimiters()
	log := zap.NewNop()
	handler := RateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/auth/refresh", nil)
	req.RemoteAddr = "10.0.0.2:12345"

	for i := 0; i < 100; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}
}

func TestAuthRateLimitLoginExceeded(t *testing.T) {
	resetRateLimiters()
	log := zap.NewNop()
	handler := AuthRateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/login", nil)
	req.RemoteAddr = "192.0.2.10:12345"

	// burst = 5: пять запросов проходят, шестой получает 429 + X-Captcha-Required.
	for i := 0; i < 5; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "запрос %d должен пройти", i+1)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.Equal(t, "true", rr.Header().Get("X-Captcha-Required"))
	assert.Contains(t, rr.Body.String(), "Превышен лимит запросов для авторизации")
}

func TestAuthRateLimitRegisterPath(t *testing.T) {
	resetRateLimiters()
	log := zap.NewNop()
	handler := AuthRateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/register", nil)
	req.RemoteAddr = "192.0.2.11:12345"

	for i := 0; i < 5; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestAuthRateLimitOtherPathSkipped(t *testing.T) {
	resetRateLimiters()
	log := zap.NewNop()
	handler := AuthRateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/profile", nil)
	req.RemoteAddr = "192.0.2.12:12345"

	// Неаuth-путь не лимитируется вообще.
	for i := 0; i < 20; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}
}

func TestAuthRateLimitDifferentIPs(t *testing.T) {
	resetRateLimiters()
	log := zap.NewNop()
	handler := AuthRateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, ip := range []string{"192.0.2.20:1", "192.0.2.21:1"} {
		req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/login", nil)
		req.RemoteAddr = ip
		for i := 0; i < 5; i++ {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		}
	}
}

func TestUserRateLimitWithoutUserID(t *testing.T) {
	resetRateLimiters()
	log := zap.NewNop()
	handler := UserRateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/data", nil)
	req.RemoteAddr = "192.0.2.30:12345"

	// Без user_id в контексте запрос всегда проходит.
	for i := 0; i < 10; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}
}

func TestUserRateLimitExemptPaths(t *testing.T) {
	resetRateLimiters()
	log := zap.NewNop()
	handler := UserRateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/health", "/api/v1/auth/refresh"} {
		req := httptest.NewRequestWithContext(context.Background(), "GET", path, nil)
		req.RemoteAddr = "192.0.2.31:12345"
		ctx := context.WithValue(req.Context(), UserIDKey, "user-1")
		req = req.WithContext(ctx)
		for i := 0; i < 10; i++ {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		}
	}
}

func TestUserRateLimitExceeded(t *testing.T) {
	resetRateLimiters()
	log := zap.NewNop()
	handler := UserRateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/data", nil)
	req.RemoteAddr = "192.0.2.32:12345"
	ctx := context.WithValue(req.Context(), UserIDKey, "user-limit-test")
	req = req.WithContext(ctx)

	// burst = 200: 200 запросов проходят, следующий получает 429.
	for i := 0; i < 200; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code, "запрос %d должен пройти", i+1)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.Contains(t, rr.Body.String(), "Превышен лимит запросов для пользователя")
}

func TestUserRateLimitDifferentUsers(t *testing.T) {
	resetRateLimiters()
	log := zap.NewNop()
	handler := UserRateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, userID := range []string{"user-a", "user-b"} {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/data", nil)
		req.RemoteAddr = "192.0.2.33:12345"
		ctx := context.WithValue(req.Context(), UserIDKey, userID)
		req = req.WithContext(ctx)
		for i := 0; i < 10; i++ {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		}
	}
}

func TestResetRateLimiterForIP(t *testing.T) {
	resetRateLimiters()
	log := zap.NewNop()
	handler := AuthRateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/login", nil)
	req.RemoteAddr = "192.0.2.40:12345"

	for i := 0; i < 5; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusTooManyRequests, rr.Code)

	ResetRateLimiterForIP("192.0.2.40")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestResetUserRateLimiterForID(t *testing.T) {
	resetRateLimiters()
	log := zap.NewNop()
	handler := UserRateLimit(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/data", nil)
	req.RemoteAddr = "192.0.2.41:12345"
	ctx := context.WithValue(req.Context(), UserIDKey, "user-reset-test")
	req = req.WithContext(ctx)

	for i := 0; i < 200; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusTooManyRequests, rr.Code)

	ResetUserRateLimiterForID("user-reset-test")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
