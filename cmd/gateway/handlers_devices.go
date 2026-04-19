package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

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

	// Sanitize deviceID to prevent path injection
	deviceID = strings.ReplaceAll(deviceID, "/", "")
	deviceID = strings.ReplaceAll(deviceID, "\\", "")
	deviceID = strings.ReplaceAll(deviceID, "..", "")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		g.log.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Ошибка чтения ответа", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Build safe URL: deviceConnectorURL validated at startup, deviceID sanitized above
	ingestURL := fmt.Sprintf("%s/api/v1/devices/%s/ingest", g.deviceConnectorURL, deviceID)

	// #nosec G704 - URL validated at startup, deviceID sanitized above
	req, err := http.NewRequestWithContext(ctx, "POST",
		ingestURL,
		bytes.NewReader(body))
	if err != nil {
		g.log.Error("Failed to create device ingest request", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	// #nosec G704 - URL validated at startup, deviceID sanitized
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
