package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	userpb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/middleware"
)

func (g *gateway) listDevicesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	resp, err := g.userClient.ListDevices(r.Context(), &userpb.ListDevicesRequest{
		UserId: userID,
	})
	if err != nil {
		g.log.Error("Failed to list devices", zap.Error(err), zap.String("user_id", userID))
		httpCode, errMsg := grpcToHTTPStatus(err)
		http.Error(w, errMsg, httpCode)
		return
	}

	devices := make([]map[string]interface{}, len(resp.GetDevices()))
	for i, d := range resp.GetDevices() {
		devices[i] = map[string]interface{}{
			"id":          d.GetDeviceId(),
			"user_id":     userID,
			"device_type": d.GetDeviceType(),
			"created_at":  d.GetLastSync(),
		}
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
