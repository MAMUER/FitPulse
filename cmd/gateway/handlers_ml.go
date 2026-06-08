package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/google/uuid"
)

func (g *gateway) mlClassifyHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

	// Запрашиваем все необходимые метрики для ML-классификации
	metricTypes := []string{"heart_rate", "hrv", "spo2", "temperature", "systolic_pressure", "diastolic_pressure", "sleep_hours"}
	metrics := make(map[string]*biometricpb.BiometricRecord)

	for _, metricType := range metricTypes {
		client, err := g.getBiometricClient()
		if err != nil {
			http.Error(w, "Сервис биометрии временно недоступен", http.StatusServiceUnavailable)
			return
		}

		bioResp, err := client.GetLatest(r.Context(), &biometricpb.GetLatestRequest{
			UserId:     userID,
			MetricType: metricType,
		})
		if err != nil {
			g.log.Debug("Failed to get metric", zap.String("metric", metricType), zap.Error(err))
			// Продолжаем с nil - будут использованы дефолтные значения
		} else {
			metrics[metricType] = bioResp
		}
	}

	// Агрегируем все метрики в один payload
	mlPayload := aggregateMLPayload(metrics)

	if g.mlAsync {
		g.handleAsyncClassify(w, r, mlPayload)
		return
	}

	reqBody, _ := json.Marshal(mlPayload)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !isValidServiceURL(g.mlClassifierURL, "http://localhost:", "http://ml-", "http://ml-classifier:", "http://classifier:") {
		g.log.Error("Invalid ML classifier URL", zap.String("url", g.mlClassifierURL))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		g.mlClassifierURL+"/classify",
		bytes.NewReader(reqBody))
	if err != nil {
		g.log.Error("Failed to create ML classifier request", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-ID", middleware.GetCorrelationID(ctx))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		g.log.Error("ML classifier request failed", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			g.log.Error("Failed to close response body", zap.Error(closeErr))
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		g.log.Error("Failed to read classifier response", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	if resp.StatusCode != http.StatusOK {
		g.log.Error("ML classifier returned error", zap.Int("status", resp.StatusCode))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		g.log.Error("Failed to parse classifier response", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}

}

// aggregateMLPayload aggregates biometric metrics into a payload for ML service
func aggregateMLPayload(metrics map[string]*biometricpb.BiometricRecord) map[string]interface{} {
	physiologicalData := make(map[string]interface{})
	for metricType, record := range metrics {
		if record != nil {
			physiologicalData[metricType] = record.Value
		} else {
			physiologicalData[metricType] = nil
		}
	}
	return map[string]interface{}{
		"physiological_data": physiologicalData,
	}
}

func (g *gateway) handleAsyncClassify(w http.ResponseWriter, r *http.Request, mlPayload map[string]interface{}) {
	jobID := uuid.New().String()
	correlationID := middleware.GetCorrelationID(r.Context())

	body, err := json.Marshal(map[string]interface{}{
		"job_id":             jobID,
		"physiological_data": mlPayload["physiological_data"],
	})
	if err != nil {
		g.log.Error("Failed to marshal classify job", zap.Error(err))
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err = g.rmqCh.PublishWithContext(ctx, "", "ml.classify", false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Headers: amqp.Table{
				"x-correlation-id": correlationID,
				"x-job-id":         jobID,
			},
		})
	if err != nil {
		g.log.Error("Failed to publish classify job to RabbitMQ", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id": jobID,
		"status": "pending",
	}); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

func (g *gateway) mlGenerateHandler(w http.ResponseWriter, r *http.Request) {

	if !isValidServiceURL(g.mlGeneratorURL, "http://localhost:", "http://ml-", "http://ml-generator:", "http://generator:") {
		g.log.Error("Invalid ML generator URL", zap.String("url", g.mlGeneratorURL))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		g.log.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Ошибка чтения запроса", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST",
		g.mlGeneratorURL+"/generate",
		bytes.NewReader(body))
	if err != nil {
		g.log.Error("Failed to create ML generator request", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-ID", middleware.GetCorrelationID(ctx))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		g.log.Error("ML generator request failed", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			g.log.Error("Failed to close response body", zap.Error(closeErr))
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		g.log.Error("Failed to read generator response", zap.Error(err))
		http.Error(w, "ML-сервис временно недоступен", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(respBody); err != nil {
		g.log.Error("Failed to write response", zap.Error(err))
	}
}
