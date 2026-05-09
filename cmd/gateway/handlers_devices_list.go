package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/MAMUER/project/internal/middleware"
	"go.uber.org/zap"
)

// getDevicesHandler returns list of devices for authenticated user
func (g *gateway) getDevicesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	if g.db == nil {
		http.Error(w, "Сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	rows, err := g.db.QueryContext(r.Context(), `
		SELECT id, device_type, created_at
		FROM devices
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		g.log.Error("Failed to query devices", zap.Error(err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			g.log.Error("Failed to close rows", zap.Error(closeErr))
		}
	}()

	devices := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, deviceType string
		var createdAt time.Time
		if err := rows.Scan(&id, &deviceType, &createdAt); err != nil {
			g.log.Error("Failed to scan device", zap.Error(err))
			continue
		}
		devices = append(devices, map[string]interface{}{
			"device_id":   id,
			"device_type": deviceType,
			"created_at":  createdAt.Format(time.RFC3339),
		})
	}

	if err := rows.Err(); err != nil {
		g.log.Error("Row iteration error", zap.Error(err))
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"devices": devices,
	}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}
