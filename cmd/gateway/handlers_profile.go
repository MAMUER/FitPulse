package main

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
)

// ========== Profile Handlers ==========

const argon2idParams = "m=65536,t=3,p=1"

const argon2KeyLen = 32

func verifyPasswordArgon2id(stored, password string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != argon2idParams {
		return false
	}
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false
	}
	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(params[0], "m=%d", &memory); err != nil {
		return false
	}
	if _, err := fmt.Sscanf(params[1], "t=%d", &iterations); err != nil {
		return false
	}
	if _, err := fmt.Sscanf(params[2], "p=%d", &parallelism); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	computed := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, argon2KeyLen)
	return subtle.ConstantTimeCompare(hash, computed) == 1
}

func (g *gateway) getProfileHandler(w http.ResponseWriter, r *http.Request) {
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

	// Требование #11: ответ кодируется как application/json
	profileResp := map[string]interface{}{
		"status":  "ok",
		"profile": resp,
	}
	w.Header().Set("Content-Type", "application/json")
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

func (g *gateway) deleteProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "password требуется", http.StatusBadRequest)
		return
	}

	if g.db == nil {
		http.Error(w, "Сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	tx, err := g.db.BeginTx(r.Context(), nil)
	if err != nil {
		g.log.Error("Failed to begin delete profile transaction", zap.Error(err))
		http.Error(w, "Ошибка удаления профиля", http.StatusInternalServerError)
		return
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			g.log.Warn("Failed to rollback delete profile transaction", zap.Error(rollbackErr))
		}
	}()

	var passwordHash string
	if queryErr := tx.QueryRowContext(r.Context(), "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&passwordHash); queryErr != nil {
		if queryErr == sql.ErrNoRows {
			http.Error(w, "Не найдено", http.StatusNotFound)
			return
		}
		g.log.Error("Failed to load user for deletion", zap.Error(queryErr))
		http.Error(w, "Ошибка удаления профиля", http.StatusInternalServerError)
		return
	}

	if !verifyPasswordArgon2id(passwordHash, req.Password) {
		http.Error(w, "Неверный пароль", http.StatusUnauthorized)
		return
	}

	if err := g.requireCriticalSession(r, userID); err != nil {
		http.Error(w, err.Error(), http.StatusPreconditionRequired)
		return
	}

	result, err := tx.ExecContext(r.Context(), "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		g.log.Error("Failed to delete user", zap.Error(err))
		http.Error(w, "Ошибка удаления профиля", http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}

	if err := tx.Commit(); err != nil {
		g.log.Error("Failed to commit delete profile transaction", zap.Error(err))
		http.Error(w, "Ошибка удаления профиля", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "deleted"}); err != nil {
		g.log.Error("Failed to encode delete profile response", zap.Error(err))
	}
}
