// Package domain provides core business domain models and types.
package domain

import (
	"time"
)

type BiometricData struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	MetricType string    `json:"metric_type"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
	DeviceType string    `json:"device_type"`
	CreatedAt  time.Time `json:"created_at"`
}

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

func NewBiometricData(userID, metricType string, value float64, timestamp time.Time, deviceType string) (*BiometricData, error) {
	data := &BiometricData{
		UserID:     userID,
		MetricType: metricType,
		Value:      value,
		Timestamp:  timestamp,
		DeviceType: deviceType,
		CreatedAt:  time.Now(),
	}
	if err := data.Validate(); err != nil {
		return nil, err
	}
	return data, nil
}

func (b *BiometricData) Validate() error {
	if b.UserID == "" {
		return InvalidMetricError{}
	}
	if b.MetricType == "" {
		return InvalidMetricError{}
	}
	if b.Timestamp.IsZero() {
		return InvalidMetricError{}
	}
	if b.Value < 0 {
		return InvalidMetricError{}
	}
	if b.DeviceType == "" {
		return InvalidMetricError{}
	}
	return nil
}

type InvalidMetricError struct{}

func (e InvalidMetricError) Error() string {
	return "invalid biometric data"
}
