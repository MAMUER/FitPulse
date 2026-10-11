package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MAMUER/project/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandleMLFeedback_Success(t *testing.T) {
	g := newTestGateway()
	g.log = &logger.Logger{Logger: zap.NewNop()}

	body, err := json.Marshal(map[string]interface{}{
		"prediction_id": "pred-123",
		"feedback":      "good",
		"user_id":       "user-123",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ml/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	g.handleMLFeedback(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "success", resp["status"])
	assert.NotEmpty(t, resp["feedback_id"])
}

func TestHandleMLFeedback_MethodNotAllowed(t *testing.T) {
	g := newTestGateway()
	g.log = &logger.Logger{Logger: zap.NewNop()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ml/feedback", nil)
	w := httptest.NewRecorder()

	g.handleMLFeedback(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleMLFeedback_MissingPredictionID(t *testing.T) {
	g := newTestGateway()
	g.log = &logger.Logger{Logger: zap.NewNop()}

	body, err := json.Marshal(map[string]interface{}{
		"feedback": "good",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ml/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	g.handleMLFeedback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleMLFeedback_MissingFeedback(t *testing.T) {
	g := newTestGateway()
	g.log = &logger.Logger{Logger: zap.NewNop()}

	body, err := json.Marshal(map[string]interface{}{
		"prediction_id": "pred-123",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ml/feedback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	g.handleMLFeedback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleMLFeedback_InvalidJSON(t *testing.T) {
	g := newTestGateway()
	g.log = &logger.Logger{Logger: zap.NewNop()}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ml/feedback", strings.NewReader("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	g.handleMLFeedback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
