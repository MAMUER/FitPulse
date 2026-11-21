package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/auth/password"
	"github.com/MAMUER/project/internal/middleware"
)

// changePasswordHandler меняет пароль текущего пользователя.
//
// Раньше кнопка «Изменить пароль» вообще ничего не делала: фронтхаб
// (api.changePassword) только проверял длину и возвращал {oldPass,newPass},
// а бэкенд-эндпоинта не существовало. Здесь — реальная реализация:
// проверка текущего пароля через Argon2id, запись нового хеша и отзыв
// refresh-токенов (чтобы старые сессии не пережили смену пароля).
func (g *gateway) changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		g.log.Error(errUnauthorized, zap.String("handler", "changePassword"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode change-password request", zap.Error(err))
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		http.Error(w, "old_password and new_password are required", http.StatusBadRequest)
		return
	}
	// Минимум 8 символов — синхронно с validator.ValidateRegisterRequest.
	if len(req.NewPassword) < 8 {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	if g.db == nil {
		http.Error(w, errDatabaseUnavailable, http.StatusServiceUnavailable)
		return
	}

	var currentHash string
	err := g.db.QueryRow(
		`SELECT password_hash FROM users WHERE id = $1`, userID,
	).Scan(&currentHash)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}
	if err != nil {
		g.log.Error("Failed to read password hash", zap.Error(err))
		http.Error(w, errDatabaseError, http.StatusInternalServerError)
		return
	}

	if !password.VerifyArgon2id(currentHash, req.OldPassword) {
		g.log.Warn("Change password rejected: wrong current password",
			zap.String("user_id", userID))
		http.Error(w, "invalid current password", http.StatusUnauthorized)
		return
	}

	newHash, err := password.HashArgon2id(req.NewPassword)
	if err != nil {
		g.log.Error("Failed to hash new password", zap.Error(err))
		http.Error(w, errDatabaseError, http.StatusInternalServerError)
		return
	}

	if _, err := g.db.Exec(
		`UPDATE users SET password_hash = $1 WHERE id = $2`, newHash, userID,
	); err != nil {
		g.log.Error("Failed to update password", zap.Error(err))
		http.Error(w, errDatabaseError, http.StatusInternalServerError)
		return
	}

	// Отзываем все refresh-токены: старые сессии должны умереть.
	if _, err := g.db.Exec(
		`DELETE FROM refresh_tokens WHERE user_id = $1`, userID,
	); err != nil {
		g.log.Warn("Failed to revoke refresh tokens after password change", zap.Error(err))
	}

	writeJSON(w, map[string]interface{}{"status": "ok"})
}

// changeEmailHandler меняет email текущего пользователя.
//
// Раньше RPC ChangeEmail был объявлен в user.proto, но не реализован, а в
// gateway не было ни хендлера, ни маршрута — сменить почту было невозможно.
func (g *gateway) changeEmailHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		g.log.Error(errUnauthorized, zap.String("handler", "changeEmail"))
		http.Error(w, msgUnauthorized, http.StatusUnauthorized)
		return
	}

	var req struct {
		NewEmail string `json:"new_email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.log.Error("Failed to decode change-email request", zap.Error(err))
		http.Error(w, errBadRequest, http.StatusBadRequest)
		return
	}

	if req.NewEmail == "" || req.Password == "" {
		http.Error(w, "new_email and password are required", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.ChangeEmail(r.Context(), &userpb.ChangeEmailRequest{
		UserId:   userID,
		NewEmail: req.NewEmail,
		Password: req.Password,
	})
	if err != nil {
		g.log.Error("Change email failed", zap.Error(err))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	writeJSON(w, map[string]interface{}{
		"status":  "ok",
		"message": resp.GetMessage(),
	})

	// Отзываем refresh-токены: клиент должен перелогиниться с новым email.
	if g.db != nil {
		if _, err := g.db.Exec(
			`DELETE FROM refresh_tokens WHERE user_id = $1`, userID,
		); err != nil {
			g.log.Warn("Failed to revoke refresh tokens after email change", zap.Error(err))
		}
	}
}
