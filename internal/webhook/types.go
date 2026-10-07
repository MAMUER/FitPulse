// Package webhook provides Open Wearables webhook handling.
package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/sanitize"
)

var (
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrInvalidPayload    = errors.New("invalid payload")
	ErrEmptyUserID       = errors.New("user_id is required")
	ErrEmptyMetrics      = errors.New("metrics array is required")
	ErrInvalidMetricType = errors.New("invalid metric_type")
	ErrInvalidSourceType = errors.New("invalid source type")
	ErrMissingTimestamp  = errors.New("timestamp is required")
)

type MetricType string

const (
	MetricTypeWeight           MetricType = "weight"
	MetricTypeHeartRate        MetricType = "heart_rate"
	MetricTypeRestingHeartRate MetricType = "resting_heart_rate"
	MetricTypeSpO2             MetricType = "spo2"
	MetricTypeSleep            MetricType = "sleep"
	MetricTypeSleepStages      MetricType = "sleep_stages"
	MetricTypeSteps            MetricType = "steps"
	MetricTypeMenstrualCycle   MetricType = "menstrual_cycle"
	MetricTypeBodyComposition  MetricType = "body_composition"
	MetricTypeHRV              MetricType = "hrv"
	MetricTypeHRVSDNN          MetricType = "hrv_sdnn"
	MetricTypeHRVRMSSD         MetricType = "hrv_rmssd"
	MetricTypeTemperature      MetricType = "temperature"
	MetricTypeBloodPressure    MetricType = "blood_pressure"
	MetricTypeBloodGlucose     MetricType = "blood_glucose"
	MetricTypeActiveEnergy     MetricType = "active_energy"
	MetricTypeBasalEnergy      MetricType = "basal_energy"
	MetricTypeBodyTemperature  MetricType = "body_temperature"
	MetricTypeBMI              MetricType = "bmi"
	MetricTypeLeanBodyMass     MetricType = "lean_body_mass"
	MetricTypeFlightsClimbed   MetricType = "flights_climbed"
	MetricTypeWaterIntake      MetricType = "water_intake"
	MetricTypeVO2Max           MetricType = "vo2_max"
	MetricTypeRespiratoryRate  MetricType = "respiratory_rate"
	MetricTypeWorkout          MetricType = "workout"
)

type SourceType string

const (
	SourceTypeAppleHealth         SourceType = "apple_health"
	SourceTypeGarmin              SourceType = "garmin"
	SourceTypeHealthConnect       SourceType = "health_connect"
	SourceTypeOpenWearables       SourceType = "open_wearables"
	SourceTypeWhoop               SourceType = "whoop"
	SourceTypePolar               SourceType = "polar"
	SourceTypeSuunto              SourceType = "suunto"
	SourceTypeStrava              SourceType = "strava"
	SourceTypeOura                SourceType = "oura"
	SourceTypeUltrahuman          SourceType = "ultrahuman"
	SourceTypeSamsungHealth       SourceType = "samsung_health"
	SourceTypeGoogleHealthConnect SourceType = "google_health_connect"
)

var allowedMetricTypes = map[MetricType]struct{}{
	MetricTypeWeight:           {},
	MetricTypeHeartRate:        {},
	MetricTypeRestingHeartRate: {},
	MetricTypeSpO2:             {},
	MetricTypeSleep:            {},
	MetricTypeSleepStages:      {},
	MetricTypeSteps:            {},
	MetricTypeMenstrualCycle:   {},
	MetricTypeBodyComposition:  {},
	MetricTypeHRV:              {},
	MetricTypeHRVSDNN:          {},
	MetricTypeHRVRMSSD:         {},
	MetricTypeTemperature:      {},
	MetricTypeBloodPressure:    {},
	MetricTypeBloodGlucose:     {},
	MetricTypeActiveEnergy:     {},
	MetricTypeBasalEnergy:      {},
	MetricTypeBodyTemperature:  {},
	MetricTypeBMI:              {},
	MetricTypeLeanBodyMass:     {},
	MetricTypeFlightsClimbed:   {},
	MetricTypeWaterIntake:      {},
	MetricTypeVO2Max:           {},
	MetricTypeRespiratoryRate:  {},
	MetricTypeWorkout:          {},
}

var allowedSourceTypes = map[SourceType]struct{}{
	SourceTypeAppleHealth:         {},
	SourceTypeGarmin:              {},
	SourceTypeHealthConnect:       {},
	SourceTypeOpenWearables:       {},
	SourceTypeWhoop:               {},
	SourceTypePolar:               {},
	SourceTypeSuunto:              {},
	SourceTypeStrava:              {},
	SourceTypeOura:                {},
	SourceTypeUltrahuman:          {},
	SourceTypeSamsungHealth:       {},
	SourceTypeGoogleHealthConnect: {},
}

// OpenWearablesWebhookPayload represents normalized data from Open Wearables
type OpenWearablesWebhookPayload struct {
	UserID    string                `json:"user_id"`
	Source    SourceType            `json:"source"`
	Timestamp time.Time             `json:"timestamp"`
	Nonce     string                `json:"nonce,omitempty"`
	Metrics   []OpenWearablesMetric `json:"metrics"`
}

// OpenWearablesMetric represents a single biometric metric
type OpenWearablesMetric struct {
	Type      MetricType `json:"metric_type"`
	Value     float64    `json:"value"`
	Unit      string     `json:"unit,omitempty"`
	Timestamp time.Time  `json:"timestamp,omitempty"`
}

// WebhookResponse represents the response sent back to Open Wearables
type WebhookResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Validate validates the webhook payload
func (p *OpenWearablesWebhookPayload) Validate() error {
	if p.UserID == "" {
		return ErrEmptyUserID
	}

	if len(p.Metrics) == 0 {
		return ErrEmptyMetrics
	}

	if _, ok := allowedSourceTypes[p.Source]; !ok {
		return fmt.Errorf("%w: %s", ErrInvalidSourceType, p.Source)
	}

	if p.Timestamp.IsZero() {
		return ErrMissingTimestamp
	}

	for i, metric := range p.Metrics {
		if _, ok := allowedMetricTypes[metric.Type]; !ok {
			return fmt.Errorf("%w: %s at index %d", ErrInvalidMetricType, metric.Type, i)
		}
		if metric.Timestamp.IsZero() {
			metric.Timestamp = p.Timestamp
		}
		p.Metrics[i] = metric
	}

	return nil
}

// DecodePayload decodes JSON payload into OpenWearablesWebhookPayload
func DecodePayload(body []byte) (*OpenWearablesWebhookPayload, error) {
	var payload OpenWearablesWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidPayload, err)
	}
	return &payload, nil
}

// WriteResponse writes JSON response to HTTP client
func WriteResponse(w http.ResponseWriter, statusCode int, response WebhookResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

// LogFields returns zap fields for logging
func LogFields(payload *OpenWearablesWebhookPayload) []zap.Field {
	return []zap.Field{
		zap.String("user_id", sanitize.LogString(payload.UserID)),
		zap.String("source", sanitize.LogString(string(payload.Source))),
		zap.Int("metrics_count", len(payload.Metrics)),
		zap.Time("timestamp", payload.Timestamp),
	}
}
