package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/MAMUER/project/internal/middleware"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func isValidDeviceID(deviceID string) bool {
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]{1,100}$`, deviceID)
	return err == nil && matched
}

func (g *gateway) listDevicesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	if g.db == nil {
		http.Error(w, "Сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	rows, err := g.db.QueryContext(r.Context(),
		"SELECT id, user_id, device_type, created_at FROM devices WHERE user_id = $1 ORDER BY created_at DESC",
		userID)
	if err != nil {
		g.log.Error("Failed to query devices", zap.Error(err))
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			g.log.Error("Failed to close rows", zap.Error(closeErr))
		}
	}()

	type deviceInfo struct {
		ID         string    `json:"id"`
		UserID     string    `json:"user_id"`
		DeviceType string    `json:"device_type"`
		CreatedAt  time.Time `json:"created_at"`
	}
	var devices []deviceInfo
	for rows.Next() {
		var d deviceInfo
		if scanErr := rows.Scan(&d.ID, &d.UserID, &d.DeviceType, &d.CreatedAt); scanErr != nil {
			g.log.Error("Failed to scan device row", zap.Error(scanErr))
			http.Error(w, "Не найдено", http.StatusNotFound)
			return
		}
		devices = append(devices, d)
	}
	if rows.Err() != nil {
		g.log.Error("Rows iteration error", zap.Error(rows.Err()))
		http.Error(w, "Не найдено", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"devices": devices,
	}); err != nil {
		g.log.Error("Failed to encode devices response", zap.Error(err))
	}
}

func (g *gateway) registerDeviceHandler(w http.ResponseWriter, r *http.Request) {
	if g.deviceConnectorURL == "" {
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	if !isValidServiceURL(g.deviceConnectorURL, "http://localhost:", "http://device-", "http://connector:") {
		g.log.Error("Invalid device connector URL", zap.String("url", g.deviceConnectorURL))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		g.log.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Ошибка чтения ответа", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST",
		g.deviceConnectorURL+"/api/v1/devices/register",
		bytes.NewReader(body))
	if err != nil {
		g.log.Error("Failed to create device register request", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		g.log.Error("Device connector unreachable", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			g.log.Error("Failed to close response body", zap.Error(closeErr))
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		g.log.Error("Failed to write response", zap.Error(err))
	}
}

func (g *gateway) deviceIngestHandler(w http.ResponseWriter, r *http.Request) {
	if g.deviceConnectorURL == "" {
		http.Error(w, "Сервис устройств временно недоступен", http.StatusServiceUnavailable)
		return
	}

	if !isValidServiceURL(g.deviceConnectorURL, "http://localhost:", "http://device-", "http://connector:") {
		g.log.Error("Invalid device connector URL", zap.String("url", g.deviceConnectorURL))
		http.Error(w, "Сервис устройств временно недоступен", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	deviceID := vars["device_id"]

	if !isValidDeviceID(deviceID) {
		g.log.Warn("Invalid device ID format", zap.String("device_id", deviceID))
		http.Error(w, "Неверный формат ID устройства", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		g.log.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Ошибка чтения запроса", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	baseURL, urlErr := url.Parse(g.deviceConnectorURL)
	if urlErr != nil {
		g.log.Error("Invalid device connector URL", zap.Error(urlErr))
		http.Error(w, "Сервис устройств временно недоступен", http.StatusServiceUnavailable)
		return
	}
	baseURL.Path = fmt.Sprintf("/api/v1/devices/%s/ingest", url.PathEscape(deviceID))

	req, err := http.NewRequestWithContext(ctx, "POST",
		baseURL.String(),
		bytes.NewReader(body))
	if err != nil {
		g.log.Error("Failed to create device ingest request", zap.Error(err))
		http.Error(w, "Сервис устройств временно недоступен", http.StatusServiceUnavailable)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		g.log.Error("Device connector unreachable", zap.Error(err))
		http.Error(w, "Сервис устройств временно недоступен", http.StatusServiceUnavailable)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			g.log.Error("Failed to close response body", zap.Error(closeErr))
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		g.log.Error("Failed to write response", zap.Error(err))
	}
}
