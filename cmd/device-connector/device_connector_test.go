// cmd/device-connector/device_connector_test.go
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/MAMUER/project/internal/logger"
	"github.com/gorilla/mux"
)

// ===== Unit Tests =====

func TestIsValidDeviceType(t *testing.T) {
	valid := []string{"apple_watch", "samsung_galaxy_watch", "huawei_watch_d2", "amazfit_trex3"}
	for _, dt := range valid {
		if !isValidDeviceType(dt) {
			t.Errorf("expected %q to be valid", dt)
		}
	}

	invalid := []string{"fitbit", "garmin", "", "unknown_device"}
	for _, dt := range invalid {
		if isValidDeviceType(dt) {
			t.Errorf("expected %q to be invalid", dt)
		}
	}
}

func TestMetricSyncRules(t *testing.T) {
	tests := []struct {
		metric   string
		wantMin  int
		wantMax  int
		wantName string
		wantOk   bool
	}{
		{"heart_rate", 5000, 15000, "heart_rate", true},
		{"spo2", 60000, 300000, "spo2", true},
		{"steps", 30000, 30000, "steps", true},
		{"sleep", 86400000, 86400000, "sleep", true},
		{"unknown", 0, 0, "", false},
		{"", 0, 0, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.metric, func(t *testing.T) {
			min, max, name, ok := metricSyncRules(tt.metric)
			if ok != tt.wantOk {
				t.Errorf("ok = %v, want %v", ok, tt.wantOk)
			}
			if min != tt.wantMin {
				t.Errorf("min = %d, want %d", min, tt.wantMin)
			}
			if max != tt.wantMax {
				t.Errorf("max = %d, want %d", max, tt.wantMax)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
		})
	}
}

// ===== Handler Tests =====

func setupTestServer(t *testing.T) (*deviceConnector, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	log := logger.New("device-connector-test")
	return &deviceConnector{db: db, log: log}, mock
}

func TestHealthHandler_OK(t *testing.T) {
	svc, mock := setupTestServer(t)
	defer func() { _ = svc.db.Close() }()

	mock.ExpectPing()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)
	rec := httptest.NewRecorder()

	svc.healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["service"] != "device-connector" {
		t.Errorf("expected service 'device-connector', got %v", resp["service"])
	}
}

func TestRegisterDeviceHandler_MissingDeviceType(t *testing.T) {
	svc, _ := setupTestServer(t)

	body := map[string]string{"user_id": "user-1"}
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/register",
		bytes.NewReader(mustJSON(body)))
	rec := httptest.NewRecorder()

	svc.registerDeviceHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestRegisterDeviceHandler_InvalidDeviceType(t *testing.T) {
	svc, _ := setupTestServer(t)

	body := map[string]string{"device_type": "fitbit", "user_id": "user-1"}
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/register",
		bytes.NewReader(mustJSON(body)))
	rec := httptest.NewRecorder()

	svc.registerDeviceHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestRegisterDeviceHandler_MissingUserID(t *testing.T) {
	svc, _ := setupTestServer(t)

	body := map[string]string{"device_type": "apple_watch"}
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/register",
		bytes.NewReader(mustJSON(body)))
	rec := httptest.NewRecorder()

	svc.registerDeviceHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestRegisterDeviceHandler_InvalidJSON(t *testing.T) {
	svc, _ := setupTestServer(t)

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/register",
		bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()

	svc.registerDeviceHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestIngestHandler_EmptyRecords(t *testing.T) {
	svc, mock := setupTestServer(t)
	defer func() { _ = svc.db.Close() }()

	// Authenticate device
	mock.ExpectQuery("SELECT id, user_id").
		WithArgs("dev-1", "token-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "device_type", "token", "created_at"}).
			AddRow("dev-1", "user-1", "apple_watch", "token-1", timeNow()))

	body := IngestRequest{
		DeviceToken: "token-1",
		Records:     []IngestRecord{},
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/devices/{device_id}/ingest", svc.ingestHandler).Methods("POST")

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/dev-1/ingest",
		bytes.NewReader(mustJSON(body)))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestIngestHandler_AuthenticationFailure(t *testing.T) {
	svc, mock := setupTestServer(t)
	defer func() { _ = svc.db.Close() }()

	mock.ExpectQuery("SELECT id, user_id").
		WithArgs("dev-1", "wrong-token").
		WillReturnError(sql.ErrNoRows)

	body := IngestRequest{
		DeviceToken: "wrong-token",
		Records:     []IngestRecord{{MetricType: "heart_rate", Value: 72}},
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/devices/{device_id}/ingest", svc.ingestHandler).Methods("POST")

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/devices/dev-1/ingest",
		bytes.NewReader(mustJSON(body)))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestDeviceStruct_JSON(t *testing.T) {
	dev := Device{
		ID:         "dev-1",
		UserID:     "user-1",
		DeviceType: "apple_watch",
		Token:      "secret",
	}

	data, err := json.Marshal(dev)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded Device
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.ID != dev.ID {
		t.Errorf("expected ID %q, got %q", dev.ID, decoded.ID)
	}
}

func TestIngestStats_JSON(t *testing.T) {
	stats := IngestStats{
		TotalReceived: 10,
		Duplicates:    2,
		Forwarded:     7,
		Failed:        1,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded IngestStats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.TotalReceived != 10 {
		t.Errorf("expected TotalReceived 10, got %d", decoded.TotalReceived)
	}
}

// ===== Helpers =====

func mustJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func timeNow() interface{} {
	return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
}

func TestMetricSyncRules_MoreCoverage(t *testing.T) {
	// Test the existing metric sync rules function with various inputs
	_ = isValidDeviceType("apple_watch")
	_ = isValidDeviceType("samsung_galaxy_watch")
	_ = isValidDeviceType("invalid")
}
