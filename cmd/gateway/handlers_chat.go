package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/chat"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/MAMUER/project/internal/sanitize"
)

// @Summary      Chat
// @Description  Rule-based FAQ/чат
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/v1/chat [post]
func (g *gateway) chatHandler(w http.ResponseWriter, r *http.Request) {
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

	answer, menu := chat.RespondWithProfile(req.Message, g.loadDietProfile(r.Context(), userID))

	resp := map[string]interface{}{
		"status":    "ok",
		"message":   req.Message,
		"answer":    answer,
		"user_id":   userID,
		"timestamp": time.Now().Unix(),
	}
	// When the user asked for a menu, attach the generated day plan so the
	// frontend renders a structured "План питания" block (see web useAI.js).
	if menu != nil {
		resp["diet"] = menu
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		g.log.Error("Failed to encode chat response", zap.Error(err))
	}
}

// loadDietProfile собирает профиль питания из user_profiles и survey_data.
// Аллергии и ограничения из опроса влияют на меню, а антропометрия — на
// суточную калорийность.
func (g *gateway) loadDietProfile(ctx context.Context, userID string) chat.DietProfile {
	profile := chat.DietProfile{Goal: "balance"}
	if g.db == nil || userID == "" {
		return profile
	}

	var age sql.NullInt32
	var gender, nutrition, activity sql.NullString
	var height sql.NullInt32
	var weight sql.NullFloat64
	var surveyRaw []byte

	err := g.db.QueryRowContext(ctx, `
		SELECT age, gender, height_cm, weight_kg, nutrition, survey_data
		FROM user_profiles WHERE user_id = $1
	`, userID).Scan(&age, &gender, &height, &weight, &nutrition, &surveyRaw)
	if err != nil {
		g.log.Debug("Не удалось загрузить профиль питания",
			zap.String("user_id", sanitize.LogString(userID)), zap.Error(err))
		return profile
	}

	if age.Valid {
		profile.Age = int(age.Int32)
	}
	if gender.Valid {
		profile.Gender = gender.String
	}
	if height.Valid {
		profile.HeightCm = float64(height.Int32)
	}
	if weight.Valid {
		profile.WeightKg = weight.Float64
	}
	if nutrition.Valid {
		profile.Diet = nutrition.String
	}

	if len(surveyRaw) > 0 {
		var survey map[string]interface{}
		if err := json.Unmarshal(surveyRaw, &survey); err == nil {
			profile.Allergies = stringSlice(survey["allergies"])
			profile.Restrictions = stringSlice(survey["restrictions"])
			profile.Diet = firstNonEmpty(profile.Diet, stringValue(survey["diet"]))
			profile.Activity = firstNonEmpty(stringValue(survey["activity"]), activity.String)
			profile.Goal = firstNonEmpty(goalKey(survey["goals"]), profile.Goal)
			profile.StressLevel = stressFromSurvey(survey)
		}
	} else if activity.Valid {
		profile.Activity = activity.String
	}

	if isFemaleProfile(profile.Gender) {
		profile.MenstrualPhase = g.latestMenstrualPhase(ctx, userID)
	}

	return profile
}

func stringSlice(v interface{}) []string {
	switch typed := v.(type) {
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return typed
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	default:
		return nil
	}
}

func stringValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// goalKey переводит ответ опроса о целях в ключ шаблона меню.
func goalKey(v interface{}) string {
	for _, item := range stringSlice(v) {
		lower := strings.ToLower(item)
		switch {
		case strings.Contains(lower, "похуд"), strings.Contains(lower, "вес"):
			return "weight_loss"
		case strings.Contains(lower, "мышц"), strings.Contains(lower, "масс"):
			return "muscle_gain"
		}
	}
	return ""
}

// latestMenstrualPhase вычисляет фазу цикла по последней записи.
func (g *gateway) latestMenstrualPhase(ctx context.Context, userID string) string {
	var start string
	err := g.db.QueryRowContext(ctx, `
		SELECT cycle_start_date FROM user_menstrual_cycles
		WHERE user_id = $1
		ORDER BY cycle_start_date DESC
		LIMIT 1
	`, userID).Scan(&start)
	if err != nil || len(start) < 10 {
		return ""
	}
	parsed, err := time.Parse("2006-01-02", start[:10])
	if err != nil {
		return ""
	}
	day := int(time.Since(parsed).Hours()/24) + 1
	switch {
	case day < 1:
		return ""
	case day <= 5:
		return "menstruation"
	case day <= 12:
		return "follicular"
	case day <= 16:
		return "ovulation"
	default:
		return "luteal"
	}
}

func isFemaleProfile(gender string) bool {
	switch strings.ToLower(strings.TrimSpace(gender)) {
	case "female", "f", "жен", "женский":
		return true
	default:
		return false
	}
}

// stressFromSurvey читает уровень стресса; косвенно — цель «снизить стресс».
func stressFromSurvey(survey map[string]interface{}) float64 {
	for _, key := range []string{"stress", "stressLevel", "stress_level"} {
		if v := firstNumber(survey[key]); v > 0 {
			return v
		}
	}
	for _, goal := range stringSlice(survey["goals"]) {
		lower := strings.ToLower(goal)
		if strings.Contains(lower, "стресс") || strings.Contains(lower, "stress") {
			return 6
		}
	}
	return 0
}

func firstNumber(v interface{}) float64 {
	switch typed := v.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}
