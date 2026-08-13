package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

	trainingpb "github.com/MAMUER/project/api/gen/training"
	"github.com/MAMUER/project/internal/middleware"
)

func TestTraining_GeneratePlanHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/training/plans/generate", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.generatePlanHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTraining_GeneratePlanHandler_InvalidJSON(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/training/plans/generate", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	g.generatePlanHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTraining_GeneratePlanHandler_Success(t *testing.T) {
	g := newTestGateway()
	withTrainingClient(g)

	w := httptest.NewRecorder()
	reqBody := []byte(`{"duration_weeks":4,"available_days":[1,3,5],"class":"endurance_basic"}`)
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/training/plans/generate", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.generatePlanHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "status")
}

func TestTraining_GeneratePlanHandler_DefaultClass(t *testing.T) {
	g := newTestGateway()
	withTrainingClient(g)

	w := httptest.NewRecorder()
	reqBody := []byte(`{"duration_weeks":4,"available_days":[1,3,5]}`)
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/training/plans/generate", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.generatePlanHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "endurance_basic")
}

func TestTraining_GeneratePlanHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	withTrainingClient(g)
	trainingClient := g.trainingClient.(*mockTrainingClient)
	trainingClient.generateErr = grpcError(codes.Internal, "internal error")

	w := httptest.NewRecorder()
	reqBody := []byte(`{"duration_weeks":4,"available_days":[1,3,5]}`)
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/training/plans/generate", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.generatePlanHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestTraining_ListPlansHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/training/plans", nil)

	g.listPlansHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTraining_ListPlansHandler_Success(t *testing.T) {
	g := newTestGateway()
	withTrainingClient(g)
	trainingClient := g.trainingClient.(*mockTrainingClient)
	trainingClient.listResp = &trainingpb.ListPlansResponse{
		Plans: []*trainingpb.TrainingPlan{
			{
				Id:     "plan-1",
				UserId: "user-123",
				Status: "active",
				PlanData: &structpb.Struct{
					Fields: map[string]*structpb.Value{},
				},
			},
		},
		Total: 1,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/api/v1/training/plans?page=1&page_size=10", nil)

	g.listPlansHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "plans")
}

func TestTraining_CompleteWorkoutHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/training/workouts/complete", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.completeWorkoutHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTraining_CompleteWorkoutHandler_Success(t *testing.T) {
	g := newTestGateway()
	withTrainingClient(g)

	w := httptest.NewRecorder()
	reqBody := []byte(`{"plan_id":"plan-1","workout_id":"workout-1","rating":5,"feedback":"Great!"}`)
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/training/workouts/complete", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.completeWorkoutHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestTraining_GetProgressHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/training/progress", nil)

	g.getProgressHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTraining_GetProgressHandler_Success(t *testing.T) {
	g := newTestGateway()
	withTrainingClient(g)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/api/v1/training/progress", nil)

	g.getProgressHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestTraining_GetPlanHandler_MissingPlanID(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/training/plans/", nil)

	g.getPlanHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTraining_GetPlanHandler_Success(t *testing.T) {
	g := newTestGateway()
	withTrainingClient(g)
	trainingClient := g.trainingClient.(*mockTrainingClient)
	trainingClient.plans["plan-1"] = &trainingpb.TrainingPlan{
		Id:     "plan-1",
		UserId: "user-123",
		Status: "active",
		PlanData: &structpb.Struct{
			Fields: map[string]*structpb.Value{},
		},
	}

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plan_id", "plan-1")
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user-123")
	req := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/training/plans/plan-1", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	g.getPlanHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "plan_id")
}

func TestTraining_GetPlanHandler_NotFound(t *testing.T) {
	g := newTestGateway()
	withTrainingClient(g)

	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plan_id", "nonexistent")
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "user-123")
	req := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/training/plans/nonexistent", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	g.getPlanHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTraining_GetAchievementsHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/training/achievements", nil)

	g.getAchievementsHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTraining_GetAchievementsHandler_Success(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "GET", "/api/v1/training/achievements", nil)

	g.getAchievementsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTraining_UnmarshalPlanData_Nil(t *testing.T) {
	result, err := unmarshalPlanData(nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTraining_UnmarshalPlanData_ValidStruct(t *testing.T) {
	planData := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"duration_weeks": {Kind: &structpb.Value_NumberValue{NumberValue: 4}},
		},
	}

	result, err := unmarshalPlanData(planData)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(4), result["duration_weeks"])
}
