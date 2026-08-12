package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
)

func TestHealth_CheckGRPCServiceNilConnection(t *testing.T) {
	got := checkGRPCService(nil, "user.UserService")
	assert.Equal(t, "down", got)
}

func TestHealth_CheckTCPServiceEmptyAddress(t *testing.T) {
	got := checkTCPService("")
	assert.Equal(t, "down", got)
}

func TestHealth_HandlerNoServices(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{log: log}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)

	g.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "gateway")
	assert.Contains(t, w.Body.String(), "degraded")
}

func TestHealth_HandlerWithUserClient(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)

	g.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestHealth_HandlerResponseFormat(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)

	g.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "timestamp")
	assert.Contains(t, w.Body.String(), "services")
	assert.Contains(t, w.Body.String(), "classifier")
	assert.Contains(t, w.Body.String(), "ml_generator")
}
