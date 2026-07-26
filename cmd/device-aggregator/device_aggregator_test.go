package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MAMUER/project/internal/logger"
)

func setupAggregatorTestServer() *aggregator {
	log := logger.New("device-aggregator-test")
	return &aggregator{
		db:  nil,
		log: log,
	}
}

func TestHealthHandler(t *testing.T) {
	a := setupAggregatorTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)

	a.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestHandleDisconnect_Unauthorized(t *testing.T) {
	a := setupAggregatorTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/disconnect", nil)

	a.handleDisconnect(w, req, func(ctx context.Context, userID string) error {
		return nil
	}, "Fitbit")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleDisconnect_Success(t *testing.T) {
	a := setupAggregatorTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/disconnect", nil)
	req.Header.Set("X-User-ID", "user-123")

	a.handleDisconnect(w, req, func(ctx context.Context, userID string) error {
		return nil
	}, "Fitbit")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "disconnected")
}

func TestHandleDisconnect_Error(t *testing.T) {
	a := setupAggregatorTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/disconnect", nil)
	req.Header.Set("X-User-ID", "user-123")

	a.handleDisconnect(w, req, func(ctx context.Context, userID string) error {
		return assert.AnError
	}, "Fitbit")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleOAuthCallback_MissingCodeOrState(t *testing.T) {
	a := setupAggregatorTestServer()

	t.Run("missing code", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/callback?state=abc", nil)
		a.handleOAuthCallback(w, req, func(ctx context.Context, code, state string) error {
			return nil
		}, "Fitbit")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing state", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/callback?code=abc", nil)
		a.handleOAuthCallback(w, req, func(ctx context.Context, code, state string) error {
			return nil
		}, "Fitbit")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleAuthStart_Unauthorized(t *testing.T) {
	a := setupAggregatorTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/auth", nil)

	a.handleAuthStart(w, req, func(userID string) (string, error) {
		return "", nil
	}, "Fitbit", "fitbit")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIsValidRedirectHost(t *testing.T) {
	a := setupAggregatorTestServer()

	tests := []struct {
		host string
		want bool
	}{
		{"www.fitbit.com", true},
		{"fitbit.com", true},
		{"account.withings.com", true},
		{"withings.net", true},
		{"fittpulse.duckdns.org", true},
		{"evil.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := a.isValidRedirectHost(tt.host)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestListProvidersHandler_Unauthorized(t *testing.T) {
	a := setupAggregatorTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/providers", nil)

	a.listProvidersHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleAuthStart_Success(t *testing.T) {
	a := setupAggregatorTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/auth", nil)
	req.Header.Set("X-User-ID", "user-123")

	a.handleAuthStart(w, req, func(userID string) (string, error) {
		return "https://www.fitbit.com/oauth2/authorize?client_id=test", nil
	}, "Fitbit", "fitbit")

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "devices/auth/fitbit")
}

func TestHandleAuthStart_InvalidRedirectScheme(t *testing.T) {
	a := setupAggregatorTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/auth", nil)
	req.Header.Set("X-User-ID", "user-123")

	a.handleAuthStart(w, req, func(userID string) (string, error) {
		return "http://example.com/auth", nil
	}, "Fitbit", "fitbit")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAuthStart_InvalidRedirectHost(t *testing.T) {
	a := setupAggregatorTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/auth", nil)
	req.Header.Set("X-User-ID", "user-123")

	a.handleAuthStart(w, req, func(userID string) (string, error) {
		return "https://evil.com/auth", nil
	}, "Fitbit", "fitbit")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNewAggregator(t *testing.T) {
	log := logger.New("device-aggregator-test")
	agg := newAggregator(nil, log, nil, nil, nil)
	assert.NotNil(t, agg)
	assert.NotNil(t, agg.log)
}
