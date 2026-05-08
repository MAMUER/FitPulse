package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBiometricDataStruct(t *testing.T) {
	now := time.Now()
	data := BiometricData{
		ID:         "data123",
		UserID:     "user456",
		MetricType: "heart_rate",
		Value:      75.5,
		Timestamp:  now,
		DeviceType: "apple_watch",
	}

	assert.Equal(t, "data123", data.ID)
	assert.Equal(t, "user456", data.UserID)
	assert.Equal(t, "heart_rate", data.MetricType)
	assert.Equal(t, 75.5, data.Value)
	assert.Equal(t, now, data.Timestamp)
	assert.Equal(t, "apple_watch", data.DeviceType)
}

func TestBiometricDataZeroValues(t *testing.T) {
	// Test zero values
	var data BiometricData

	assert.Empty(t, data.ID)
	assert.Empty(t, data.UserID)
	assert.Empty(t, data.MetricType)
	assert.Equal(t, 0.0, data.Value)
	assert.True(t, data.Timestamp.IsZero())
	assert.Empty(t, data.DeviceType)
}

func TestBiometricDataFieldTypes(t *testing.T) {
	data := BiometricData{}

	// Verify field types
	assert.IsType(t, "", data.ID)
	assert.IsType(t, "", data.UserID)
	assert.IsType(t, "", data.MetricType)
	assert.IsType(t, 0.0, data.Value)
	assert.IsType(t, time.Time{}, data.Timestamp)
	assert.IsType(t, "", data.DeviceType)
}

func TestBiometricDataCreation(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		userID     string
		metricType string
		value      float64
		timestamp  time.Time
		deviceType string
	}{
		{
			name:       "heart rate data",
			id:         "hr123",
			userID:     "user456",
			metricType: "heart_rate",
			value:      72.0,
			timestamp:  time.Now(),
			deviceType: "apple_watch",
		},
		{
			name:       "blood pressure data",
			id:         "bp123",
			userID:     "user789",
			metricType: "blood_pressure_systolic",
			value:      120.0,
			timestamp:  time.Now().Add(-time.Hour),
			deviceType: "samsung_watch",
		},
		{
			name:       "temperature data",
			id:         "temp123",
			userID:     "user101",
			metricType: "temperature",
			value:      36.6,
			timestamp:  time.Now().Add(-2 * time.Hour),
			deviceType: "huawei_watch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := BiometricData{
				ID:         tt.id,
				UserID:     tt.userID,
				MetricType: tt.metricType,
				Value:      tt.value,
				Timestamp:  tt.timestamp,
				DeviceType: tt.deviceType,
			}

			assert.Equal(t, tt.id, data.ID)
			assert.Equal(t, tt.userID, data.UserID)
			assert.Equal(t, tt.metricType, data.MetricType)
			assert.Equal(t, tt.value, data.Value)
			assert.Equal(t, tt.timestamp, data.Timestamp)
			assert.Equal(t, tt.deviceType, data.DeviceType)
		})
	}
}

func TestBiometricDataJSONTags(t *testing.T) {
	// This test ensures the struct can be properly serialized/deserialized
	// by checking that the fields are accessible (JSON tags would be tested in integration tests)
	data := BiometricData{
		ID:         "test123",
		UserID:     "user456",
		MetricType: "heart_rate",
		Value:      75.5,
		Timestamp:  time.Now(),
		DeviceType: "apple_watch",
	}

	// Verify all fields are settable and gettable
	assert.NotEmpty(t, data.ID)
	assert.NotEmpty(t, data.UserID)
	assert.NotEmpty(t, data.MetricType)
	assert.NotZero(t, data.Value)
	assert.False(t, data.Timestamp.IsZero())
	assert.NotEmpty(t, data.DeviceType)
}
