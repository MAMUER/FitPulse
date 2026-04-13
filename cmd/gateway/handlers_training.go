package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	trainingpb "github.com/MAMUER/Project/api/gen/training"
	"github.com/MAMUER/Project/internal/middleware"
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

	_, err := g.trainingClient.GeneratePlan(r.Context(), &trainingpb.GeneratePlanRequest{
		UserId:              userID,
		ClassificationClass: class,
		Confidence:          req.Confidence,
		DurationWeeks:       safeIntToInt32(req.DurationWeeks),
		AvailableDays:       availableDays,
	})
	if err != nil {
		g.log.Error("Failed to generate plan", zap.Error(err), zap.String("user_id", userID))
		httpCode, errMsg := grpcToHTTPStatus(err)
		// Если сервис недоступен — даём понятное сообщение
		if httpCode == http.StatusInternalServerError {
			http.Error(w, "Сервис тренировок временно недоступен. Попробуйте позже.", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, errMsg, httpCode)
		return
	}

	planResp := map[string]interface{}{"status": "ok"}
	if err := json.NewEncoder(w).Encode(planResp); err != nil {
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

	_, err := g.trainingClient.ListPlans(r.Context(), &trainingpb.ListPlansRequest{
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

	plansResp := map[string]interface{}{"status": "ok"}
	if err := json.NewEncoder(w).Encode(plansResp); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
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
