package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestExtractPublicHost(t *testing.T) {
	cases := map[string]string{
		"https://fitpulse.example.com":  "fitpulse.example.com",
		"http://localhost:8080":         "localhost:8080",
		"https://a.b.c/x?y=1":           "a.b.c",
		"":                              "",
		"not-a-url":                     "",
		"https://fitpulse.example.com/": "fitpulse.example.com",
	}
	for in, want := range cases {
		assert.Equal(t, want, extractPublicHost(in), "вход: %q", in)
	}
}

func TestMaskRabbitMQURL(t *testing.T) {
	got := maskRabbitMQURL("amqp://guest:guest@localhost:5672/")
	// url.UserPassword percent-кодирует маску: *** -> %2A%2A%2A.
	assert.Contains(t, got, "%2A%2A%2A")
	assert.NotContains(t, got, "guest:guest")
	assert.Contains(t, got, "amqp://guest:")

	// URL без креденшелов возвращается как есть.
	assert.Equal(t, "amqp://localhost:5672/", maskRabbitMQURL("amqp://localhost:5672/"))

	// Битый URL не роняет функцию.
	assert.Equal(t, "<invalid-url>", maskRabbitMQURL("://bad url"))
}

func TestResolveRedirectHost(t *testing.T) {
	host, err := resolveRedirectHost("fitpulse.example.com", nil)
	require.NoError(t, err)
	assert.Equal(t, "fitpulse.example.com", host)

	_, err = resolveRedirectHost("", nil)
	assert.Error(t, err)
}

func TestBuildHTTPRedirectHandler_Health(t *testing.T) {
	log := zap.NewNop()
	handler := buildHTTPRedirectHandler(log, "fitpulse.example.com", "8080")

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status":"ok"`)
}

func TestBuildHTTPRedirectHandler_Redirect(t *testing.T) {
	log := zap.NewNop()
	handler := buildHTTPRedirectHandler(log, "fitpulse.example.com", "8080")

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/profile", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Редирект на https того же хоста (нестандартный порт заменяется на 8443).
	assert.Equal(t, http.StatusMovedPermanently, rr.Code)
	assert.Equal(t, "https://fitpulse.example.com:8443/api/v1/profile", rr.Header().Get("Location"))
}

func TestBuildHTTPRedirectHandler_NoHost(t *testing.T) {
	log := zap.NewNop()
	handler := buildHTTPRedirectHandler(log, "", "8080")

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/profile", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestValidateRequestURI(t *testing.T) {
	ok := httptest.NewRequestWithContext(context.Background(), "GET", "/normal/path", nil)
	require.NoError(t, validateRequestURI(ok))

	// Протокол-относительные URI должны отклоняться.
	bad := httptest.NewRequestWithContext(context.Background(), "GET", "//evil.com/path", nil)
	require.Error(t, validateRequestURI(bad))
}

func TestValkeyAddressDefault(t *testing.T) {
	t.Setenv("VALKEY_HOST", "cache.internal")
	// Порт фиксирован на 6379 (VALKEY_PORT не учитывается).
	assert.Equal(t, "cache.internal:6379", valkeyAddress())
}
