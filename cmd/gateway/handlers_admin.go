package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MAMUER/Project/internal/middleware"
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
	// Требование #11: SignAndSendJSON гарантирует подпись тех же байтов, что отправляются клиенту
	if err := middleware.SignAndSendJSON(w, adminResp, g.jwtSecret, g.log.Logger); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}
