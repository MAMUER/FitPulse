package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	userpb "github.com/MAMUER/Project/api/gen/user"
	"github.com/MAMUER/Project/internal/auth"
	"github.com/MAMUER/Project/internal/middleware"
	"go.uber.org/zap"
)

// ========== Profile Handlers ==========

func (g *gateway) profileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	resp, err := g.userClient.GetProfile(r.Context(), &userpb.GetProfileRequest{
		UserId: userID,
	})
	if err != nil {
		g.log.Error("Failed to get profile", zap.Error(err), zap.String("user_id", userID))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	// Требование #11: HMAC-SHA256 подпись критического ответа
	profileResp := map[string]interface{}{
		"status":  "ok",
		"profile": resp,
	}
	if signature, err := auth.SignResponse(profileResp, g.jwtSecret); err == nil {
		w.Header().Set("X-Response-Signature", signature)
	}

	if err := json.NewEncoder(w).Encode(profileResp); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

func (g *gateway) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	var req struct {
		FullName          string   `json:"full_name"`
		Age               int32    `json:"age"`
		Gender            string   `json:"gender"`
		HeightCm          int32    `json:"height_cm"`
		WeightKg          float64  `json:"weight_kg"`
		FitnessLevel      string   `json:"fitness_level"`
		Goals             []string `json:"goals"`
		Contraindications []string `json:"contraindications"`
		Nutrition         string   `json:"nutrition"`
		SleepHours        float32  `json:"sleep_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode update profile request", zap.Error(err))
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	_, err := g.userClient.UpdateProfile(r.Context(), &userpb.UpdateProfileRequest{
		UserId:            userID,
		FullName:          ptrString(req.FullName),
		Age:               ptrInt32(req.Age),
		Gender:            ptrString(req.Gender),
		HeightCm:          ptrInt32(req.HeightCm),
		WeightKg:          ptrFloat64(req.WeightKg),
		FitnessLevel:      ptrString(req.FitnessLevel),
		Goals:             req.Goals,
		Contraindications: req.Contraindications,
		Nutrition:         ptrString(req.Nutrition),
		SleepHours:        ptrFloat32(req.SleepHours),
	})
	if err != nil {
		g.log.Error("Failed to update profile", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		if httpCode == http.StatusInternalServerError {
			http.Error(w, "Сервис пользователей временно недоступен. Попробуйте позже.", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, errMsg, httpCode)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

// ========== Security #10: Server-side Role Re-verification ==========

// verifyUserRole re-queries the user's role from the database to prevent privilege escalation.
func (g *gateway) verifyUserRole(ctx context.Context, userID, requiredRole string) bool {
	if g.db == nil {
		g.log.Warn("Database not available for role verification")
		return false
	}
	var actualRole string
	err := g.db.QueryRowContext(ctx, "SELECT role FROM users WHERE id = $1", userID).Scan(&actualRole)
	if err == sql.ErrNoRows {
		g.log.Warn("User not found during role verification", zap.String("user_id", userID))
		return false
	}
	if err != nil {
		g.log.Error("Database error during role verification", zap.Error(err))
		return false
	}
	return actualRole == requiredRole
}

// deleteProfileHandler handles profile deletion with server-side role re-verification.
func (g *gateway) deleteProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}

	if !g.verifyUserRole(r.Context(), userID, "user") {
		g.log.Warn("Role verification failed for profile deletion", zap.String("user_id", userID))
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}

	// Delete profile directly from database
	if g.db == nil {
		g.log.Error("Database not available for profile deletion")
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}
	_, err := g.db.ExecContext(r.Context(), "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		g.log.Error("Failed to delete profile", zap.Error(err), zap.String("user_id", userID))
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}

	// Требование #1: Инвалидация сессии после удаления профиля
	logoutHeaders := middleware.LogoutHeaders()
	for key, values := range logoutHeaders {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "deleted"}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}
