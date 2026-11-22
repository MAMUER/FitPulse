package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/planner"
)

func TestResolveClassification_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/classify", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"predicted_class": "power_hiit",
			"confidence":      0.93,
			"data_quality":    "high",
		})
	}))
	defer srv.Close()

	s := &trainingServer{log: logger.New("test"), classifierURL: srv.URL}
	got := s.resolveClassification(context.Background(), "u1", &planner.Biometrics{
		HeartRate: 185, HRV: 18, Spo2: 95, Temperature: 37.2,
	})
	assert.Equal(t, "power_hiit", got)
}

func TestResolveClassification_Fallback(t *testing.T) {
	t.Run("no url", func(t *testing.T) {
		s := &trainingServer{}
		assert.Equal(t, "endurance_basic", s.resolveClassification(context.Background(), "u1", &planner.Biometrics{}))
	})

	t.Run("no biometrics", func(t *testing.T) {
		s := &trainingServer{classifierURL: "http://127.0.0.1:1"}
		assert.Equal(t, "endurance_basic", s.resolveClassification(context.Background(), "u1", nil))
	})

	t.Run("unknown class name", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"predicted_class": "garbage"})
		}))
		defer srv.Close()

		s := &trainingServer{log: logger.New("test"), classifierURL: srv.URL}
		assert.Equal(t, "endurance_basic", s.resolveClassification(context.Background(), "u1", &planner.Biometrics{}))
	})

	t.Run("service error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		s := &trainingServer{log: logger.New("test"), classifierURL: srv.URL}
		assert.Equal(t, "endurance_basic", s.resolveClassification(context.Background(), "u1", &planner.Biometrics{}))
	})
}
