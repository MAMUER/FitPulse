package main

import (
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/sanitize"
)

func openWearablesWebhookHandler(w http.ResponseWriter, r *http.Request) {
	handleAggregatorWebhook(w, r, "open_wearables", func(notification map[string]interface{}) map[string]string {
		userID, _ := notification["user_id"].(string)
		source, _ := notification["source"].(string)
		return map[string]string{"user_id": userID, "source": source}
	})
}

func handleAggregatorWebhook(w http.ResponseWriter, r *http.Request, source string, extractFields func(map[string]interface{}) map[string]string) {
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

	fields := extractFields(notification)
	fields["source"] = source

	log.Info("Webhook received",
		zap.String("source", sanitize.LogString(source)),
		zap.Any("fields", fields),
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
