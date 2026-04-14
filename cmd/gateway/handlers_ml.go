package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	biometricpb "github.com/MAMUER/Project/api/gen/biometric"
	trainingpb "github.com/MAMUER/Project/api/gen/training"
	userpb "github.com/MAMUER/Project/api/gen/user"
	"github.com/MAMUER/Project/internal/middleware"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// mlJobStatusHandler — общая функция проверки статуса ML-задачи
// Устраняет дублирование между classifyStatusHandler и generatePlanStatusHandler
func (g *gateway) mlJobStatusHandler(w http.ResponseWriter, r *http.Request, redisKey string) {
	vars := mux.Vars(r)
	jobID := vars["job_id"]
	if jobID == "" {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	val, err := g.rdb.Get(ctx, fmt.Sprintf("%s%s", redisKey, jobID)).Result()
	if errors.Is(err, redis.Nil) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if encErr := json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id": jobID,
			"status": "processing",
		}); encErr != nil {
			g.log.Error("Failed to encode response", zap.Error(encErr))
		}
		return
	}
	if err != nil {
		g.log.Error("Failed to get job result from Redis", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	safeVal := html.EscapeString(val)
	if _, err := w.Write([]byte(safeVal)); err != nil {
		g.log.Error("Failed to write response", zap.Error(err))
	}
}

func (g *gateway) classifyHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	bioResp, err := g.biometricClient.GetLatest(r.Context(), &biometricpb.GetLatestRequest{
		UserId:     userID,
		MetricType: "heart_rate",
	})
	if err != nil {
		g.log.Warn("Failed to get heart rate", zap.Error(err))
	}

	mlPayload := extractMLPayload(bioResp)

	if g.mlAsync {
		g.handleAsyncClassify(w, r, mlPayload)
		return
	}

	reqBody, _ := json.Marshal(mlPayload)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !isValidServiceURL(g.mlClassifierURL, "http://localhost:", "http://ml-", "http://classifier:") {
		g.log.Error("Invalid ML classifier URL", zap.String("url", g.mlClassifierURL))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		g.mlClassifierURL+"/classify",
		bytes.NewReader(reqBody))
	if err != nil {
		g.log.Error("Failed to create ML classifier request", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		g.log.Error("ML classifier request failed", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			g.log.Error("Failed to close response body", zap.Error(closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		g.log.Error("ML classifier returned error", zap.Int("status", resp.StatusCode))
		http.Error(w, "Ошибка классификации", resp.StatusCode)
		return
	}

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		g.log.Error("Failed to write response", zap.Error(err))
	}
}

func (g *gateway) handleAsyncClassify(w http.ResponseWriter, r *http.Request, mlPayload map[string]interface{}) {
	jobID := uuid.New().String()

	body, err := json.Marshal(map[string]interface{}{
		"job_id":             jobID,
		"physiological_data": mlPayload["physiological_data"],
	})
	if err != nil {
		g.log.Error("Failed to marshal classify job", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err = g.rmqCh.PublishWithContext(ctx, "", "ml.classify", false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})
	if err != nil {
		g.log.Error("Failed to publish classify job to RabbitMQ", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id": jobID,
		"status": "pending",
	}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

// classifyStatusHandler — делегирует общую логику mlJobStatusHandler
func (g *gateway) classifyStatusHandler(w http.ResponseWriter, r *http.Request) {
	g.mlJobStatusHandler(w, r, "ml:result:")
}

func (g *gateway) generateMLPlanHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	var req struct {
		ClassName     string `json:"training_class"`
		DurationWeeks int    `json:"duration_weeks"`
		AvailableDays []int  `json:"available_days"`
		Preferences   struct {
			MaxDuration        int      `json:"max_duration"`
			AvailableEquipment []string `json:"available_equipment"`
			PreferredTime      string   `json:"preferred_time"`
		} `json:"preferences"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode generate ML plan request", zap.Error(err))
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	profile, err := g.userClient.GetProfile(r.Context(), &userpb.GetProfileRequest{UserId: userID})
	if err != nil {
		g.log.Error("Failed to get profile", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	userProfile := map[string]interface{}{}
	if profile.Age > 0 {
		userProfile["age"] = profile.Age
	}
	if profile.Gender != "" {
		userProfile["gender"] = profile.Gender
	}
	if profile.WeightKg > 0 {
		userProfile["weight"] = profile.WeightKg
	}
	if profile.HeightCm > 0 {
		userProfile["height"] = profile.HeightCm
	}
	if profile.FitnessLevel != "" {
		userProfile["fitness_level"] = profile.FitnessLevel
	}
	if len(profile.Contraindications) > 0 {
		userProfile["health_conditions"] = profile.Contraindications
	}
	if len(profile.Goals) > 0 {
		userProfile["goals"] = profile.Goals
	}
	if profile.SleepHours > 0 {
		userProfile["sleep_hours"] = profile.SleepHours
	}
	if profile.Nutrition != "" {
		userProfile["nutrition"] = profile.Nutrition
	}

	if g.mlAsync {
		g.handleAsyncGeneratePlan(w, r, req.ClassName, userProfile, req.Preferences)
		return
	}

	genReq := map[string]interface{}{
		"training_class": req.ClassName,
		"user_profile":   userProfile,
		"preferences": map[string]interface{}{
			"max_duration":        req.Preferences.MaxDuration,
			"available_equipment": req.Preferences.AvailableEquipment,
			"preferred_time":      req.Preferences.PreferredTime,
		},
	}

	reqBody, err := json.Marshal(genReq)
	if err != nil {
		g.log.Error("Failed to marshal generator request", zap.Error(err))
		http.Error(w, "Ошибка формирования запроса", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	mlReq, err := http.NewRequestWithContext(r.Context(), "POST", g.mlGeneratorURL+"/generate-plan", bytes.NewBuffer(reqBody))
	if err != nil {
		g.log.Error("Failed to create request", zap.Error(err))
		http.Error(w, "Ошибка формирования запроса", http.StatusInternalServerError)
		return
	}
	mlReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(mlReq)
	if err != nil {
		g.log.Error("Failed to call ML generator", zap.Error(err))
		http.Error(w, "Ошибка обращения к ML-генератору", http.StatusInternalServerError)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			g.log.Error("Failed to close response body", zap.Error(closeErr))
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		g.log.Error("Failed to read response body", zap.Error(err))
		http.Error(w, "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}

	var mlResp map[string]interface{}
	if err := json.Unmarshal(body, &mlResp); err == nil {
		availableDays := make([]int32, len(req.AvailableDays))
		for i, d := range req.AvailableDays {
			availableDays[i] = safeIntToInt32(d)
		}

		_, saveErr := g.trainingClient.GeneratePlan(r.Context(), &trainingpb.GeneratePlanRequest{
			UserId:              userID,
			ClassificationClass: req.ClassName,
			Confidence:          0.85,
			DurationWeeks:       safeIntToInt32(req.DurationWeeks),
			AvailableDays:       availableDays,
		})
		if saveErr != nil {
			g.log.Warn("Failed to save ML plan to training service (non-critical)", zap.Error(saveErr))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(body); err != nil {
		g.log.Error("Failed to write response body", zap.Error(err))
	}
}

func (g *gateway) handleAsyncGeneratePlan(w http.ResponseWriter, r *http.Request, trainingClass string, userProfile map[string]interface{}, prefs struct {
	MaxDuration        int      `json:"max_duration"`
	AvailableEquipment []string `json:"available_equipment"`
	PreferredTime      string   `json:"preferred_time"`
}) {
	jobID := uuid.New().String()

	body, err := json.Marshal(map[string]interface{}{
		"job_id":         jobID,
		"training_class": trainingClass,
		"user_profile":   userProfile,
		"preferences": map[string]interface{}{
			"max_duration":        prefs.MaxDuration,
			"available_equipment": prefs.AvailableEquipment,
			"preferred_time":      prefs.PreferredTime,
		},
	})
	if err != nil {
		g.log.Error("Failed to marshal generate-plan job", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err = g.rmqCh.PublishWithContext(ctx, "", "ml.generate", false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})
	if err != nil {
		g.log.Error("Failed to publish generate-plan job to RabbitMQ", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id": jobID,
		"status": "pending",
	}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

// generatePlanStatusHandler — делегирует общую логику mlJobStatusHandler
func (g *gateway) generatePlanStatusHandler(w http.ResponseWriter, r *http.Request) {
	g.mlJobStatusHandler(w, r, "ml:generate:")
}
