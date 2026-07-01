package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/MAMUER/project/internal/middleware"
	"github.com/MAMUER/project/internal/sanitize"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// adminListUsersHandler handles admin user listing with server-side role re-verification.
func (g *gateway) adminListUsersHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}

	if !g.verifyUserRole(r.Context(), userID, "admin") {
		g.log.Warn("Non-admin attempted to access user list", zap.String("user_id", userID))
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}

	if g.db == nil {
		g.log.Error("Database not available for user listing")
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	pageSize := 20
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 {
			pageSize = val
		}
	}
	offset := (page - 1) * pageSize

	rows, err := g.db.QueryContext(r.Context(),
		"SELECT id, email, full_name, role, created_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		pageSize, offset)
	if err != nil {
		g.log.Error("Failed to query users", zap.Error(err))
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			g.log.Error("Failed to close rows", zap.Error(closeErr))
		}
	}()

	type userInfo struct {
		ID        string    `json:"id"`
		Email     string    `json:"email"`
		FullName  string    `json:"full_name"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
	}
	var users []userInfo
	for rows.Next() {
		var u userInfo
		if scanErr := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.CreatedAt); scanErr != nil {
			g.log.Error("Failed to scan user row", zap.Error(scanErr))
			http.Error(w, "Не найдено", http.StatusNotFound)
			return
		}
		users = append(users, u)
	}
	if scanErr := rows.Err(); scanErr != nil {
		g.log.Error("Rows iteration error", zap.Error(scanErr))
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}

	adminResp := map[string]interface{}{"status": "ok", "users": users, "total": len(users)}
	if err := middleware.SignAndSendJSON(w, adminResp, g.responseSigningSecret, g.log.Logger); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

// adminListInvitesHandler — список всех инвайт-кодов (только для админов)
func (g *gateway) adminListInvitesHandler(w http.ResponseWriter, r *http.Request) {
	if g.db == nil {
		http.Error(w, "Сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	pageSize := 50
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 && val <= 100 {
			pageSize = val
		}
	}
	offset := (page - 1) * pageSize

	rows, err := g.db.QueryContext(r.Context(), `
		SELECT code, role, specialty, max_uses, used_count, is_active, created_at
		FROM invite_codes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		g.log.Error("Failed to query invites", zap.Error(err))
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			g.log.Warn("Failed to close rows", zap.Error(closeErr))
		}
	}()

	type inviteInfo struct {
		Code      string    `json:"code"`
		Role      string    `json:"role"`
		Specialty string    `json:"specialty,omitempty"`
		MaxUses   int       `json:"max_uses"`
		UsedCount int       `json:"used_count"`
		IsActive  bool      `json:"is_active"`
		CreatedAt time.Time `json:"created_at"`
		InviteURL string    `json:"invite_url"`
	}

	var invites []inviteInfo
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "https://fittpulse.duckdns.org"
	}
	for rows.Next() {
		var inv inviteInfo
		var specialty sql.NullString
		if err := rows.Scan(&inv.Code, &inv.Role, &specialty, &inv.MaxUses, &inv.UsedCount, &inv.IsActive, &inv.CreatedAt); err != nil {
			g.log.Error("Failed to scan invite", zap.Error(err))
			continue
		}
		if specialty.Valid {
			inv.Specialty = specialty.String
		}
		inv.InviteURL = fmt.Sprintf("%s/register?invite=%s", baseURL, inv.Code)
		invites = append(invites, inv)
	}

	if invites == nil {
		invites = []inviteInfo{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := middleware.SignAndSendJSON(w, map[string]interface{}{
		"status":  "ok",
		"invites": invites,
		"page":    page,
	}, g.responseSigningSecret, g.log.Logger); err != nil {
		g.log.Error("Failed to encode invites response", zap.Error(err))
	}
}

// adminCreateInviteHandler — создание нового инвайт-кода
func (g *gateway) adminCreateInviteHandler(w http.ResponseWriter, r *http.Request) {
	if g.db == nil {
		http.Error(w, "Сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	userID, _ := r.Context().Value(middleware.UserIDKey).(string)

	var req struct {
		Role      string `json:"role"`
		Specialty string `json:"specialty"`
		MaxUses   int    `json:"max_uses"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = "client"
	}
	if req.Role != "client" && req.Role != "admin" {
		http.Error(w, "Роль должна быть 'client' или 'admin'", http.StatusBadRequest)
		return
	}
	if req.MaxUses <= 0 {
		req.MaxUses = 1
	}
	if req.MaxUses > 100 {
		http.Error(w, "Максимум 100 использований", http.StatusBadRequest)
		return
	}

	code := "INV-" + generateInviteCode()

	var specialty interface{}
	if req.Specialty != "" {
		specialty = req.Specialty
	}

	_, err := g.db.ExecContext(r.Context(), `
		INSERT INTO invite_codes (code, role, specialty, max_uses, created_by, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, TRUE, NOW())
	`, code, req.Role, specialty, req.MaxUses, userID)

	if err != nil {
		g.log.Error("Failed to create invite", zap.Error(err))
		http.Error(w, "Ошибка создания инвайта", http.StatusInternalServerError)
		return
	}

	g.log.Info("Invite code created",
		zap.String("code", code),
		zap.String("role", sanitize.LogString(req.Role)),
		zap.Int("max_uses", req.MaxUses),
		zap.String("created_by", userID))

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "https://2.27.32.242:30443"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := middleware.SignAndSendJSON(w, map[string]interface{}{
		"status":     "ok",
		"code":       code,
		"role":       req.Role,
		"max_uses":   req.MaxUses,
		"specialty":  req.Specialty,
		"invite_url": fmt.Sprintf("%s/register?invite=%s", baseURL, code),
	}, g.responseSigningSecret, g.log.Logger, http.StatusCreated); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

// adminRevokeInviteHandler — деактивация инвайт-кода
func (g *gateway) adminRevokeInviteHandler(w http.ResponseWriter, r *http.Request) {
	if g.db == nil {
		http.Error(w, "Сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		http.Error(w, "Код инвайта не указан", http.StatusBadRequest)
		return
	}

	result, err := g.db.ExecContext(r.Context(), `
		UPDATE invite_codes SET is_active = FALSE WHERE code = $1
	`, code)
	if err != nil {
		g.log.Error("Failed to revoke invite", zap.Error(err))
		http.Error(w, "Ошибка отзыва инвайта", http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Инвайт не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := middleware.SignAndSendJSON(w, map[string]string{"status": "ok"}, g.responseSigningSecret, g.log.Logger); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

func generateInviteCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}
