package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	"github.com/MAMUER/project/internal/middleware"
)

func (g *gateway) classifyHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "Необходима авторизация", http.StatusUnauthorized)
		return
	}

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
		} else {
			metrics[metricType] = bioResp
		}
	}

	mlPayload := aggregateMLPayload(metrics)
	reqBody, _ := json.Marshal(mlPayload)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := g.callClassifier(ctx, reqBody)
	if err != nil {
		http.Error(w, "Сервис классификации временно недоступен", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		g.log.Error("Failed to encode response", zap.Error(err))
	}
}

func (g *gateway) callClassifier(ctx context.Context, payload []byte) (map[string]interface{}, error) {
	if !isValidServiceURL(g.classifierURL, "http://localhost:", "http://classifier:", "http://classifier-service:") {
		g.log.Error("Invalid classifier URL", zap.String("url", g.classifierURL))
		return nil, errors.New("invalid classifier URL")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.classifierURL+"/classify", bytes.NewReader(payload))
	if err != nil {
		g.log.Error("Failed to create classifier request", zap.Error(err))
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-ID", middleware.GetCorrelationID(ctx))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		g.log.Error("Classifier request failed", zap.Error(err))
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			g.log.Error("Failed to close response body", zap.Error(closeErr))
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		g.log.Error("Failed to read classifier response", zap.Error(err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		g.log.Error("Classifier returned error", zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("classifier status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		g.log.Error("Failed to parse classifier response", zap.Error(err))
		return nil, err
	}

	return result, nil
}

// aggregateMLPayload aggregates biometric metrics into a payload for classifier service
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

func (g *gateway) proxyToMLGenerator(w http.ResponseWriter, r *http.Request, path string) {
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
		g.mlGeneratorURL+path,
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
	_, err = bytes.NewBuffer(respBody).WriteTo(w)
	if err != nil {
		g.log.Error("Failed to write response", zap.Error(err))
	}
}

func (g *gateway) mlGenerateHandler(w http.ResponseWriter, r *http.Request) {
	g.proxyToMLGenerator(w, r, "/generate-plan")
}

func (g *gateway) mlDietHandler(w http.ResponseWriter, r *http.Request) {
	g.proxyToMLGenerator(w, r, "/generate-diet")
}
