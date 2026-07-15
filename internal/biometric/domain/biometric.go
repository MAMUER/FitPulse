// Package domain defines biometric domain models.
package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ==========================================
// Common Interfaces (Hexagonal Architecture Ports)
// ==========================================

// BiometricSource defines the interface for fetching biometric data from external sources.
// This is the primary port (inbound) that adapters must implement.
type BiometricSource interface {
	Fetch(ctx context.Context, userID string, metricTypes []string) ([]BiometricSample, error)
	Supports(metricType string) bool
	DeviceType() string
	HealthCheck(ctx context.Context) error
}

// BiometricSink defines the interface for persisting biometric data.
type BiometricSink interface {
	Store(ctx context.Context, samples []BiometricSample) error
	BatchStore(ctx context.Context, userID string, samples []BiometricSample) error
}

// ==========================================
// Domain Models
// ==========================================

// BiometricSample represents a single biometric measurement.
type BiometricSample struct {
	UserID     string                 `json:"user_id"`
	DeviceID   string                 `json:"device_id"`
	DeviceType string                 `json:"device_type"`
	MetricType string                 `json:"metric_type"`
	Value      float64                `json:"value"`
	Unit       string                 `json:"unit"`
	Timestamp  time.Time              `json:"timestamp"`
	Quality    string                 `json:"quality"`
	Confidence float64                `json:"confidence"`
	SourceID   string                 `json:"source_id"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// MetricType defines supported biometric metrics.
type MetricType string

const (
	MetricHeartRate        MetricType = "heart_rate"
	MetricHRV              MetricType = "hrv"
	MetricSpO2             MetricType = "spo2"
	MetricTemperature      MetricType = "temperature"
	MetricBloodPressureSys MetricType = "blood_pressure_systolic"
	MetricBloodPressureDia MetricType = "blood_pressure_diastolic"
	MetricECG              MetricType = "ecg"
	MetricSleepStage       MetricType = "sleep_stage"
	MetricSteps            MetricType = "steps"
	MetricDistance         MetricType = "distance"
	MetricCalories         MetricType = "calories"
	MetricRespiratoryRate  MetricType = "respiratory_rate"
	MetricBloodGlucose     MetricType = "blood_glucose"
	MetricOxygenSaturation MetricType = "oxygen_saturation"
)

func AllMetricTypes() []MetricType {
	return []MetricType{
		MetricHeartRate, MetricHRV, MetricSpO2, MetricTemperature,
		MetricBloodPressureSys, MetricBloodPressureDia, MetricECG,
		MetricSleepStage, MetricSteps, MetricDistance, MetricCalories,
		MetricRespiratoryRate, MetricBloodGlucose, MetricOxygenSaturation,
	}
}

// VendorCapabilities returns supported metrics for each device type.
func VendorCapabilities() map[string]map[MetricType]bool {
	return map[string]map[MetricType]bool{
		"fitbit": {
			MetricHeartRate: true, MetricHRV: true, MetricSpO2: true,
			MetricTemperature: false, MetricBloodPressureSys: false,
			MetricECG: false, MetricSleepStage: true, MetricSteps: true,
		},
		"garmin": {
			MetricHeartRate: true, MetricHRV: true, MetricSpO2: true,
			MetricTemperature: true, MetricBloodPressureSys: false,
			MetricECG: false, MetricSleepStage: true, MetricSteps: true,
		},
		"withings": {
			MetricHeartRate: true, MetricHRV: true, MetricSpO2: true,
			MetricTemperature: true, MetricBloodPressureSys: true,
			MetricECG: false, MetricSleepStage: true, MetricSteps: true,
		},
	}
}

// StandardMedicalCodes returns ICD-10 code reference data.
func StandardMedicalCodes() map[string]struct {
	Label    string
	Category string
	Severity string
} {
	return map[string]struct {
		Label    string
		Category string
		Severity string
	}{
		"I10": {"Essential hypertension", "cardiovascular", "moderate"},
		"I20": {"Angina pectoris", "cardiovascular", "moderate"},
		"I21": {"Acute myocardial infarction", "cardiovascular", "severe"},
		"I25": {"Chronic ischemic heart disease", "cardiovascular", "moderate"},
		"I50": {"Heart failure", "cardiovascular", "severe"},
		"I63": {"Cerebral infarction", "neurological", "severe"},
		"G40": {"Epilepsy", "neurological", "moderate"},
		"J45": {"Asthma", "respiratory", "moderate"},
		"E10": {"Type 1 diabetes", "endocrine", "moderate"},
		"E11": {"Type 2 diabetes", "endocrine", "moderate"},
		"M17": {"Knee osteoarthritis", "musculoskeletal", "mild"},
		"M19": {"Other osteoarthritis", "musculoskeletal", "mild"},
		"R26": {"Abnormalities of gait", "musculoskeletal", "mild"},
	}
}

// MedicalConstraint represents a medical condition.
type MedicalConstraint struct {
	ID               string       `json:"id"`
	Code             string       `json:"code"`
	Label            string       `json:"label"`
	Category         string       `json:"category"`
	Severity         string       `json:"severity"`
	CustomText       string       `json:"custom_text,omitempty"`
	ImpactOnTraining []ImpactRule `json:"impact_on_training"`
	ValidatedBy      *string      `json:"validated_by,omitempty"`
	ValidatedAt      *time.Time   `json:"validated_at,omitempty"`
	Active           bool         `json:"active"`
}

// ImpactRule defines how a condition affects training.
type ImpactRule struct {
	Metric       string   `json:"metric"`
	ThresholdMin *float64 `json:"threshold_min,omitempty"`
	ThresholdMax *float64 `json:"threshold_max,omitempty"`
	Action       string   `json:"action"`
	Description  string   `json:"description"`
}

// GetStandardCode returns condition info for ICD-10 code.
func GetStandardCode(code string) (struct{ Label, Category, Severity string }, bool) {
	info, ok := StandardMedicalCodes()[code]
	return info, ok
}

// WorkoutPlan represents a training session.
type WorkoutPlan struct {
	ID                     string     `json:"id"`
	UserID                 string     `json:"user_id"`
	Name                   string     `json:"name"`
	Type                   string     `json:"type"`
	DurationMinutes        int        `json:"duration_minutes"`
	TargetHeartRate        int        `json:"target_heart_rate"`
	MaxHeartRate           int        `json:"max_heart_rate"`
	Intensity              string     `json:"intensity"`
	BloodPressureSystolic  int        `json:"blood_pressure_systolic"`
	BloodPressureDiastolic int        `json:"blood_pressure_diastolic"`
	Exercises              []Exercise `json:"exercises"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// Exercise defines an exercise.
type Exercise struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Sets        int     `json:"sets"`
	Reps        int     `json:"reps"`
	Weight      float64 `json:"weight,omitempty"`
	RestSeconds int     `json:"rest_seconds"`
	Notes       string  `json:"notes,omitempty"`
}

// ConstraintViolation indicates a constraint breach.
type ConstraintViolation struct {
	ConstraintID string  `json:"constraint_id"`
	Type         string  `json:"type"`
	Metric       string  `json:"metric,omitempty"`
	ActualValue  float64 `json:"actual_value"`
	Threshold    float64 `json:"threshold,omitempty"`
	Description  string  `json:"description"`
	Action       string  `json:"action"`
}

// ModificationSuggestion suggests workout adjustments.
type ModificationSuggestion struct {
	ConstraintID   string  `json:"constraint_id"`
	Type           string  `json:"type"`
	Metric         string  `json:"metric"`
	CurrentValue   float64 `json:"current_value"`
	SuggestedValue float64 `json:"suggested_value"`
	Reason         string  `json:"reason"`
	Priority       int     `json:"priority"`
}

func (c *MedicalConstraint) Validate() error {
	if c.Code == "" {
		return errors.New("code is required")
	}
	if c.Label == "" {
		return errors.New("label is required")
	}
	if c.Category == "" {
		return errors.New("category is required")
	}
	if len(c.ImpactOnTraining) == 0 {
		return errors.New("impact rules required")
	}
	for _, r := range c.ImpactOnTraining {
		if r.Metric == "" {
			return errors.New("metric required")
		}
		if r.Action == "" {
			return errors.New("action required")
		}
		switch r.Action {
		case "modify", "avoid", "caution", "require_approval":
		default:
			return fmt.Errorf("invalid action: %s", r.Action)
		}
	}
	return nil
}

var (
	ErrSourceUnavailable  = errors.New("source unavailable")
	ErrPartialData        = errors.New("partial data")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrRateLimited        = errors.New("rate limited")
	ErrDeviceOffline      = errors.New("device offline")
	ErrUnsupportedMetric  = errors.New("unsupported metric")
)
