package main

import (
	"context"
	"errors"
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
		got := checkTCPService(zap.NewNop(), "")
		assert.Equal(t, "down", got)
	})
}

func TestListHealthConditionsHandler(t *testing.T) {
	g := newHealthFeatureGateway()

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.listHealthConditionsHandler(w, healthRequest(t, "GET", "/health/conditions", ""))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/health/conditions", nil)

		g.listHealthConditionsHandler(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestUpsertHealthConditionHandler(t *testing.T) {
	g := newHealthFeatureGateway()

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequest(t, "POST", "/health/conditions", `{"condition_type":"allergy","condition_name":"Peanuts","severity":"medium","is_active":true}`)

		g.upsertHealthConditionHandler(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("bad_request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequest(t, "POST", "/health/conditions", `invalid json`)

		g.upsertHealthConditionHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDeleteHealthConditionHandler(t *testing.T) {
	g := newHealthFeatureGateway()

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequestWithParam(t, "DELETE", "/health/conditions/cond-123", "", "condition_id", "cond-123")

		g.deleteHealthConditionHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "deleted")
	})
}

func TestDeleteEntityHandler(t *testing.T) {
	g := newHealthFeatureGateway()

	t.Run("missing_param", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequest(t, "DELETE", "/health/conditions/", "")

		g.deleteEntityHandler(w, req, "condition_id", func(id string) error {
			return nil
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete_error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequestWithParam(t, "DELETE", "/health/conditions/cond-123", "", "condition_id", "cond-123")

		g.deleteEntityHandler(w, req, "condition_id", func(id string) error {
			return errors.New("db error")
		})

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
