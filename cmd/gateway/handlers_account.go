package main

import (
	"net/http"

	"github.com/MAMUER/project/internal/middleware"
	"go.uber.org/zap"
)

// deleteAccountHandler удаляет аккаунт пользователя и все его данные.
// Большинство таблиц имеют FK ... ON DELETE CASCADE от users(id), поэтому
// достаточно удалить строку users. Таблицы без каскада (friends, friend_requests,
// nutrition_meals, calendar_events, menstrual_cycles, leaderboard, reports)
// чистим явно (ошибки «table not found» игнорируем).
func (g *gateway) deleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		g.writeJSON(w, http.StatusUnauthorized, map[string]interface{}{"error": msgUnauthorized})
		return
	}
	if g.db == nil {
		g.writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"error": "db unavailable"})
		return
	}

	// Явная очистка таблиц, которые могут не иметь ON DELETE CASCADE.
	explicit := []struct {
		q    string
		args []interface{}
	}{
		{"DELETE FROM friends WHERE user_id = $1", []interface{}{userID}},
		{"DELETE FROM friend_requests WHERE from_user_id = $1", []interface{}{userID}},
		{"DELETE FROM nutrition_meals WHERE user_id = $1", []interface{}{userID}},
		{"DELETE FROM calendar_events WHERE user_id = $1", []interface{}{userID}},
		{"DELETE FROM menstrual_cycles WHERE user_id = $1", []interface{}{userID}},
		{"DELETE FROM leaderboard WHERE user_id = $1", []interface{}{userID}},
		{"DELETE FROM reports WHERE reporter_id = $1", []interface{}{userID}},
	}
	for _, stmt := range explicit {
		// Ошибки (напр. таблицы нет) не фатальны — чистим что есть.
		_, _ = g.db.Exec(stmt.q, stmt.args...)
	}

	// Удаление пользователя каскадит остальные FK-таблицы.
	if _, err := g.db.Exec("DELETE FROM users WHERE id = $1", userID); err != nil {
		g.log.Error("delete account failed", zap.Error(err))
		g.writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "failed to delete account"})
		return
	}

	g.writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "message": "account deleted"})
}