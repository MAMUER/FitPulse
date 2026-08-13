package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MAMUER/project/internal/middleware"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
)

func TestML_ClassifyHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()
	withClassifierURL(g, "http://localhost:8001")

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/ml/classify", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.classifyHandler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestML_ClassifyHandler_InvalidClassifierURL(t *testing.T) {
	g := newTestGateway()
	g.classifierURL = "invalid-url"

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/ml/classify", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.classifyHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestML_ClassifyHandler_ClassifierUnavailable(t *testing.T) {
	g := newTestGateway()
	withClassifierURL(g, "http://localhost:99999")
	withBiometricClient(g)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/ml/classify", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.classifyHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestML_ClassifyHandler_Success(t *testing.T) {
	g := newTestGateway()
	withBiometricClient(g)

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"predicted_class": "endurance_basic",
			"confidence":      0.95,
			"recommendations": []string{"increase cardio", "rest more"},
		})
	})}
	go server.Serve(listener)
	defer server.Close()

	withClassifierURL(g, "http://localhost:"+strconv.Itoa(port))

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "user-123"), "POST", "/api/v1/ml/classify", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.classifyHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
	assert.Contains(t, w.Body.String(), "endurance_basic")
}

func TestML_AggregateMLPayload(t *testing.T) {
	tests := []struct {
		name string
		metrics map[string]*biometricpb.BiometricRecord
		wantPhysiologicalData map[string]interface{}
	}{
		{
			name: "with metrics",
			metrics: map[string]*biometricpb.BiometricRecord{
				"heart_rate": {MetricType: "heart_rate", Value: 70},
				"hrv":        {MetricType: "hrv", Value: 50},
			},
			wantPhysiologicalData: map[string]interface{}{
				"heart_rate": float64(70),
				"heart_rate_variability": float64(50),
			},
		},
		{
			name: "nil record",
			metrics: map[string]*biometricpb.BiometricRecord{
				"heart_rate": nil,
			},
			wantPhysiologicalData: map[string]interface{}{
				"heart_rate": nil,
			},
		},
		{
			name:     "empty metrics",
			metrics:  map[string]*biometricpb.BiometricRecord{},
			wantPhysiologicalData: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aggregateMLPayload(tt.metrics)
			physData, ok := result["physiological_data"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, tt.wantPhysiologicalData, physData)
		})
	}
}

func TestML_MapMetricTypeToClassifierKey(t *testing.T) {
	tests := []struct {
		metricType string
		want       string
	}{
		{"heart_rate", "heart_rate"},
		{"hrv", "heart_rate_variability"},
		{"systolic_pressure", "blood_pressure_systolic"},
		{"diastolic_pressure", "blood_pressure_diastolic"},
		{"spo2", "spo2"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.metricType, func(t *testing.T) {
			got := mapMetricTypeToClassifierKey(tt.metricType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestML_TransformClassifierResponse(t *testing.T) {
	tests := []struct {
		name           string
		result         map[string]interface{}
		wantStatus     string
		wantState      string
		wantConfidence float64
	}{
		{
			name: "recovery state",
			result: map[string]interface{}{
				"predicted_class": "recovery",
				"confidence":      0.9,
				"recommendations": []interface{}{"rest", "sleep"},
			},
			wantStatus:     "success",
			wantState:      "recovery",
			wantConfidence: 0.9,
		},
		{
			name: "overtraining state",
			result: map[string]interface{}{
				"predicted_class": "overtraining",
				"confidence":      0.8,
			},
			wantStatus:     "success",
			wantState:      "overtraining",
			wantConfidence: 0.8,
		},
		{
			name: "unknown state",
			result: map[string]interface{}{
				"confidence": 0.5,
			},
			wantStatus:     "success",
			wantState:      "",
			wantConfidence: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transformClassifierResponse(tt.result)
			assert.Equal(t, tt.wantStatus, got["status"])
			assert.Equal(t, tt.wantState, got["state"])
			assert.Equal(t, tt.wantConfidence, got["confidence"])
		})
	}
}

func TestML_MapClassToScores(t *testing.T) {
	tests := []struct {
		predictedClass string
		wantFatigue    float64
		wantMotivation float64
		wantRecovery   float64
	}{
		{"recovery", 0.1, 0.8, 0.9},
		{"endurance_basic", 0.3, 0.7, 0.7},
		{"endurance_threshold", 0.5, 0.6, 0.5},
		{"power_hiit", 0.7, 0.5, 0.3},
		{"overtraining", 0.9, 0.2, 0.1},
		{"illness", 1.0, 0.0, 0.0},
		{"unknown", 0.5, 0.5, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.predictedClass, func(t *testing.T) {
			fatigue, motivation, recovery := mapClassToScores(tt.predictedClass)
			assert.Equal(t, tt.wantFatigue, fatigue)
			assert.Equal(t, tt.wantMotivation, motivation)
			assert.Equal(t, tt.wantRecovery, recovery)
		})
	}
}

func TestML_CallClassifier_InvalidURL(t *testing.T) {
	g := newTestGateway()
	g.classifierURL = "invalid-url"

	ctx := context.Background()
	result, err := g.callClassifier(ctx, []byte(`{}`))

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestML_ProxyToMLGenerator_InvalidURL(t *testing.T) {
	g := newTestGateway()
	g.mlGeneratorURL = "invalid-url"

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/ml/generate", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.proxyToMLGenerator(w, req, "/generate-plan")

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestML_ProxyToMLGenerator_ReadBodyError(t *testing.T) {
	g := newTestGateway()
	withMLGeneratorURL(g, "http://localhost:99999")

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/ml/generate", bytes.NewReader([]byte(`{}`)))
	req.Body = &errorReader{err: errors.New("read error")}

	g.proxyToMLGenerator(w, req, "/generate-plan")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func (e *errorReader) Close() error {
	return nil
}

func TestML_MLGenerateHandler_InvalidURL(t *testing.T) {
	g := newTestGateway()
	g.mlGeneratorURL = "invalid-url"

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/ml/generate", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.mlGenerateHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestML_MLDietHandler_InvalidURL(t *testing.T) {
	g := newTestGateway()
	g.mlGeneratorURL = "invalid-url"

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/ml/diet", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	g.mlDietHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestML_IsValidServiceURL(t *testing.T) {
	tests := []struct {
		url     string
		prefixes []string
		want    bool
	}{
		{"http://localhost:8001", []string{"http://localhost:"}, true},
		{"https://localhost:8001", []string{"http://localhost:"}, false},
		{"http://classifier:8001", []string{"http://classifier:"}, true},
		{"ftp://localhost:8001", []string{"http://localhost:"}, false},
		{"http://evil.com", []string{"http://localhost:"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := isValidServiceURL(tt.url, tt.prefixes...)
			assert.Equal(t, tt.want, got)
		})
	}
}
