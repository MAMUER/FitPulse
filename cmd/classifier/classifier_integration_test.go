package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/middleware"
)

func TestClassifierIntegration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	log := logger.New("classifier-test")
	s := &server{log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("/classify", s.classifyHandler)
	mux.HandleFunc("/health", s.healthHandler)

	handler := middleware.RecoveryMiddleware(log.Logger)(mux)
	handler = middleware.RequestID(handler)
	handler = classifierLoggingMiddleware(log.Logger)(handler)

	srv := &http.Server{Addr: ":0", Handler: handler}
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !isServerClosed(err) {
			t.Logf("server error: %v", err)
		}
	}()
	defer func() { _ = srv.Close() }()

	baseURL := "http://" + ln.Addr().String()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("health check", func(t *testing.T) {
		req, _ := http.NewRequestWithContext(ctx, "GET", baseURL+"/health", nil)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("classify valid request", func(t *testing.T) {
		reqBody := classifyRequest{
			PhysiologicalData: physiologicalData{
				HeartRate:              160,
				HeartRateVariability:   45,
				SpO2:                   96,
				Temperature:            36.6,
				BloodPressureSystolic:  120,
				BloodPressureDiastolic: 80,
				SleepHours:             7,
			},
			UserProfile: &userProfile{
				Age:          28,
				FitnessLevel: "intermediate",
				Goals:        []string{"endurance"},
			},
		}
		payload, _ := json.Marshal(reqBody)
		req, _ := http.NewRequestWithContext(ctx, "POST", baseURL+"/classify", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result classifyResponse
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, "success", result.Status)
		assert.NotEmpty(t, result.State)
		assert.Greater(t, result.Confidence, 0.0)
		assert.NotNil(t, result.FatigueLevel)
		assert.NotNil(t, result.MotivationScore)
		assert.NotNil(t, result.RecoveryQuality)
		assert.Len(t, result.Probabilities, 6)
	})

	t.Run("classify invalid json", func(t *testing.T) {
		req, _ := http.NewRequestWithContext(ctx, "POST", baseURL+"/classify", bytes.NewReader([]byte(`{invalid`)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("classify validation error", func(t *testing.T) {
		reqBody := classifyRequest{PhysiologicalData: physiologicalData{HeartRate: 500}}
		payload, _ := json.Marshal(reqBody)
		req, _ := http.NewRequestWithContext(ctx, "POST", baseURL+"/classify", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func isServerClosed(err error) bool {
	if err == nil {
		return false
	}
	return err == http.ErrServerClosed
}
