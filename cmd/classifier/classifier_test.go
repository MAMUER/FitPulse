package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MAMUER/project/internal/logger"
)

func setupClassifierTestServer() *server {
	return &server{log: logger.New("classifier")}
}

func TestHealthHandler(t *testing.T) {
	s := setupClassifierTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp healthResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", resp.Status)
	assert.True(t, resp.ModelLoaded)
	assert.True(t, resp.ScalerLoaded)
	assert.False(t, resp.AsyncEnabled)
}

func TestClassesHandler(t *testing.T) {
	s := setupClassifierTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/classes", nil)

	s.classesHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.True(t, len(w.Body.Bytes()) > 0)
}

func TestMetricsHandler(t *testing.T) {
	s := setupClassifierTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/metrics", nil)

	s.metricsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; version=0.0.4", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "Classifier metrics")
}

func TestModelInfoHandler(t *testing.T) {
	s := setupClassifierTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/model-info", nil)

	s.modelInfoHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.True(t, len(w.Body.Bytes()) > 0)
}

func TestClassifyHandler_InvalidJSON(t *testing.T) {
	s := setupClassifierTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/classify", bytes.NewReader([]byte(`{invalid json`)))

	s.classifyHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClassifyHandler_ValidRequest(t *testing.T) {
	s := setupClassifierTestServer()

	reqBody := classifyRequest{
		PhysiologicalData: physiologicalData{
			HeartRate:            140.0,
			HeartRateVariability: 50.0,
			SpO2:                 98.0,
			SleepHours:           7.0,
		},
		UserProfile: &userProfile{
			Age:          30,
			FitnessLevel: "intermediate",
			Goals:        []string{"endurance"},
		},
	}

	body, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	s.classifyHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp classifyResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	assert.NotEmpty(t, resp.State)
	assert.Greater(t, resp.Confidence, float64(0))
	assert.NotEmpty(t, resp.Recommendation)
	assert.NotNil(t, resp.FatigueLevel)
	assert.NotNil(t, resp.MotivationScore)
	assert.NotNil(t, resp.RecoveryQuality)
	assert.NotEmpty(t, resp.PredictedClass)
	assert.NotEmpty(t, resp.PredictedClassRu)
	assert.Len(t, resp.Probabilities, 6)
}

func TestClassifyHandler_DefaultValues(t *testing.T) {
	s := setupClassifierTestServer()

	reqBody := classifyRequest{
		PhysiologicalData: physiologicalData{
			HeartRate: 0,
		},
	}

	body, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	s.classifyHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClassifyHandler_ValidationErrors(t *testing.T) {
	s := setupClassifierTestServer()

	tests := []struct {
		name    string
		payload physiologicalData
		want    int
	}{
		{"heart_rate too low", physiologicalData{HeartRate: 10}, http.StatusBadRequest},
		{"heart_rate too high", physiologicalData{HeartRate: 300}, http.StatusBadRequest},
		{"spo2 too low", physiologicalData{HeartRate: 100, SpO2: 50}, http.StatusBadRequest},
		{"temperature too high", physiologicalData{HeartRate: 100, Temperature: 50}, http.StatusBadRequest},
		{"bp systolic too low", physiologicalData{HeartRate: 100, BloodPressureSystolic: 30}, http.StatusBadRequest},
		{"sleep hours too high", physiologicalData{HeartRate: 100, SleepHours: 30}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := classifyRequest{PhysiologicalData: tt.payload}
			body, _ := json.Marshal(reqBody)
			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), "POST", "/classify", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			s.classifyHandler(w, req)
			assert.Equal(t, tt.want, w.Code)
		})
	}
}

func TestClassifyHandler_MethodNotAllowed(t *testing.T) {
	s := setupClassifierTestServer()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/classify", nil)

	s.classifyHandler(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
