package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
)

func (g *gateway) callClassifier(ctx context.Context, payload []byte) (map[string]interface{}, error) {
	if !isValidServiceURL(g.classifierURL, "http://classifier:", "http://classifier-service:", "http://127.0.0.1:") {
		g.log.Error("Некорректный URL классификатора", zap.String("url", g.classifierURL))
		return nil, errors.New("некорректный URL классификатора")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.classifierURL+"/classify", bytes.NewReader(payload))
	if err != nil {
		g.log.Error("Failed to create classifier request", zap.Error(err))
		return nil, err
	}
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set("X-Correlation-ID", middleware.GetCorrelationID(ctx))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		g.log.Error("Classifier request failed", zap.Error(err))
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			g.log.Error("Failed to close response body", zap.Error(closeErr))
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		g.log.Error("Failed to read classifier response", zap.Error(err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		g.log.Error("Classifier returned error", zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("classifier status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		g.log.Error("Failed to parse classifier response", zap.Error(err))
		return nil, err
	}

	return result, nil
}

// aggregateMLPayload aggregates biometric metrics into a payload for classifier service
func aggregateMLPayload(metrics map[string]*biometricpb.BiometricRecord) map[string]interface{} {
	physiologicalData := make(map[string]interface{})
	for metricType, record := range metrics {
		key := mapMetricTypeToClassifierKey(metricType)
		if record != nil {
			physiologicalData[key] = record.Value
		} else {
			physiologicalData[key] = nil
		}
	}
	return map[string]interface{}{
		"physiological_data": physiologicalData,
	}
}

func mapMetricTypeToClassifierKey(metricType string) string {
	switch metricType {
	case "hrv":
		return "heart_rate_variability"
	case "systolic_pressure":
		return "blood_pressure_systolic"
	case "diastolic_pressure":
		return "blood_pressure_diastolic"
	default:
		return metricType
	}
}

func transformClassifierResponse(result map[string]interface{}) map[string]interface{} {
	predictedClass, _ := result["predicted_class"].(string)
	confidence, _ := result["confidence"].(float64)

	var recommendations []interface{}
	if recs, ok := result["recommendations"].([]interface{}); ok {
		recommendations = recs
	}

	fatigueLevel, motivationScore, recoveryQuality := mapClassToScores(predictedClass)

	return map[string]interface{}{
		"status":           "success",
		"state":            predictedClass,
		"confidence":       confidence,
		"recommendation":   recommendations,
		"fatigue_level":    fatigueLevel,
		"motivation_score": motivationScore,
		"recovery_quality": recoveryQuality,
	}
}

func mapClassToScores(predictedClass string) (float64, float64, float64) {
	switch predictedClass {
	case "recovery":
		return 0.1, 0.8, 0.9
	case "endurance_basic":
		return 0.3, 0.7, 0.7
	case "endurance_threshold":
		return 0.5, 0.6, 0.5
	case "power_hiit":
		return 0.7, 0.5, 0.3
	case "overtraining":
		return 0.9, 0.2, 0.1
	case "illness":
		return 1.0, 0.0, 0.0
	default:
		return 0.5, 0.5, 0.5
	}
}

func (g *gateway) proxyToMLGenerator(ctx context.Context, path string, body []byte) (int, []byte, error) {
	if !isValidServiceURL(g.mlGeneratorURL, "https://ml-", "https://ml-generator:", "https://generator:", "https://127.0.0.1:", "http://ml-", "http://ml-generator:", "http://generator:", "http://127.0.0.1:") { // NOSONAR: S5332
		g.log.Error("Некорректный URL ML генератора", zap.String("url", g.mlGeneratorURL))
		return http.StatusServiceUnavailable, nil, errors.New("mlServiceUnavailable")
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		g.mlGeneratorURL+path,
		bytes.NewReader(body))
	if err != nil {
		g.log.Error("Failed to create ML generator request", zap.Error(err))
		return http.StatusServiceUnavailable, nil, errors.New("mlServiceUnavailable")
	}
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set("X-Correlation-ID", middleware.GetCorrelationID(ctx))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		g.log.Error("ML generator request failed", zap.Error(err))
		return http.StatusServiceUnavailable, nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			g.log.Error("Failed to close response body", zap.Error(closeErr))
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		g.log.Error("Failed to read generator response", zap.Error(err))
		return http.StatusServiceUnavailable, nil, err
	}

	return resp.StatusCode, respBody, nil
}

func (g *gateway) fetchUserProfile(ctx context.Context, userID string) (*userpb.UserProfile, error) {
	if g.userClient == nil {
		return nil, errors.New("user client not initialized")
	}

	resp, err := g.userClient.GetProfile(ctx, &userpb.GetProfileRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (g *gateway) fetchLatestBiometrics(ctx context.Context, userID string) (map[string]*biometricpb.BiometricRecord, error) {
	client, err := g.getBiometricClient()
	if err != nil {
		return nil, err
	}

	metricTypes := []string{"heart_rate", "hrv", "spo2", "temperature", "systolic_pressure", "diastolic_pressure", "sleep_hours"}
	metrics := make(map[string]*biometricpb.BiometricRecord)

	for _, metricType := range metricTypes {
		bioResp, err := client.GetLatest(ctx, &biometricpb.GetLatestRequest{
			UserId:     userID,
			MetricType: metricType,
		})
		if err != nil {
			g.log.Debug("Failed to get metric", zap.String("metric", metricType), zap.Error(err))
		} else {
			metrics[metricType] = bioResp
		}
	}

	return metrics, nil
}

func (g *gateway) fetchTrainingStats(ctx context.Context, userID string) (*userpb.TrainingStats, error) {
	if g.userClient == nil {
		return nil, errors.New("user client not initialized")
	}

	resp, err := g.userClient.GetTrainingStats(ctx, &userpb.GetTrainingStatsRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, err
	}

	return resp.Stats, nil
}

func buildPlanGenerationPayload(profile *userpb.UserProfile, classification map[string]interface{}, biometrics map[string]*biometricpb.BiometricRecord, stats *userpb.TrainingStats) map[string]interface{} {
	predictedClass, _ := classification["predicted_class"].(string)
	confidence, _ := classification["confidence"].(float64)

	userProfile := map[string]interface{}{
		"age":               profile.Age,
		"gender":            profile.Gender,
		"fitness_level":     profile.FitnessLevel,
		"goals":             profile.Goals,
		"contraindications": profile.Contraindications,
	}
	if profile.WeightKg != 0 {
		userProfile["weight"] = profile.WeightKg
	}
	if profile.HeightCm != 0 {
		userProfile["height"] = profile.HeightCm
	}

	healthStatus := map[string]interface{}{
		"predicted_class":         predictedClass,
		"confidence":              confidence,
		"active_conditions_count": 0,
		"menstrual_phase":         "unknown",
	}

	if hr, ok := biometrics["heart_rate"]; ok && hr != nil {
		healthStatus["heart_rate"] = hr.Value
	}
	if hrv, ok := biometrics["hrv"]; ok && hrv != nil {
		healthStatus["hrv"] = hrv.Value
	}
	if spo2, ok := biometrics["spo2"]; ok && spo2 != nil {
		healthStatus["spo2"] = spo2.Value
	}
	if temp, ok := biometrics["temperature"]; ok && temp != nil {
		healthStatus["temperature"] = temp.Value
	}
	if sleep, ok := biometrics["sleep_hours"]; ok && sleep != nil {
		healthStatus["sleep_hours"] = sleep.Value
	}

	trainingHistory := map[string]interface{}{
		"completed_workouts_count": 0,
		"avg_intensity":            0.5,
		"last_workout_date":        nil,
	}
	if stats != nil {
		trainingHistory["completed_workouts_count"] = stats.CompletedWorkouts
		if stats.AverageDurationMinutes > 0 {
			trainingHistory["avg_intensity"] = 0.5
		}
	}

	preferences := map[string]interface{}{
		"time":           "morning",
		"equipment":      []string{},
		"available_days": []string{"mon", "wed", "fri"},
	}

	return map[string]interface{}{
		"training_class":   predictedClass,
		"user_profile":     userProfile,
		"health_status":    healthStatus,
		"training_history": trainingHistory,
		"preferences":      preferences,
		"constraints": map[string]interface{}{
			"duration_weeks":        4,
			"max_sessions_per_week": 4,
		},
	}
}

func buildDietGenerationPayload(profile *userpb.UserProfile, classification map[string]interface{}) map[string]interface{} {
	predictedClass, _ := classification["predicted_class"].(string)

	goals := []string{}
	if profile.Goals != nil {
		goals = profile.Goals
	}

	trainingGoal := "general_fitness"
	if predictedClass == "recovery" {
		trainingGoal = "recovery"
	} else if predictedClass == "overtraining" {
		trainingGoal = "recovery"
	} else if predictedClass == "illness" {
		trainingGoal = "general_fitness"
	} else if len(goals) > 0 {
		trainingGoal = goals[0]
	}

	return map[string]interface{}{
		"user_id":           profile.UserId,
		"age":               profile.Age,
		"gender":            profile.Gender,
		"weight_kg":         70.0,
		"height_cm":         170.0,
		"fitness_level":     profile.FitnessLevel,
		"goals":             goals,
		"diet_type":         "balanced",
		"meals_count":       4,
		"allergies":         []string{},
		"contraindications": profile.Contraindications,
		"training_goal":     trainingGoal,
	}
}

func (g *gateway) mlChatHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "message обязателен", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	ctx := r.Context()

	profile, err := g.fetchUserProfile(ctx, userID)
	if err != nil {
		g.log.Error("Failed to fetch user profile", zap.Error(err))
		http.Error(w, "Сервис пользователей временно недоступен", http.StatusServiceUnavailable)
		return
	}

	metrics, err := g.fetchLatestBiometrics(ctx, userID)
	if err != nil {
		g.log.Error("Failed to fetch biometrics", zap.Error(err))
		http.Error(w, "Сервис биометрии временно недоступен", http.StatusServiceUnavailable)
		return
	}

	stats, err := g.fetchTrainingStats(ctx, userID)
	if err != nil {
		g.log.Debug("Failed to fetch training stats", zap.Error(err))
	}

	mlPayload := aggregateMLPayload(metrics)
	reqBody, _ := json.Marshal(mlPayload)

	classifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	classification, err := g.callClassifier(classifyCtx, reqBody)
	cancel()
	if err != nil {
		g.log.Error("Failed to classify user state", zap.Error(err))
		http.Error(w, "Сервис классификации временно недоступен", http.StatusServiceUnavailable)
		return
	}

	planPayload := buildPlanGenerationPayload(profile, classification, metrics, stats)
	planBody, _ := json.Marshal(planPayload)

	planCtx, cancelPlan := context.WithTimeout(ctx, 10*time.Second)
	planStatus, planBodyBytes, err := g.proxyToMLGenerator(planCtx, "/generate-plan", planBody)
	cancelPlan()
	if err != nil || planStatus != http.StatusOK {
		g.log.Error("Failed to generate plan", zap.Error(err), zap.Int("status", planStatus))
	}

	dietPayload := buildDietGenerationPayload(profile, classification)
	dietBody, _ := json.Marshal(dietPayload)

	dietCtx, cancelDiet := context.WithTimeout(ctx, 10*time.Second)
	dietStatus, dietBodyBytes, err := g.proxyToMLGenerator(dietCtx, "/generate-diet", dietBody)
	cancelDiet()
	if err != nil || dietStatus != http.StatusOK {
		g.log.Error("Failed to generate diet", zap.Error(err), zap.Int("status", dietStatus))
	}

	var plan map[string]interface{}
	var diet map[string]interface{}
	_ = json.Unmarshal(planBodyBytes, &plan)
	_ = json.Unmarshal(dietBodyBytes, &diet)

	response := map[string]interface{}{
		"status":         "success",
		"message":        req.Message,
		"classification": classification,
		"plan":           plan,
		"diet":           diet,
		"timestamp":      time.Now().Unix(),
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	_ = json.NewEncoder(w).Encode(response)
}
