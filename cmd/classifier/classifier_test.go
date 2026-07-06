package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func setupClassifierTestServer() *server {
	logger, _ := zap.NewDevelopment()
	return &server{log: logger}
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
	assert.NotEmpty(t, resp.PredictedClass)
	assert.NotEmpty(t, resp.PredictedClassRu)
	assert.Greater(t, resp.Confidence, float64(0))
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
