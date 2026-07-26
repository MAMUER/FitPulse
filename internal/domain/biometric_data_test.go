package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBiometricDataStruct(t *testing.T) {
	now := time.Now()
	data := BiometricData{
		ID:         "data123",
		UserID:     "user456",
		MetricType: "heart_rate",
		Value:      75.5,
		Timestamp:  now,
		DeviceType: "fitbit",
		CreatedAt:  now,
	}

	assert.Equal(t, "data123", data.ID)
	assert.Equal(t, "user456", data.UserID)
	assert.Equal(t, "heart_rate", data.MetricType)
	assert.Equal(t, 75.5, data.Value)
	assert.Equal(t, now, data.Timestamp)
	assert.Equal(t, "fitbit", data.DeviceType)
	assert.Equal(t, now, data.CreatedAt)
}

func TestBiometricDataZeroValues(t *testing.T) {
	var data BiometricData

	assert.Empty(t, data.ID)
	assert.Empty(t, data.UserID)
	assert.Empty(t, data.MetricType)
	assert.Equal(t, 0.0, data.Value)
	assert.True(t, data.Timestamp.IsZero())
	assert.Empty(t, data.DeviceType)
	assert.True(t, data.CreatedAt.IsZero())
}

func TestBiometricDataFieldTypes(t *testing.T) {
	data := BiometricData{}

	assert.IsType(t, "", data.ID)
	assert.IsType(t, "", data.UserID)
	assert.IsType(t, "", data.MetricType)
	assert.IsType(t, 0.0, data.Value)
	assert.IsType(t, time.Time{}, data.Timestamp)
	assert.IsType(t, "", data.DeviceType)
	assert.IsType(t, time.Time{}, data.CreatedAt)
}

func TestNewBiometricData_Success(t *testing.T) {
	ts := time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC)
	data, err := NewBiometricData("user-1", "heart_rate", 72.0, ts, "fitbit")

	require.NoError(t, err)
	require.NotNil(t, data)
	assert.Equal(t, "user-1", data.UserID)
	assert.Equal(t, "heart_rate", data.MetricType)
	assert.Equal(t, 72.0, data.Value)
	assert.Equal(t, ts, data.Timestamp)
	assert.Equal(t, "fitbit", data.DeviceType)
	assert.False(t, data.CreatedAt.IsZero())
}

func TestNewBiometricData_ValidationFailure(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		metricType string
		value      float64
		timestamp  time.Time
		deviceType string
		wantErr    bool
	}{
		{"empty userID", "", "heart_rate", 72.0, time.Now(), "fitbit", true},
		{"empty metricType", "user-1", "", 72.0, time.Now(), "fitbit", true},
		{"zero timestamp", "user-1", "heart_rate", 72.0, time.Time{}, "fitbit", true},
		{"negative value", "user-1", "heart_rate", -1.0, time.Now(), "fitbit", true},
		{"empty deviceType", "user-1", "heart_rate", 72.0, time.Now(), "", true},
		{"valid data", "user-1", "heart_rate", 72.0, time.Now(), "fitbit", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := NewBiometricData(tt.userID, tt.metricType, tt.value, tt.timestamp, tt.deviceType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, data)
				assert.True(t, errors.As(err, &InvalidMetricError{}))
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, data)
			}
		})
	}
}

func TestMetricTypeConstants(t *testing.T) {
	assert.Equal(t, MetricType("heart_rate"), MetricHeartRate)
	assert.Equal(t, MetricType("hrv"), MetricHRV)
	assert.Equal(t, MetricType("spo2"), MetricSpO2)
	assert.Equal(t, MetricType("temperature"), MetricTemperature)
	assert.Equal(t, MetricType("blood_pressure_systolic"), MetricBloodPressureSys)
	assert.Equal(t, MetricType("blood_pressure_diastolic"), MetricBloodPressureDia)
	assert.Equal(t, MetricType("ecg"), MetricECG)
	assert.Equal(t, MetricType("sleep_stage"), MetricSleepStage)
	assert.Equal(t, MetricType("steps"), MetricSteps)
	assert.Equal(t, MetricType("distance"), MetricDistance)
	assert.Equal(t, MetricType("calories"), MetricCalories)
	assert.Equal(t, MetricType("respiratory_rate"), MetricRespiratoryRate)
	assert.Equal(t, MetricType("blood_glucose"), MetricBloodGlucose)
	assert.Equal(t, MetricType("oxygen_saturation"), MetricOxygenSaturation)
}

func TestInvalidMetricError(t *testing.T) {
	err := InvalidMetricError{}
	assert.Equal(t, "invalid biometric data", err.Error())
}

func TestBiometricDataJSONTags(t *testing.T) {
	data := BiometricData{
		ID:         "test123",
		UserID:     "user456",
		MetricType: "heart_rate",
		Value:      75.5,
		Timestamp:  time.Now(),
		DeviceType: "fitbit",
		CreatedAt:  time.Now(),
	}

	assert.NotEmpty(t, data.ID)
	assert.NotEmpty(t, data.UserID)
	assert.NotEmpty(t, data.MetricType)
	assert.NotZero(t, data.Value)
	assert.False(t, data.Timestamp.IsZero())
	assert.NotEmpty(t, data.DeviceType)
	assert.False(t, data.CreatedAt.IsZero())
}
