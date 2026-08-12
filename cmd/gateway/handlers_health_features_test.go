package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/middleware"
)

func TestHealthFeatures_ListHealthConditionsHandler(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/health/conditions", nil)

		g.listHealthConditionsHandler(w, req)

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
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"condition_type":"allergy","condition_name":"Peanuts","severity":"medium","is_active":true}`)
		req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/health/conditions", body)
		req.Header.Set("Content-Type", "application/json")

		g.upsertHealthConditionHandler(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("bad_request", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := bytes.NewBufferString(`invalid json`)
		req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/health/conditions", body)
		req.Header.Set("Content-Type", "application/json")

		g.upsertHealthConditionHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHealthFeatures_DeleteHealthConditionHandler(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("condition_id", "cond-123")
		ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user-123")
		req := httptest.NewRequestWithContext(ctx, "DELETE", "/health/conditions/cond-123", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		g.deleteHealthConditionHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "deleted")
	})
}

func TestHealthFeatures_DeleteEntityHandler(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	t.Run("missing_param", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "DELETE", "/health/conditions/", nil)

		g.deleteEntityHandler(w, req, "condition_id", func(id string) error {
			return nil
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete_error", func(t *testing.T) {
		w := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("condition_id", "cond-123")
		ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user-123")
		req := httptest.NewRequestWithContext(ctx, "DELETE", "/health/conditions/cond-123", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		g.deleteEntityHandler(w, req, "condition_id", func(id string) error {
			return assert.AnError
		})

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHealthFeatures_CreateBodyCompositionHandler(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"recorded_at":"2024-01-01","weight_kg":70.5,"height_cm":175}`)
		req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/health/body-composition", body)
		req.Header.Set("Content-Type", "application/json")

		g.createBodyCompositionHandler(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("bad_request", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := bytes.NewBufferString(`invalid json`)
		req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/health/body-composition", body)
		req.Header.Set("Content-Type", "application/json")

		g.createBodyCompositionHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHealthFeatures_ListBodyCompositionHandler(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/health/body-composition?from=2024-01-01&to=2024-12-31", nil)

		g.listBodyCompositionHandler(w, req)

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
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/health/menstrual-cycles", nil)

		g.listMenstrualCyclesHandler(w, req)

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
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"cycle_start_date":"2024-01-01","flow_intensity":"medium"}`)
		req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/health/menstrual-cycles", body)
		req.Header.Set("Content-Type", "application/json")

		g.createMenstrualCycleHandler(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("bad_request", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := bytes.NewBufferString(`invalid json`)
		req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/health/menstrual-cycles", body)
		req.Header.Set("Content-Type", "application/json")

		g.createMenstrualCycleHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHealthFeatures_UpdateMenstrualCycleHandler(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("cycle_id", "cycle-123")
		ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user-123")
		body := bytes.NewBufferString(`{"cycle_start_date":"2024-01-01","flow_intensity":"light"}`)
		req := httptest.NewRequestWithContext(ctx, "PUT", "/health/menstrual-cycles/cycle-123", body)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req.Header.Set("Content-Type", "application/json")

		g.updateMenstrualCycleHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("missing_cycle_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user-123")
		body := bytes.NewBufferString(`{"cycle_start_date":"2024-01-01"}`)
		req := httptest.NewRequestWithContext(ctx, "PUT", "/health/menstrual-cycles/", body)
		req.Header.Set("Content-Type", "application/json")

		g.updateMenstrualCycleHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad_request", func(t *testing.T) {
		w := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("cycle_id", "cycle-123")
		ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user-123")
		body := bytes.NewBufferString(`invalid json`)
		req := httptest.NewRequestWithContext(ctx, "PUT", "/health/menstrual-cycles/cycle-123", body)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req.Header.Set("Content-Type", "application/json")

		g.updateMenstrualCycleHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHealthFeatures_DeleteMenstrualCycleHandler(t *testing.T) {
	log := &logger.Logger{Logger: zap.NewNop()}
	g := &gateway{
		log:        log,
		userClient: &mockUserServiceClient{},
	}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("cycle_id", "cycle-123")
		ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user-123")
		req := httptest.NewRequestWithContext(ctx, "DELETE", "/health/menstrual-cycles/cycle-123", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		g.deleteMenstrualCycleHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "deleted")
	})
}
