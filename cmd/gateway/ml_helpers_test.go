package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
)

func TestMapMetricTypeToClassifierKey(t *testing.T) {
	assert.Equal(t, "heart_rate_variability", mapMetricTypeToClassifierKey("hrv"))
	assert.Equal(t, "blood_pressure_systolic", mapMetricTypeToClassifierKey("systolic_pressure"))
	assert.Equal(t, "blood_pressure_diastolic", mapMetricTypeToClassifierKey("diastolic_pressure"))
	assert.Equal(t, "custom_metric", mapMetricTypeToClassifierKey("custom_metric"))
}

func TestMapClassToScores(t *testing.T) {
	cases := map[string][3]float64{
		"recovery":            {0.1, 0.8, 0.9},
		"endurance_basic":     {0.3, 0.7, 0.7},
		"endurance_threshold": {0.5, 0.6, 0.5},
		"power_hiit":          {0.7, 0.5, 0.3},
		"overtraining":        {0.9, 0.2, 0.1},
		"illness":             {1.0, 0.0, 0.0},
		"unknown":             {0.5, 0.5, 0.5},
	}
	for class, want := range cases {
		f, m, r := mapClassToScores(class)
		assert.Equal(t, want[0], f, class)
		assert.Equal(t, want[1], m, class)
		assert.Equal(t, want[2], r, class)
	}
}

func TestAggregateMLPayload(t *testing.T) {
	metrics := map[string]*biometricpb.BiometricRecord{
		"hrv":       {Value: 42},
		"nullified": nil,
	}
	out := aggregateMLPayload(metrics)
	data, ok := out["physiological_data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(42), data["heart_rate_variability"])
	assert.Nil(t, data["nullified"])
}

func TestTransformClassifierResponse_Full(t *testing.T) {
	out := transformClassifierResponse(map[string]interface{}{
		"predicted_class": "recovery",
		"confidence":      float64(0.9),
		"recommendations": []interface{}{"rest", "sleep"},
	})
	assert.Equal(t, "success", out["status"])
	assert.Equal(t, "recovery", out["state"])
	assert.Equal(t, 0.9, out["confidence"])
	assert.Equal(t, 0.1, out["fatigue_level"])
	assert.Equal(t, 0.8, out["motivation_score"])
	assert.Equal(t, 0.9, out["recovery_quality"])
}

func TestTransformClassifierResponse_Empty(t *testing.T) {
	out := transformClassifierResponse(map[string]interface{}{})
	assert.Equal(t, "", out["state"])
	assert.Equal(t, float64(0), out["confidence"])
	assert.Equal(t, 0.5, out["fatigue_level"])
}

func TestCallClassifier_InvalidURL(t *testing.T) {
	g := newTestGateway()
	g.classifierURL = "ftp://evil.example"

	_, err := g.callClassifier(context.Background(), []byte(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "URL")
}

func TestCallClassifier_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"predicted_class":"recovery","confidence":0.8}`))
	}))
	defer srv.Close()

	g := newTestGateway()
	g.classifierURL = srv.URL

	res, err := g.callClassifier(context.Background(), []byte(`{}`))
	require.NoError(t, err)
	assert.Equal(t, "recovery", res["predicted_class"])
}

func TestCallClassifier_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	g := newTestGateway()
	g.classifierURL = srv.URL

	_, err := g.callClassifier(context.Background(), []byte(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "classifier status")
}

func TestCallClassifier_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	g := newTestGateway()
	g.classifierURL = srv.URL

	_, err := g.callClassifier(context.Background(), []byte(`{}`))
	require.Error(t, err)
}
