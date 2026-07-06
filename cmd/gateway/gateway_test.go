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

func TestHealthHandler_NoServices(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{log: log}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)

	g.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "gateway")
	assert.Contains(t, w.Body.String(), "degraded")
}

func TestHealthHandler_WithUserClient(t *testing.T) {
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

func TestVerifyUserRole_NilDB(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{log: log, db: nil}

	got := g.verifyUserRole(context.Background(), "user-id", "admin")
	assert.False(t, got)
}

func TestGRPCClientGetters(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}

	t.Run("getBiometricClient_returns_error_when_no_addr", func(t *testing.T) {
		g2 := &gateway{log: log, biometricAddr: ""}
		_, err := g2.getBiometricClient()
		assert.Error(t, err)
	})

	t.Run("getTrainingClient_returns_error_when_no_addr", func(t *testing.T) {
		g2 := &gateway{log: log, trainingAddr: ""}
		_, err := g2.getTrainingClient()
		assert.Error(t, err)
	})
}

func TestCheckGRPCService(t *testing.T) {
	t.Run("nil connection returns down", func(t *testing.T) {
		got := checkGRPCService(nil, "user.UserService")
		assert.Equal(t, "down", got)
	})
}

func TestCheckTCPService(t *testing.T) {
	t.Run("empty address returns down", func(t *testing.T) {
		got := checkTCPService("")
		assert.Equal(t, "down", got)
	})
}
