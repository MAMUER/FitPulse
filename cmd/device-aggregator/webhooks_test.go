package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenWearablesWebhookHandler(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/open-wearables/webhook", nil)
		openWearablesWebhookHandler(w, r)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/open-wearables/webhook", bytes.NewReader([]byte("not-json")))
		openWearablesWebhookHandler(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid webhook", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := map[string]interface{}{
			"user_id": "user-1",
			"source":  "open_wearables",
			"metrics": []map[string]interface{}{{"metric_type": "heart_rate", "value": 72.0}},
		}
		body, _ := json.Marshal(payload)
		r := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/open-wearables/webhook", bytes.NewReader(body))
		openWearablesWebhookHandler(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		assert.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.Equal(t, "accepted", resp["status"])
	})
}

func TestHandleAggregatorWebhookSignature(t *testing.T) {
	payload := map[string]interface{}{
		"user_id": "user-1",
		"source":  "open_wearables",
		"metrics": []map[string]interface{}{{"metric_type": "heart_rate", "value": 72.0}},
	}
	body, _ := json.Marshal(payload)

	t.Run("valid payload is accepted", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		handleAggregatorWebhook(w, r, "open_wearables", func(n map[string]interface{}) map[string]string {
			return map[string]string{"user_id": "user-1", "source": "open_wearables"}
		})
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestReadBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("hello")))
	body, err := readBody(r)
	assert.NoError(t, err)
	assert.Equal(t, []byte("hello"), body)
}
