package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	"github.com/MAMUER/project/internal/middleware"
)

func (g *gateway) callClassifier(ctx context.Context, payload []byte) (map[string]interface{}, error) {
	if !isValidServiceURL(g.classifierURL, "http://classifier:", "http://classifier-service:", "http://127.0.0.1:") {
		g.log.Error("Некорректный URL классификатора", zap.String("url", g.classifierURL))
		return nil, errors.New("некорректный URL классификатора")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.classifierURL+"/classify", bytes.NewReader(payload))
	if err != nil {
		g.log.Error("Failed to create classifier request", zap.Error(err))
		return nil, err
	}
	req.Header.Set(headerContentType, contentTypeJSON)
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
		key := mapMetricTypeToClassifierKey(metricType)
		if record != nil {
			physiologicalData[key] = record.Value
		} else {
			physiologicalData[key] = nil
		}
	}
	return map[string]interface{}{
		"physiological_data": physiologicalData,
	}
}

func mapMetricTypeToClassifierKey(metricType string) string {
	switch metricType {
	case "hrv":
		return "heart_rate_variability"
	case "systolic_pressure":
		return "blood_pressure_systolic"
	case "diastolic_pressure":
		return "blood_pressure_diastolic"
	default:
		return metricType
	}
}

func transformClassifierResponse(result map[string]interface{}) map[string]interface{} {
	predictedClass, _ := result["predicted_class"].(string)
	confidence, _ := result["confidence"].(float64)

	var recommendations []interface{}
	if recs, ok := result["recommendations"].([]interface{}); ok {
		recommendations = recs
	}

	fatigueLevel, motivationScore, recoveryQuality := mapClassToScores(predictedClass)

	return map[string]interface{}{
		"status":           "success",
		"state":            predictedClass,
		"confidence":       confidence,
		"recommendation":   recommendations,
		"fatigue_level":    fatigueLevel,
		"motivation_score": motivationScore,
		"recovery_quality": recoveryQuality,
	}
}

func mapClassToScores(predictedClass string) (float64, float64, float64) {
	switch predictedClass {
	case "recovery":
		return 0.1, 0.8, 0.9
	case "endurance_basic":
		return 0.3, 0.7, 0.7
	case "endurance_threshold":
		return 0.5, 0.6, 0.5
	case "power_hiit":
		return 0.7, 0.5, 0.3
	case "overtraining":
		return 0.9, 0.2, 0.1
	case "illness":
		return 1.0, 0.0, 0.0
	default:
		return 0.5, 0.5, 0.5
	}
}

