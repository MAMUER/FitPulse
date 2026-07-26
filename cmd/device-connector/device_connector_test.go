package main

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
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

func TestRegisterDeviceHandler_MissingDeviceType(t *testing.T) {
	s := setupDeviceConnector()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/register", bytes.NewReader([]byte(`{"user_id":"user-123"}`)))
	req.Header.Set("Content-Type", "application/json")

	s.registerDeviceHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterDeviceHandler_InvalidDeviceType(t *testing.T) {
	s := setupDeviceConnector()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/register", bytes.NewReader([]byte(`{"device_type":"unknown","user_id":"user-123"}`)))
	req.Header.Set("Content-Type", "application/json")

	s.registerDeviceHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterDeviceHandler_MissingUserID(t *testing.T) {
	s := setupDeviceConnector()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/register", bytes.NewReader([]byte(`{"device_type":"fitbit"}`)))
	req.Header.Set("Content-Type", "application/json")

	s.registerDeviceHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIngestInputs_MissingDeviceID(t *testing.T) {
	s := setupDeviceConnector()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices//ingest", bytes.NewReader([]byte(`{"records":[]}`)))
	req.Header.Set("Content-Type", "application/json")

	_, _, apiErr := s.ingestInputs(req)
	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
}

func TestIngestInputs_InvalidJSON(t *testing.T) {
	s := setupDeviceConnector()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/device-123/ingest", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	_, _, apiErr := s.ingestInputs(req)
	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
}

func TestIngestInputs_EmptyRecords(t *testing.T) {
	s := setupDeviceConnector()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/device-123/ingest", bytes.NewReader([]byte(`{"records":[]}`)))
	req.Header.Set("Content-Type", "application/json")

	_, _, apiErr := s.ingestInputs(req)
	assert.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
}

func TestIngestInputs_Success(t *testing.T) {
	s := setupDeviceConnector()
	body := `{"device_type":"fitbit","device_token":"token","sync_interval_ms":5000,"records":[{"metric_type":"heart_rate","value":70,"timestamp":"2024-01-01T00:00:00Z"}]}`
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/device-123/ingest", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("device_id", "device-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	deviceID, ingestReq, apiErr := s.ingestInputs(req)
	assert.Nil(t, apiErr)
	assert.Equal(t, "device-123", deviceID)
	assert.Equal(t, "fitbit", ingestReq.DeviceType)
	assert.Equal(t, 1, len(ingestReq.Records))
}

func TestValidateIngestRecord_EmptyMetricType(t *testing.T) {
	s := setupDeviceConnector()
	stats := &IngestStats{}
	rec := &IngestRecord{MetricType: "", Value: 70, Timestamp: time.Now()}

	ok := s.validateIngestRecord(rec, stats)
	assert.False(t, ok)
	assert.Equal(t, 1, stats.Failed)
}

func TestValidateIngestRecord_NegativeValue(t *testing.T) {
	s := setupDeviceConnector()
	stats := &IngestStats{}
	rec := &IngestRecord{MetricType: "heart_rate", Value: -10, Timestamp: time.Now()}

	ok := s.validateIngestRecord(rec, stats)
	assert.False(t, ok)
	assert.Equal(t, 1, stats.Failed)
}

func TestValidateIngestRecord_HeartRateOutOfRange(t *testing.T) {
	s := setupDeviceConnector()
	stats := &IngestStats{}
	rec := &IngestRecord{MetricType: "heart_rate", Value: 250, Timestamp: time.Now()}

	ok := s.validateIngestRecord(rec, stats)
	assert.False(t, ok)
	assert.Equal(t, 1, stats.Failed)
}

func TestValidateIngestRecord_SpO2OutOfRange(t *testing.T) {
	s := setupDeviceConnector()
	stats := &IngestStats{}
	rec := &IngestRecord{MetricType: "spo2", Value: 60, Timestamp: time.Now()}

	ok := s.validateIngestRecord(rec, stats)
	assert.False(t, ok)
	assert.Equal(t, 1, stats.Failed)
}

func TestValidateIngestRecord_Valid(t *testing.T) {
	s := setupDeviceConnector()
	stats := &IngestStats{}
	rec := &IngestRecord{MetricType: "heart_rate", Value: 70, Timestamp: time.Now()}

	ok := s.validateIngestRecord(rec, stats)
	assert.True(t, ok)
	assert.Equal(t, 0, stats.Failed)
}

func TestAuthenticateDevice_InvalidCredentials(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT id, user_id").
		WithArgs("device-123", "wrong-token").
		WillReturnError(sql.ErrNoRows)

	s := &deviceConnector{
		db:  db,
		log: &logger.Logger{Logger: zap.NewNop()},
	}

	device, err := s.authenticateDevice(context.Background(), "device-123", "wrong-token")
	assert.Nil(t, device)
	assert.Equal(t, "invalid device credentials", err.Error())
}

func TestAuthenticateDevice_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	mock.ExpectQuery("SELECT id, user_id").
		WithArgs("device-123", "token-123").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "device_type", "token", "created_at"}).
			AddRow("device-123", "user-123", "fitbit", "token-123", now))

	s := &deviceConnector{
		db:  db,
		log: &logger.Logger{Logger: zap.NewNop()},
	}

	device, err := s.authenticateDevice(context.Background(), "device-123", "token-123")
	require.NoError(t, err)
	assert.Equal(t, "device-123", device.ID)
	assert.Equal(t, "user-123", device.UserID)
	assert.Equal(t, "fitbit", device.DeviceType)
}
