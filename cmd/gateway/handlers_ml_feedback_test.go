package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/logger"
)

func newFeedbackGateway(t *testing.T, handler http.HandlerFunc) *gateway {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &gateway{log: logger.New("test"), classifierURL: srv.URL}
}

func postFeedback(t *testing.T, g *gateway, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ml/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	g.handleMLFeedback(rec, req)
	return rec
}

func TestHandleMLFeedback_Success(t *testing.T) {
	g := newFeedbackGateway(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok", "applied": true, "target_used": 1.0,
		})
	})

	rec := postFeedback(t, g, `{"prediction_id":"p1","feedback":"correct","user_id":"u1","class_index":2}`)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp mlFeedbackResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.DeltaApplied)
	assert.NotEmpty(t, resp.FeedbackID)
}

func TestHandleMLFeedback_ClassifierError(t *testing.T) {
	g := newFeedbackGateway(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	rec := postFeedback(t, g, `{"prediction_id":"p1","feedback":"correct","user_id":"u1","class_index":2}`)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp mlFeedbackResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.False(t, resp.DeltaApplied)
	assert.NotEmpty(t, resp.DeltaReason)
}

func TestHandleMLFeedback_NotApplied(t *testing.T) {
	g := newFeedbackGateway(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"applied": false})
	})

	rec := postFeedback(t, g, `{"prediction_id":"p1","feedback":"correct","user_id":"u1","class_index":2}`)
	var resp mlFeedbackResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.False(t, resp.DeltaApplied)
}

func TestHandleMLFeedback_BadRequests(t *testing.T) {
	g := newFeedbackGateway(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"applied": true})
	})

	cases := []struct {
		name string
		body string
	}{
		{"invalid json", `{`},
		{"no prediction", `{"feedback":"x"}`},
		{"no feedback", `{"prediction_id":"p1"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := postFeedback(t, g, tc.body)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestHandleMLFeedback_NoUser(t *testing.T) {
	g := newFeedbackGateway(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"applied": true})
	})

	rec := postFeedback(t, g, `{"prediction_id":"p1","feedback":"correct"}`)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp mlFeedbackResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.False(t, resp.DeltaApplied)
	assert.Contains(t, resp.DeltaReason, "user_id")
}

func TestHandleMLFeedback_NoClass(t *testing.T) {
	g := newFeedbackGateway(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"applied": true})
	})

	rec := postFeedback(t, g, `{"prediction_id":"p1","feedback":"correct","user_id":"u1"}`)
	var resp mlFeedbackResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.False(t, resp.DeltaApplied)
	assert.Contains(t, resp.DeltaReason, "class")
}

func TestHandleMLFeedback_MethodNotAllowed(t *testing.T) {
	g := &gateway{log: logger.New("test")}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ml/feedback", nil)
	rec := httptest.NewRecorder()
	g.handleMLFeedback(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestResolveFeedbackClassIndex(t *testing.T) {
	g := &gateway{}
	idx := 3
	req := &mlFeedbackRequest{ClassIndex: &idx}
	got, ok := g.resolveFeedbackClassIndex(req)
	assert.True(t, ok)
	assert.Equal(t, 3, got)

	req = &mlFeedbackRequest{ClassifierState: "power_hiit"}
	got, ok = g.resolveFeedbackClassIndex(req)
	assert.True(t, ok)
	assert.Equal(t, 3, got)

	req = &mlFeedbackRequest{Metadata: map[string]interface{}{"predicted_class": "recovery"}}
	got, ok = g.resolveFeedbackClassIndex(req)
	assert.True(t, ok)
	assert.Equal(t, 0, got)

	req = &mlFeedbackRequest{}
	_, ok = g.resolveFeedbackClassIndex(req)
	assert.False(t, ok)
}
