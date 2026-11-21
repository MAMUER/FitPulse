package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/sanitize"
)

// biometricWebhookPath — путь приёма вебхуков в biometric-service.
const biometricWebhookPath = "/api/v1/integrations/open-wearables/webhook"

func openWearablesWebhookHandler(w http.ResponseWriter, r *http.Request) {
	handleAggregatorWebhook(w, r, func(notification map[string]interface{}) map[string]string {
		fields := map[string]string{}
		if userID, ok := notification["user_id"].(string); ok {
			fields["user_id"] = userID
		}
		if source, ok := notification["source"].(string); ok {
			fields["source"] = source
		}
		return fields
	})
}

func handleAggregatorWebhook(w http.ResponseWriter, r *http.Request, extractFields func(map[string]interface{}) map[string]string) {
	const source = "open_wearables"
	log := logger.New("device-aggregator-webhook")

	if r.Method != http.MethodPost {
		log.Error("Method not allowed", zap.String("method", sanitize.LogString(r.Method)), zap.String("path", sanitize.LogString(r.URL.Path)))
		webhookRequestsTotal.WithLabelValues(source, "method_not_allowed").Inc()
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := readBody(r)
	if err != nil {
		log.Error("Failed to read webhook body", zap.Error(err))
		webhookRequestsTotal.WithLabelValues(source, "bad_request").Inc()
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	var notification map[string]interface{}
	if err := json.Unmarshal(body, &notification); err != nil {
		log.Error("Failed to parse webhook JSON", zap.Error(err))
		webhookRequestsTotal.WithLabelValues(source, "bad_request").Inc()
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	fields := extractFields(notification)
	fields["source"] = source

	log.Info("Webhook received",
		zap.String("source", sanitize.LogString(source)),
		zap.Any("fields", sanitize.MapStringString(fields)),
	)

	// Агрегатор не должен быть «чёрной дырой»: пересылаем payload в
	// biometric-service, который сохраняет метрики в biometric_data.
	forwarded, forwardErr := forwardToBiometricService(r.Context(), body)
	if forwardErr != nil {
		log.Warn("Не удалось переслать вебхук в biometric-service", zap.Error(forwardErr))
		webhookRequestsTotal.WithLabelValues(source, "forward_failed").Inc()
	} else {
		log.Info("Webhook forwarded", zap.String("status", sanitize.LogString(forwarded)))
	}

	webhookRequestsTotal.WithLabelValues(source, "success").Inc()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "accepted",
		"forwarded": forwardErr == nil,
	}); err != nil {
		log.Warn("failed to write webhook response", zap.Error(err))
	}
}

// forwardToBiometricService отправляет исходный payload в biometric-service,
// подписывая его тем же HMAC-секретом, что и внешний провайдер.
func forwardToBiometricService(ctx context.Context, body []byte) (string, error) {
	target := strings.TrimRight(os.Getenv("BIOMETRIC_SERVICE_URL"), "/")
	if target == "" {
		target = "http://biometric-service:8085"
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := signPayload([]byte(os.Getenv("OPEN_WEARABLES_WEBHOOK_SECRET")), timestamp, body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target+biometricWebhookPath, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create forward request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", "sha256="+signature)
	req.Header.Set("X-Webhook-Timestamp", timestamp)
	req.Header.Set("X-Webhook-Id", "device-aggregator-"+timestamp)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("forward request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("biometric-service returned status %d", resp.StatusCode)
	}
	return resp.Status, nil
}

// signPayload — HMAC-SHA256 над строкой timestamp + "." + body.
func signPayload(secret []byte, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func readBody(r *http.Request) ([]byte, error) {
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			log := logger.New("device-aggregator-webhook")
			log.Warn("Failed to close webhook request body", zap.Error(closeErr))
		}
	}()
	return io.ReadAll(r.Body)
}
