package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
)

type mockUserClientHealth struct {
	userpb.UserServiceClient
	listHC   func(context.Context, *userpb.ListHealthConditionsRequest, ...grpc.CallOption) (*userpb.ListHealthConditionsResponse, error)
	upsertHC func(context.Context, *userpb.UpsertHealthConditionRequest, ...grpc.CallOption) (*userpb.HealthCondition, error)
	delHC    func(context.Context, *userpb.DeleteHealthConditionRequest, ...grpc.CallOption) (*userpb.DeleteHealthConditionResponse, error)
	createBC func(context.Context, *userpb.CreateBodyCompositionRequest, ...grpc.CallOption) (*userpb.BodyCompositionRecord, error)
	listBC   func(context.Context, *userpb.ListBodyCompositionRequest, ...grpc.CallOption) (*userpb.ListBodyCompositionResponse, error)
	listMC   func(context.Context, *userpb.ListMenstrualCyclesRequest, ...grpc.CallOption) (*userpb.ListMenstrualCyclesResponse, error)
	createMC func(context.Context, *userpb.CreateMenstrualCycleRequest, ...grpc.CallOption) (*userpb.MenstrualCycle, error)
	updateMC func(context.Context, *userpb.UpdateMenstrualCycleRequest, ...grpc.CallOption) (*userpb.MenstrualCycle, error)
	delMC    func(context.Context, *userpb.DeleteMenstrualCycleRequest, ...grpc.CallOption) (*userpb.DeleteMenstrualCycleResponse, error)
}

func (m *mockUserClientHealth) ListHealthConditions(ctx context.Context, in *userpb.ListHealthConditionsRequest, opts ...grpc.CallOption) (*userpb.ListHealthConditionsResponse, error) {
	if m.listHC != nil {
		return m.listHC(ctx, in, opts...)
	}
	return &userpb.ListHealthConditionsResponse{Conditions: []*userpb.HealthCondition{{Id: "1"}}, Total: 1}, nil
}
func (m *mockUserClientHealth) UpsertHealthCondition(ctx context.Context, in *userpb.UpsertHealthConditionRequest, opts ...grpc.CallOption) (*userpb.HealthCondition, error) {
	if m.upsertHC != nil {
		return m.upsertHC(ctx, in, opts...)
	}
	return &userpb.HealthCondition{Id: "1"}, nil
}
func (m *mockUserClientHealth) DeleteHealthCondition(ctx context.Context, in *userpb.DeleteHealthConditionRequest, opts ...grpc.CallOption) (*userpb.DeleteHealthConditionResponse, error) {
	if m.delHC != nil {
		return m.delHC(ctx, in, opts...)
	}
	return &userpb.DeleteHealthConditionResponse{}, nil
}
func (m *mockUserClientHealth) CreateBodyComposition(ctx context.Context, in *userpb.CreateBodyCompositionRequest, opts ...grpc.CallOption) (*userpb.BodyCompositionRecord, error) {
	if m.createBC != nil {
		return m.createBC(ctx, in, opts...)
	}
	return &userpb.BodyCompositionRecord{Id: "1"}, nil
}
func (m *mockUserClientHealth) ListBodyComposition(ctx context.Context, in *userpb.ListBodyCompositionRequest, opts ...grpc.CallOption) (*userpb.ListBodyCompositionResponse, error) {
	if m.listBC != nil {
		return m.listBC(ctx, in, opts...)
	}
	return &userpb.ListBodyCompositionResponse{Records: []*userpb.BodyCompositionRecord{{Id: "1"}}, Total: 1}, nil
}
func (m *mockUserClientHealth) ListMenstrualCycles(ctx context.Context, in *userpb.ListMenstrualCyclesRequest, opts ...grpc.CallOption) (*userpb.ListMenstrualCyclesResponse, error) {
	if m.listMC != nil {
		return m.listMC(ctx, in, opts...)
	}
	return &userpb.ListMenstrualCyclesResponse{Cycles: []*userpb.MenstrualCycle{{Id: "1"}}, Total: 1}, nil
}
func (m *mockUserClientHealth) CreateMenstrualCycle(ctx context.Context, in *userpb.CreateMenstrualCycleRequest, opts ...grpc.CallOption) (*userpb.MenstrualCycle, error) {
	if m.createMC != nil {
		return m.createMC(ctx, in, opts...)
	}
	return &userpb.MenstrualCycle{Id: "1"}, nil
}
func (m *mockUserClientHealth) UpdateMenstrualCycle(ctx context.Context, in *userpb.UpdateMenstrualCycleRequest, opts ...grpc.CallOption) (*userpb.MenstrualCycle, error) {
	if m.updateMC != nil {
		return m.updateMC(ctx, in, opts...)
	}
	return &userpb.MenstrualCycle{Id: "1"}, nil
}
func (m *mockUserClientHealth) DeleteMenstrualCycle(ctx context.Context, in *userpb.DeleteMenstrualCycleRequest, opts ...grpc.CallOption) (*userpb.DeleteMenstrualCycleResponse, error) {
	if m.delMC != nil {
		return m.delMC(ctx, in, opts...)
	}
	return &userpb.DeleteMenstrualCycleResponse{}, nil
}

func TestHealthHandlers(t *testing.T) {
	g := newTestGateway()
	g.userClient = &mockUserClientHealth{}

	// list health
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodGet, "/api/v1/health/conditions", nil)
	w := httptest.NewRecorder()
	g.listHealthConditionsHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/health/conditions", nil)
	w = httptest.NewRecorder()
	g.listHealthConditionsHandler(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	g.userClient = &mockUserClientHealth{listHC: func(context.Context, *userpb.ListHealthConditionsRequest, ...grpc.CallOption) (*userpb.ListHealthConditionsResponse, error) {
		return nil, assert.AnError
	}}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodGet, "/api/v1/health/conditions", nil)
	w = httptest.NewRecorder()
	g.listHealthConditionsHandler(w, req)
	require.NotEqual(t, http.StatusOK, w.Code)

	// upsert
	g.userClient = &mockUserClientHealth{}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPost, "/api/v1/health/conditions", bytes.NewBufferString(`{"condition_type":"a"}`))
	w = httptest.NewRecorder()
	g.upsertHealthConditionHandler(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/health/conditions", bytes.NewBufferString(`{}`))
	w = httptest.NewRecorder()
	g.upsertHealthConditionHandler(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPost, "/api/v1/health/conditions", bytes.NewBufferString(`bad`))
	w = httptest.NewRecorder()
	g.upsertHealthConditionHandler(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	g.userClient = &mockUserClientHealth{upsertHC: func(context.Context, *userpb.UpsertHealthConditionRequest, ...grpc.CallOption) (*userpb.HealthCondition, error) {
		return nil, status.Error(codes.Internal, "e")
	}}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPost, "/api/v1/health/conditions", bytes.NewBufferString(`{}`))
	w = httptest.NewRecorder()
	g.upsertHealthConditionHandler(w, req)
	require.NotEqual(t, http.StatusOK, w.Code)
}

func TestDeleteEntityHandler(t *testing.T) {
	g := newTestGateway()
	// no user
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/health/conditions/1", nil)
	rctx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	g.deleteEntityHandler(w, req, "condition_id", func(string) error { return nil })
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// no param
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodDelete, "/api/v1/health/conditions/", nil)
	rctx = chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	g.deleteEntityHandler(w, req, "condition_id", func(string) error { return nil })
	require.Equal(t, http.StatusBadRequest, w.Code)

	// error
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodDelete, "/api/v1/health/conditions/1", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("condition_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	g.deleteEntityHandler(w, req, "condition_id", func(string) error { return assert.AnError })
	require.NotEqual(t, http.StatusOK, w.Code)

	// success
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodDelete, "/api/v1/health/conditions/1", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("condition_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	g.deleteEntityHandler(w, req, "condition_id", func(string) error { return nil })
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHealthBodyAndMenstrual(t *testing.T) {
	g := newTestGateway()
	g.userClient = &mockUserClientHealth{}

	// create body
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPost, "/api/v1/health/body-composition", bytes.NewBufferString(`{"weight_kg":70}`))
	w := httptest.NewRecorder()
	g.createBodyCompositionHandler(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/health/body-composition", bytes.NewBufferString(`{}`))
	w = httptest.NewRecorder()
	g.createBodyCompositionHandler(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPost, "/api/v1/health/body-composition", bytes.NewBufferString(`bad`))
	w = httptest.NewRecorder()
	g.createBodyCompositionHandler(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	g.userClient = &mockUserClientHealth{createBC: func(context.Context, *userpb.CreateBodyCompositionRequest, ...grpc.CallOption) (*userpb.BodyCompositionRecord, error) {
		return nil, status.Error(codes.Internal, "e")
	}}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPost, "/api/v1/health/body-composition", bytes.NewBufferString(`{}`))
	w = httptest.NewRecorder()
	g.createBodyCompositionHandler(w, req)
	require.NotEqual(t, http.StatusOK, w.Code)

	// list body
	g.userClient = &mockUserClientHealth{}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodGet, "/api/v1/health/body-composition?limit=50", nil)
	w = httptest.NewRecorder()
	g.listBodyCompositionHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/health/body-composition", nil)
	w = httptest.NewRecorder()
	g.listBodyCompositionHandler(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	g.userClient = &mockUserClientHealth{listBC: func(context.Context, *userpb.ListBodyCompositionRequest, ...grpc.CallOption) (*userpb.ListBodyCompositionResponse, error) {
		return nil, assert.AnError
	}}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodGet, "/api/v1/health/body-composition", nil)
	w = httptest.NewRecorder()
	g.listBodyCompositionHandler(w, req)
	require.NotEqual(t, http.StatusOK, w.Code)
}

func TestMenstrualHandlers(t *testing.T) {
	g := newTestGateway()
	g.userClient = &mockUserClientHealth{}

	// list
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodGet, "/api/v1/health/menstrual-cycles", nil)
	w := httptest.NewRecorder()
	g.listMenstrualCyclesHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/health/menstrual-cycles", nil)
	w = httptest.NewRecorder()
	g.listMenstrualCyclesHandler(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	g.userClient = &mockUserClientHealth{listMC: func(context.Context, *userpb.ListMenstrualCyclesRequest, ...grpc.CallOption) (*userpb.ListMenstrualCyclesResponse, error) {
		return nil, assert.AnError
	}}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodGet, "/api/v1/health/menstrual-cycles", nil)
	w = httptest.NewRecorder()
	g.listMenstrualCyclesHandler(w, req)
	require.NotEqual(t, http.StatusOK, w.Code)

	// create
	g.userClient = &mockUserClientHealth{}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPost, "/api/v1/health/menstrual-cycles", bytes.NewBufferString(`{"cycle_start_date":"2026-01-01"}`))
	w = httptest.NewRecorder()
	g.createMenstrualCycleHandler(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/health/menstrual-cycles", bytes.NewBufferString(`{}`))
	w = httptest.NewRecorder()
	g.createMenstrualCycleHandler(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPost, "/api/v1/health/menstrual-cycles", bytes.NewBufferString(`bad`))
	w = httptest.NewRecorder()
	g.createMenstrualCycleHandler(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	g.userClient = &mockUserClientHealth{createMC: func(context.Context, *userpb.CreateMenstrualCycleRequest, ...grpc.CallOption) (*userpb.MenstrualCycle, error) {
		return nil, status.Error(codes.Internal, "e")
	}}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPost, "/api/v1/health/menstrual-cycles", bytes.NewBufferString(`{}`))
	w = httptest.NewRecorder()
	g.createMenstrualCycleHandler(w, req)
	require.NotEqual(t, http.StatusOK, w.Code)

	// update
	g.userClient = &mockUserClientHealth{}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPut, "/api/v1/health/menstrual-cycles/1", bytes.NewBufferString(`{}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("cycle_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	g.updateMenstrualCycleHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/health/menstrual-cycles/1", bytes.NewBufferString(`{}`))
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("cycle_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	g.updateMenstrualCycleHandler(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPut, "/api/v1/health/menstrual-cycles/", bytes.NewBufferString(`{}`))
	rctx = chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	g.updateMenstrualCycleHandler(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPut, "/api/v1/health/menstrual-cycles/1", bytes.NewBufferString(`bad`))
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("cycle_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	g.updateMenstrualCycleHandler(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	g.userClient = &mockUserClientHealth{updateMC: func(context.Context, *userpb.UpdateMenstrualCycleRequest, ...grpc.CallOption) (*userpb.MenstrualCycle, error) {
		return nil, status.Error(codes.Internal, "e")
	}}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodPut, "/api/v1/health/menstrual-cycles/1", bytes.NewBufferString(`{}`))
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("cycle_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	g.updateMenstrualCycleHandler(w, req)
	require.NotEqual(t, http.StatusOK, w.Code)

	// delete hc
	g.userClient = &mockUserClientHealth{}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodDelete, "/api/v1/health/conditions/1", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("condition_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	g.deleteHealthConditionHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	g.userClient = &mockUserClientHealth{delHC: func(context.Context, *userpb.DeleteHealthConditionRequest, ...grpc.CallOption) (*userpb.DeleteHealthConditionResponse, error) {
		return nil, assert.AnError
	}}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodDelete, "/api/v1/health/conditions/1", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("condition_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	g.deleteHealthConditionHandler(w, req)
	require.NotEqual(t, http.StatusOK, w.Code)

	// delete mc
	g.userClient = &mockUserClientHealth{}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodDelete, "/api/v1/health/menstrual-cycles/1", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("cycle_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	g.deleteMenstrualCycleHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	g.userClient = &mockUserClientHealth{delMC: func(context.Context, *userpb.DeleteMenstrualCycleRequest, ...grpc.CallOption) (*userpb.DeleteMenstrualCycleResponse, error) {
		return nil, assert.AnError
	}}
	req = httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "u1"), http.MethodDelete, "/api/v1/health/menstrual-cycles/1", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("cycle_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	g.deleteMenstrualCycleHandler(w, req)
	require.NotEqual(t, http.StatusOK, w.Code)
}
