package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	trainingpb "github.com/MAMUER/project/api/gen/training"
	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
)

const dateFormat = "dateFormat"

type completeWorkoutRequest struct {
	PlanID    string `json:"plan_id"`
	WorkoutID string `json:"workout_id"`
	Rating    int32  `json:"rating"`
	Feedback  string `json:"feedback"`
}

// @Summary      Generate training plan
// @Description  Generates a new personalized training plan based on user preferences and biometric classification
// @Tags         Training
// @Accept       json
// @Produce      json
// @Param        request  body  object  required  "Training plan generation parameters"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/training/generate [post]

func (g *gateway) generatePlanHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "generatePlan"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
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
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}

	class := req.Class
	if class == "" {
		class = "endurance_basic"
	}

	availableDays := make([]int32, len(req.AvailableDays))
	for i, d := range req.AvailableDays {
		availableDays[i] = safeIntToInt32(d)
	}

	client, err := g.getTrainingClient()
	if err != nil {
		g.log.Error("Failed to get training client", zap.Error(err))
		http.Error(w, serviceTrainingUnavailable, http.StatusServiceUnavailable)
		return
	}

	resp, err := client.GeneratePlan(r.Context(), &trainingpb.GeneratePlanRequest{
		UserId:              userID,
		ClassificationClass: class,
		Confidence:          req.Confidence,
		DurationWeeks:       safeIntToInt32(req.DurationWeeks),
		AvailableDays:       availableDays,
	})
	sanitize := func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(s, "\n", ""), "\r", "")
	}
	if err != nil {
		g.log.Error("Failed to generate plan",
			zap.Error(err),
			zap.String("user_id", sanitize(userID)),
			zap.String("class", sanitize(class)),
		)
		httpCode, errMsg := grpcToHTTPStatus(err)
		g.log.Info("gRPC error details",
			zap.Int("httpCode", httpCode),
			zap.String("errMsg", sanitize(errMsg)),
			zap.String("grpc_code", sanitize(err.Error())),
		)
		if httpCode == http.StatusInternalServerError {
			g.log.Error("Training service unavailable during plan generation", zap.Error(err))
			http.Error(w, "Сервис тренировок временно недоступен. Попробуйте позже.", http.StatusServiceUnavailable)
			return
		}
		g.log.Error("Plan generation failed", zap.Int("http_code", httpCode), zap.String("error", errMsg))
		http.Error(w, errMsg, httpCode)
		return
	}

	planDataJSON, err := json.Marshal(resp.PlanData)
	if err != nil {
		g.log.Error("Failed to marshal plan data", zap.Error(err))
		http.Error(w, "encodeResponseError", http.StatusInternalServerError)
		return
	}
	planData := make(map[string]interface{})
	if len(planDataJSON) > 0 && string(planDataJSON) != "null" {
		if err := json.Unmarshal(planDataJSON, &planData); err != nil {
			g.log.Error("Failed to unmarshal plan data", zap.Error(err))
			http.Error(w, "encodeResponseError", http.StatusInternalServerError)
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

	w.Header().Set(headerContentType, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
		http.Error(w, "encodeResponseError", http.StatusInternalServerError)
		return
	}
}

// @Summary      List training plans
// @Description  Retrieves a paginated list of training plans for the authenticated user
// @Tags         Training
// @Produce      json
// @Param        page      query  int  false  "Page number (default 1)"
// @Param        page_size query  int  false  "Items per page (default 20)"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/training/plans [get]

func (g *gateway) listPlansHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "listPlans"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	page, pageSize := parsePagination(r)

	client, err := g.getTrainingClient()
	if err != nil {
		g.log.Error("Failed to get training client", zap.Error(err))
		http.Error(w, serviceTrainingUnavailable, http.StatusServiceUnavailable)
		return
	}

	resp, err := client.ListPlans(r.Context(), &trainingpb.ListPlansRequest{
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

	plans := make([]map[string]interface{}, len(resp.Plans))
	for i, plan := range resp.Plans {
		planData, err := unmarshalPlanData(plan.PlanData)
		if err != nil {
			g.log.Error("Failed to unmarshal plan data", zap.Error(err))
			http.Error(w, encodeResponseError, http.StatusInternalServerError)
			return
		}

		durationWeeks, _ := planData["duration_weeks"].(float64)
		trainingGoal, _ := planData["training_goal"].(string)

		plans[i] = map[string]interface{}{
			"plan_id":        plan.Id,
			"user_id":        plan.UserId,
			"plan_data":      planData,
			"status":         plan.Status,
			"duration_weeks": durationWeeks,
			"training_goal":  trainingGoal,
			"start_date":     plan.StartDate.AsTime().Format(dateFormat),
			"end_date":       plan.EndDate.AsTime().Format(dateFormat),
		}
	}

	response := map[string]interface{}{
		"status": "ok",
		"plans":  plans,
		"total":  resp.Total,
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}

func unmarshalPlanData(planDataProto *structpb.Struct) (map[string]interface{}, error) {
	if planDataProto == nil {
		return map[string]interface{}{}, nil
	}
	planDataJSON, err := json.Marshal(planDataProto)
	if err != nil {
		return nil, err
	}
	var planData map[string]interface{}
	if err := json.Unmarshal(planDataJSON, &planData); err != nil {
		return nil, err
	}
	return planData, nil
}

// @Summary      Complete workout
// @Description  Marks a workout as completed with optional rating and feedback
// @Tags         Training
// @Accept       json
// @Produce      json
// @Param        request  body  object  required  "Workout completion data"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/training/complete [post]

func (g *gateway) completeWorkoutHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "completeWorkout"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	var req completeWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode complete workout request", zap.Error(err))
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}

	client, err := g.getTrainingClient()
	if err != nil {
		g.log.Error("Failed to get training client", zap.Error(err))
		http.Error(w, serviceTrainingUnavailable, http.StatusServiceUnavailable)
		return
	}

	_, err = client.CompleteWorkout(r.Context(), &trainingpb.CompleteWorkoutRequest{
		UserId:    userID,
		PlanId:    req.PlanID,
		WorkoutId: req.WorkoutID,
		Rating:    req.Rating,
		Feedback:  req.Feedback,
	})
	if err != nil {
		g.log.Error("Failed to complete workout", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
		http.Error(w, "encodeResponseError", http.StatusInternalServerError)
		return
	}
}

// @Summary      Get training progress
// @Description  Retrieves training progress for the authenticated user
// @Tags         Training
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/training/progress [get]

func (g *gateway) getProgressHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "getProgress"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	client, err := g.getTrainingClient()
	if err != nil {
		g.log.Error("Failed to get training client", zap.Error(err))
		http.Error(w, serviceTrainingUnavailable, http.StatusServiceUnavailable)
		return
	}

	_, err = client.GetProgress(r.Context(), &trainingpb.GetProgressRequest{
		UserId: userID,
	})
	if err != nil {
		g.log.Error("Failed to get progress", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
		http.Error(w, "encodeResponseError", http.StatusInternalServerError)
		return
	}
}

// @Summary      Get training plan
// @Description  Retrieves a specific training plan by ID
// @Tags         Training
// @Produce      json
// @Param        plan_id  path  string  true  "Training plan ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/training/plans/{plan_id} [get]

func (g *gateway) getPlanHandler(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "plan_id")
	if planID == "" {
		g.log.Error("Missing plan_id in request")
		http.Error(w, "plan_id required", http.StatusBadRequest)
		return
	}

	client, err := g.getTrainingClient()
	if err != nil {
		g.log.Error("Failed to get training client", zap.Error(err))
		http.Error(w, serviceTrainingUnavailable, http.StatusServiceUnavailable)
		return
	}

	resp, err := client.GetPlan(r.Context(), &trainingpb.GetPlanRequest{
		PlanId: planID,
	})
	if err != nil {
		g.log.Error("Failed to get plan", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	planDataMap := resp.GetPlanData().AsMap()
	if planDataMap == nil {
		planDataMap = map[string]interface{}{}
	}
	planDataMap["plan_id"] = resp.GetId()
	planDataMap["user_id"] = resp.GetUserId()
	planDataMap["status"] = resp.GetStatus()
	if resp.GetStartDate() != nil {
		planDataMap["start_date"] = resp.GetStartDate().AsTime().Format(dateFormat)
	}
	if resp.GetEndDate() != nil {
		planDataMap["end_date"] = resp.GetEndDate().AsTime().Format(dateFormat)
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"plan_id":   resp.GetId(),
		"plan_data": planDataMap,
	}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}

// @Summary      Get user achievements
// @Description  Retrieves achievements for the authenticated user
// @Tags         Training
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/achievements [get]

func (g *gateway) getAchievementsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "getAchievements"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	resp, err := g.userClient.GetAchievements(r.Context(), &userpb.GetAchievementsRequest{
		UserId: userID,
	})
	if err != nil {
		g.log.Error("Failed to get achievements", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
	}
}
