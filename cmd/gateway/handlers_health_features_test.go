package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
)

func newHealthFeatureGateway() *gateway {
	return &gateway{
		log:        &logger.Logger{Logger: zap.NewNop()},
		userClient: &mockUserServiceClient{},
	}
}

func healthRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()

	var reader io.Reader

	if body != "" {
		reader = bytes.NewBufferString(body)
	}

	req := httptest.NewRequestWithContext(withUserID(context.Background(), "user-123"), method, target, reader)

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	return req
}

func healthRequestWithParam(t *testing.T, method, target, body, param, value string) *http.Request {
	t.Helper()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(param, value)

	req := healthRequest(t, method, target, body)

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestHealthFeatures_ListHealthConditionsHandler(t *testing.T) {
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

func TestHealthFeatures_UpsertHealthConditionHandler(t *testing.T) {
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

func TestHealthFeatures_DeleteHealthConditionHandler(t *testing.T) {
	g := newHealthFeatureGateway()

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequestWithParam(t, "DELETE", "/health/conditions/cond-123", "", "condition_id", "cond-123")

		g.deleteHealthConditionHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "deleted")
	})
}

func TestHealthFeatures_DeleteEntityHandler(t *testing.T) {
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
			return assert.AnError
		})

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHealthFeatures_CreateBodyCompositionHandler(t *testing.T) {
	g := newHealthFeatureGateway()

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequest(t, "POST", "/health/body-composition", `{"recorded_at":"2024-01-01","weight_kg":70.5,"height_cm":175}`)

		g.createBodyCompositionHandler(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("bad_request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequest(t, "POST", "/health/body-composition", `invalid json`)

		g.createBodyCompositionHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHealthFeatures_ListBodyCompositionHandler(t *testing.T) {
	g := newHealthFeatureGateway()

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.listBodyCompositionHandler(w, healthRequest(t, "GET", "/health/body-composition?from=2024-01-01&to=2024-12-31", ""))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/health/body-composition", nil)

		g.listBodyCompositionHandler(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHealthFeatures_ListMenstrualCyclesHandler(t *testing.T) {
	g := newHealthFeatureGateway()

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()

		g.listMenstrualCyclesHandler(w, healthRequest(t, "GET", "/health/menstrual-cycles", ""))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/health/menstrual-cycles", nil)

		g.listMenstrualCyclesHandler(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHealthFeatures_CreateMenstrualCycleHandler(t *testing.T) {
	g := newHealthFeatureGateway()

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequest(t, "POST", "/health/menstrual-cycles", `{"cycle_start_date":"2024-01-01","flow_intensity":"medium"}`)

		g.createMenstrualCycleHandler(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("bad_request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequest(t, "POST", "/health/menstrual-cycles", `invalid json`)

		g.createMenstrualCycleHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHealthFeatures_UpdateMenstrualCycleHandler(t *testing.T) {
	g := newHealthFeatureGateway()

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequestWithParam(t, "PUT", "/health/menstrual-cycles/cycle-123", `{"cycle_start_date":"2024-01-01","flow_intensity":"light"}`, "cycle_id", "cycle-123")

		g.updateMenstrualCycleHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("missing_cycle_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequest(t, "PUT", "/health/menstrual-cycles/", `{"cycle_start_date":"2024-01-01"}`)

		g.updateMenstrualCycleHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad_request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequestWithParam(t, "PUT", "/health/menstrual-cycles/cycle-123", `invalid json`, "cycle_id", "cycle-123")

		g.updateMenstrualCycleHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHealthFeatures_DeleteMenstrualCycleHandler(t *testing.T) {
	g := newHealthFeatureGateway()

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := healthRequestWithParam(t, "DELETE", "/health/menstrual-cycles/cycle-123", "", "cycle_id", "cycle-123")

		g.deleteMenstrualCycleHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "deleted")
	})
}
