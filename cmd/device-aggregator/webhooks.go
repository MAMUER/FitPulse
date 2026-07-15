package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
)

func fitbitWebhookHandler(w http.ResponseWriter, r *http.Request) {
	log := logger.New("device-aggregator-webhook")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := readBody(r)
	if err != nil {
		log.Error("Failed to read webhook body", zap.Error(err))
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	var notification map[string]interface{}
	if err := json.Unmarshal(body, &notification); err != nil {
		log.Error("Failed to parse webhook JSON", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	notificationType, _ := notification["type"].(string)
	userID, _ := notification["user_id"].(string)

	log.Info("Fitbit webhook received",
		zap.String("type", notificationType),
		zap.String("user_id", userID),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "accepted"}); err != nil {
		log.Warn("failed to write webhook response", zap.Error(err))
	}
}

func withingsWebhookHandler(w http.ResponseWriter, r *http.Request) {
	log := logger.New("device-aggregator-webhook")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	signature := r.Header.Get("X-Withings-Signature")
	if signature == "" {
		log.Warn("missing Withings signature")
		http.Error(w, "Missing signature", http.StatusBadRequest)
		return
	}

	body, err := readBody(r)
	if err != nil {
		log.Error("Failed to read webhook body", zap.Error(err))
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	if !verifyWithingsSignature(signature, body) {
		log.Warn("invalid Withings signature")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	var notification map[string]interface{}
	if err := json.Unmarshal(body, &notification); err != nil {
		log.Error("Failed to parse webhook JSON", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	action, _ := notification["action"].(string)
	log.Info("Withings webhook received",
		zap.String("action", action),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "accepted"}); err != nil {
		log.Warn("failed to write webhook response", zap.Error(err))
	}
}

func readBody(r *http.Request) ([]byte, error) {
	defer func() { _ = r.Body.Close() }()
	return io.ReadAll(r.Body)
}

func verifyWithingsSignature(signature string, body []byte) bool {
	secret := getWithingsWebhookSecret()
	if secret == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

func getWithingsWebhookSecret() string {
	return ""
}

func init() {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			subscribeFitbitWebhooks()
			subscribeWithingsWebhooks()
		}
	}()
}

func subscribeFitbitWebhooks() {
}

func subscribeWithingsWebhooks() {
}
