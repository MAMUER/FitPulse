package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassToGANIndex(t *testing.T) {
	tests := []struct {
		name      string
		class     string
		wantIndex int
	}{
		{"recovery class", "recovery", 0},
		{"endurance_basic class", "endurance_basic", 1},
		{"endurance_threshold class", "endurance_threshold", 2},
		{"power_hiit class", "power_hiit", 3},
		{"overtraining class", "overtraining", 4},
		{"illness class", "illness", 5},
		{"unknown class defaults to endurance_basic", "unknown", 1},
		{"empty class defaults to endurance_basic", "", 1},
		{"garbage class defaults to endurance_basic", "garbage_class", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classToGANIndex(tt.class)
			assert.Equal(t, tt.wantIndex, got)
		})
	}
}

func TestVectorToGANParamsEmpty(t *testing.T) {
	params := vectorToGANParams(nil)
	assert.Equal(t, 0.5, params.Intensity)
	assert.Equal(t, 45, params.DurationMinutes)
	assert.Equal(t, 3, params.SessionsPerWeek)
	assert.Nil(t, params.RawVector)
	assert.Equal(t, "template", params.GenerationMethod)

	params = vectorToGANParams([]float64{})
	assert.Equal(t, 0.5, params.Intensity)
	assert.Equal(t, "template", params.GenerationMethod)
}

func TestVectorToGANParamsFull(t *testing.T) {
	vec := []float64{
		0.75, 0.50, 0.67,
		0.10, 0.20, 0.30, 0.40, 0.50, 0.60, 0.70, 0.80,
		0.90, 0.05, 0.15, 0.25, 0.35, 0.45, 0.55,
		0.65,
	}
	params := vectorToGANParams(vec)

	assert.Equal(t, 0.75, params.Intensity)
	assert.Equal(t, 60, params.DurationMinutes)
	assert.Equal(t, 5, params.SessionsPerWeek)
	assert.Equal(t, vec, params.RawVector)
	assert.Equal(t, "gan", params.GenerationMethod)
}

func TestVectorToGANParamsEdgeCases(t *testing.T) {
	params := vectorToGANParams([]float64{-0.5, 0.5, 0.5})
	assert.Equal(t, 0.0, params.Intensity)

	params = vectorToGANParams([]float64{1.5, 0.5, 0.5})
	assert.Equal(t, 1.0, params.Intensity)

	params = vectorToGANParams([]float64{0.5, 0.0, 0.5})
	assert.Equal(t, 30, params.DurationMinutes)

	params = vectorToGANParams([]float64{0.5, 1.0, 0.5})
	assert.Equal(t, 90, params.DurationMinutes)

	params = vectorToGANParams([]float64{0.5, 0.5, 0.0})
	assert.Equal(t, 1, params.SessionsPerWeek)

	params = vectorToGANParams([]float64{0.5, 0.5, 1.0})
	assert.Equal(t, 7, params.SessionsPerWeek)

	params = vectorToGANParams([]float64{0.8})
	assert.Equal(t, 0.8, params.Intensity)
	assert.Equal(t, 45, params.DurationMinutes)
	assert.Equal(t, 3, params.SessionsPerWeek)
}

func TestCallGANServiceEmptyURL(t *testing.T) {
	s := &trainingServer{ganURL: ""}
	resp, err := s.callGANService(1, userProfile{Age: 30})
	assert.Error(t, err)
	assert.EqualError(t, err, "gan service url not configured")
	assert.Nil(t, resp)
}

func TestCallGANServiceSuccess(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/gan/generate", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &reqBody))
		assert.Equal(t, float64(2), reqBody["class_index"])
		assert.Equal(t, float64(25), reqBody["age"])
		assert.Equal(t, float64(1), reqBody["gender"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "ok",
			"class_index": 2,
			"plans": [
				{"sample": 0, "plan": [0.8, 0.6, 0.7, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.05, 0.15, 0.25, 0.35, 0.45, 0.55, 0.65]}
			],
			"model": "conditional_gan_v1"
		}`))
	}))
	defer mockServer.Close()

	s := &trainingServer{ganURL: mockServer.URL}
	resp, err := s.callGANService(2, userProfile{Age: 25, Gender: 1, Weight: 70.0, Height: 175.0, Goal: 0})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, 2, resp.ClassIndex)
	assert.Equal(t, "conditional_gan_v1", resp.Model)
	require.Len(t, resp.Plans, 1)
	require.Len(t, resp.Plans[0].Plan, 19)
	assert.Equal(t, 0.8, resp.Plans[0].Plan[0])
	assert.Equal(t, 0, resp.Plans[0].Sample)
}

func TestCallGANServiceNon200(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error": "model loading"}`))
	}))
	defer mockServer.Close()

	s := &trainingServer{ganURL: mockServer.URL}
	resp, err := s.callGANService(1, userProfile{Age: 30})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gan service returned status 503")
	assert.Nil(t, resp)
}

func TestCallGANServiceConnectionError(t *testing.T) {
	s := &trainingServer{ganURL: "http://127.0.0.1:1/gan/generate"}
	resp, err := s.callGANService(1, userProfile{Age: 30})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gan request failed")
	assert.Nil(t, resp)
}

func TestLoadUserProfileDefaults(t *testing.T) {
	s := &trainingServer{db: nil}
	prof := s.loadUserProfile(context.Background(), "nonexistent_user")

	assert.Equal(t, 30.0, prof.Age)
	assert.Equal(t, 1.0, prof.Gender)
	assert.Equal(t, 75.0, prof.Weight)
	assert.Equal(t, 180.0, prof.Height)
	assert.Equal(t, 1.0, prof.Goal)
}
