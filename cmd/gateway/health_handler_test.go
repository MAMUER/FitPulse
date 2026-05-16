package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler(t *testing.T) {
	g := &gateway{
		mlClassifierURL: "http://localhost:8001",
		mlGeneratorURL:  "http://localhost:8002",
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	g.healthHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, "gateway", response["service"])
	assert.NotEmpty(t, response["timestamp"])
}
