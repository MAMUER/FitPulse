package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	trainingpb "github.com/MAMUER/Project/api/gen/training"
	"github.com/MAMUER/Project/internal/middleware"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func (g *gateway) generatePlanHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	var req struct {
		DurationWeeks int     `json:"duration_weeks"`
		AvailableDays []int   `json:"available_days"`
		Class         string  `json:"class"`
		Confidence    float64 `json:"confidence"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode generate plan request", zap.Error(err))
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	class := req.Class
	if class == "" {
		class = "endurance_e1e2"
	}

	availableDays := make([]int32, len(req.AvailableDays))
	for i, d := range req.AvailableDays {
		availableDays[i] = safeIntToInt32(d)
	}

	resp, err := g.trainingClient.GeneratePlan(r.Context(), &trainingpb.GeneratePlanRequest{
		UserId:              userID,
		ClassificationClass: class,
		Confidence:          req.Confidence,
		DurationWeeks:       safeIntToInt32(req.DurationWeeks),
		AvailableDays:       availableDays,
	})
	if err != nil {
		g.log.Error("Failed to generate plan", zap.Error(err), zap.String("user_id", userID), zap.String("class", class))
		httpCode, errMsg := grpcToHTTPStatus(err)
		g.log.Info("gRPC error details", zap.Int("httpCode", httpCode), zap.String("errMsg", errMsg), zap.String("grpc_code", err.Error()))
		if httpCode == http.StatusInternalServerError {
			http.Error(w, "Сервис тренировок временно недоступен. Попробуйте позже.", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, errMsg, httpCode)
		return
	}

	planDataJSON, err := json.Marshal(resp.PlanData)
	if err != nil {
		g.log.Error("Failed to marshal plan data", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
	planData := make(map[string]interface{})
	if len(planDataJSON) > 0 && string(planDataJSON) != "null" {
		if err := json.Unmarshal(planDataJSON, &planData); err != nil {
			g.log.Error("Failed to unmarshal plan data", zap.Error(err))
			http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
			return
		}
	}

	planData["duration_weeks"] = req.DurationWeeks
	planData["training_goal"] = class

	response := map[string]interface{}{
		"status":        "ok",
		"plan_id":       resp.PlanId,
		"plan_data":     planData,
		"training_type": class,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) getPlansHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	pageSize := 10
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 {
			pageSize = val
		}
	}

	resp, err := g.trainingClient.ListPlans(r.Context(), &trainingpb.ListPlansRequest{
		UserId:   userID,
		Page:     safeIntToInt32(page),
		PageSize: safeIntToInt32(pageSize),
	})
	if err != nil {
		g.log.Error("Failed to get plans", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	// Convert protobuf plans to JSON
	plans := make([]map[string]interface{}, len(resp.Plans))
	for i, plan := range resp.Plans {
		planDataJSON, err := json.Marshal(plan.PlanData)
		if err != nil {
			g.log.Error("Failed to marshal plan data", zap.Error(err))
			http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
			return
		}
		var planData map[string]interface{}
		if err := json.Unmarshal(planDataJSON, &planData); err != nil {
			g.log.Error("Failed to unmarshal plan data", zap.Error(err))
			http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
			return
		}

		// Extract common fields for frontend compatibility
		durationWeeks, _ := planData["duration_weeks"].(float64)
		trainingGoal, _ := planData["training_goal"].(string)

		plans[i] = map[string]interface{}{
			"plan_id":        plan.Id,
			"user_id":        plan.UserId,
			"plan_data":      planData,
			"status":         plan.Status,
			"duration_weeks": durationWeeks,
			"training_goal":  trainingGoal,
			// Also expose start_date/end_date as strings for frontend
			"start_date": plan.StartDate.AsTime().Format("2006-01-02"),
			"end_date":   plan.EndDate.AsTime().Format("2006-01-02"),
		}
	}

	response := map[string]interface{}{
		"status": "ok",
		"plans":  plans,
		"total":  resp.Total,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
	}
}

func (g *gateway) getPlanHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := r.Context().Value(middleware.UserIDKey).(string); !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	planID := vars["plan_id"]
	if planID == "" {
		http.Error(w, "ID плана не указан", http.StatusBadRequest)
		return
	}

	resp, err := g.trainingClient.GetPlan(r.Context(), &trainingpb.GetPlanRequest{
		PlanId: planID,
	})
	if err != nil {
		g.log.Error("Failed to get plan", zap.Error(err), zap.String("plan_id", planID))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	planDataJSON, err := json.Marshal(resp.PlanData)
	if err != nil {
		g.log.Error("Failed to marshal plan data", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
	var planData map[string]interface{}
	if err := json.Unmarshal(planDataJSON, &planData); err != nil {
		g.log.Error("Failed to unmarshal plan data", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status": "ok",
		"plan": map[string]interface{}{
			"plan_id":      resp.Id,
			"user_id":      resp.UserId,
			"plan_data":    planData,
			"status":       resp.Status,
			"start_date":   resp.StartDate.AsTime().Format("2006-01-02"),
			"end_date":     resp.EndDate.AsTime().Format("2006-01-02"),
			"generated_at": resp.GeneratedAt.AsTime().Format("2006-01-02"),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
	}
}

func (g *gateway) completeWorkoutHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	var req struct {
		PlanId    string `json:"plan_id"`
		WorkoutId string `json:"workout_id"`
		Rating    int32  `json:"rating"`
		Feedback  string `json:"feedback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode complete workout request", zap.Error(err))
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	_, err := g.trainingClient.CompleteWorkout(r.Context(), &trainingpb.CompleteWorkoutRequest{
		UserId:    userID,
		PlanId:    req.PlanId,
		WorkoutId: req.WorkoutId,
		Rating:    req.Rating,
		Feedback:  req.Feedback,
	})
	if err != nil {
		g.log.Error("Failed to complete workout", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) getProgressHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	_, err := g.trainingClient.GetProgress(r.Context(), &trainingpb.GetProgressRequest{
		UserId: userID,
	})
	if err != nil {
		g.log.Error("Failed to get progress", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}
