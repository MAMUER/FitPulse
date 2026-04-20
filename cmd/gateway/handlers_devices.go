package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// isValidDeviceID validates device ID format to prevent injection
func isValidDeviceID(deviceID string) bool {
	// UUID format: 8-4-4-4-12 hexadecimal digits
	// Also allow alphanumeric with hyphens and underscores
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]{1,100}$`, deviceID)
	return err == nil && matched
}

// deviceRegisterHandler proxies device registration to device-connector
func (g *gateway) deviceRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if g.deviceConnectorURL == "" {
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	// Validate device connector URL to prevent SSRF
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

// deviceIngestHandler proxies data ingestion to device-connector
func (g *gateway) deviceIngestHandler(w http.ResponseWriter, r *http.Request) {
	if g.deviceConnectorURL == "" {
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	// Validate device connector URL to prevent SSRF
	if !isValidServiceURL(g.deviceConnectorURL, "http://localhost:", "http://device-", "http://connector:") {
		g.log.Error("Invalid device connector URL", zap.String("url", g.deviceConnectorURL))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	deviceID := vars["device_id"]

	// Validate device ID format to prevent SSRF/injection attacks
	if !isValidDeviceID(deviceID) {
		g.log.Warn("Invalid device ID format", zap.String("device_id", deviceID))
		http.Error(w, "Неверный формат ID устройства", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		g.log.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Ошибка чтения ответа", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Build URL safely using url.URL to prevent SSRF
	baseURL, urlErr := url.Parse(g.deviceConnectorURL)
	if urlErr != nil {
		g.log.Error("Invalid device connector URL", zap.Error(urlErr))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}
	baseURL.Path = fmt.Sprintf("/api/v1/devices/%s/ingest", url.PathEscape(deviceID))

	req, err := http.NewRequestWithContext(ctx, "POST",
		baseURL.String(),
		bytes.NewReader(body))
	if err != nil {
		g.log.Error("Failed to create device ingest request", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
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
