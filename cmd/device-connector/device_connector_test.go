package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
)

func setupDeviceConnector() *deviceConnector {
	log := zap.NewNop()
	logger := &logger.Logger{Logger: log}
	return &deviceConnector{log: logger}
}

func TestIsValidDeviceType(t *testing.T) {
	tests := []struct {
		name     string
		deviceID string
		want     bool
	}{
		{"fitbit", "fitbit", true},
		{"garmin", "garmin", true},
		{"withings", "withings", true},
		{"invalid type", "unknown_device", false},
		{"empty string", "", false},
		{"case sensitive", "Fitbit", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidDeviceType(tt.deviceID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMetricSyncRules(t *testing.T) {
	tests := []struct {
		name     string
		metric   string
		wantOK   bool
		wantName string
	}{
		{"heart rate", "heart_rate", true, "heart_rate"},
		{"spo2", "spo2", true, "spo2"},
		{"steps", "steps", true, "steps"},
		{"sleep", "sleep", true, "sleep"},
		{"unknown metric", "unknown", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minMs, maxMs, name, ok := metricSyncRules(tt.metric)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantName, name)
				assert.GreaterOrEqual(t, maxMs, minMs)
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	s := &deviceConnector{
		db:  db,
		log: &logger.Logger{Logger: zap.NewNop()},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "device-connector")
	assert.Contains(t, w.Body.String(), "ok")
}

func TestRegisterDeviceHandler_InvalidJSON(t *testing.T) {
	s := setupDeviceConnector()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/register", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	s.registerDeviceHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
