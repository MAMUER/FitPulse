package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
)

// ========== Profile Handlers ==========

// @Summary      Get user profile
// @Description  Retrieves the authenticated user's profile information
// @Tags         Profile
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/profile [get]

func (g *gateway) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "getProfile"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
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

	profileResp := map[string]interface{}{
		"status":  "ok",
		"profile": resp,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profileResp); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
		http.Error(w, "encodeResponseError", http.StatusInternalServerError)
		return
	}
}

// @Summary      Update user profile
// @Description  Updates the authenticated user's profile with fitness and health details
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Param        request  body  object  required  "Profile update data"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/profile [put]

func (g *gateway) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "updateProfile"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
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
		http.Error(w, errBadRequest, http.StatusBadRequest)
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
			g.log.Error("User service unavailable during profile update", zap.Error(err))
			http.Error(w, "Сервис пользователей временно недоступен. Попробуйте позже.", http.StatusServiceUnavailable)
			return
		}
		g.log.Error("Profile update failed", zap.Int("http_code", httpCode), zap.String("error", errMsg))
		http.Error(w, errMsg, httpCode)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"}); err != nil {
		g.log.Error(logFailedToEncodeResponse, zap.Error(err))
		http.Error(w, "encodeResponseError", http.StatusInternalServerError)
		return
	}
}

// ========== Security #10: Server-side Role Re-verification ==========
// Role verification is now performed inside user-service RPCs.

// @Summary      Delete user profile
// @Description  Deletes the authenticated user's profile after password confirmation
// @Tags         Profile
// @Accept       json
// @Produce      json
// @Param        request  body  object  required  "Password confirmation for profile deletion"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/v1/profile [delete]

func (g *gateway) deleteProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		g.log.Error("Unauthorized access", zap.String("handler", "deleteProfile"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	req, err := decodeDeleteProfileRequest(r)
	if err != nil {
		g.log.Error("Failed to decode delete profile request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.DeleteProfile(r.Context(), &userpb.DeleteProfileRequest{
		UserId:   userID,
		Password: req.Password,
	})
	if err != nil {
		httpCode, errMsg := grpcToHTTPStatus(err)
		g.log.Error("Failed to delete profile", zap.Error(err))
		http.Error(w, errMsg, httpCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  resp.GetStatus(),
		"message": resp.GetMessage(),
	}); err != nil {
		g.log.Error("Failed to encode delete profile response", zap.Error(err))
	}
}

type deleteProfileRequest struct {
	Password string
}

func decodeDeleteProfileRequest(r *http.Request) (*deleteProfileRequest, error) {
	var req deleteProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, fmt.Errorf("некорректный запрос: %w", err)
	}
	if req.Password == "" {
		return nil, errors.New("password требуется")
	}
	return &req, nil
}
