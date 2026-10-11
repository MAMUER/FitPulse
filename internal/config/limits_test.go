package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	// Test timeout constants
	assert.Equal(t, 5*time.Second, DefaultTimeout)

	// Test batch and size limits
	assert.Equal(t, 100, MaxBatchSize)

	// Test Valkey TTL
	assert.Equal(t, 3600, ValkeyTTLSeconds)

	// Test JWT expiration
	assert.Equal(t, 24, JWTExpirationHours)

	// Test heart rate limits
	assert.Equal(t, 30, MinHeartRate)
	assert.Equal(t, 220, MaxHeartRate)

	// Test SpO2 limits
	assert.Equal(t, 70, MinSpO2)
	assert.Equal(t, 100, MaxSpO2)

	// Test header constant
	assert.Equal(t, "X-Correlation-ID", CorrelationIDHeader)
}

func TestConstantRelationships(t *testing.T) {
	// Test that min values are less than max values
	assert.Less(t, MinHeartRate, MaxHeartRate)
	assert.Less(t, MinSpO2, MaxSpO2)

	// Test that timeout is reasonable (between 1 second and 1 minute)
	assert.GreaterOrEqual(t, DefaultTimeout, 1*time.Second)
	assert.LessOrEqual(t, DefaultTimeout, 1*time.Minute)

	// Test that batch size is reasonable
	assert.Greater(t, MaxBatchSize, 0)
	assert.LessOrEqual(t, MaxBatchSize, 1000) // Reasonable upper bound

	// Test that Valkey TTL is reasonable (between 1 minute and 1 day)
	assert.GreaterOrEqual(t, ValkeyTTLSeconds, 60)
	assert.LessOrEqual(t, ValkeyTTLSeconds, 86400)

	// Test that JWT expiration is reasonable (between 1 hour and 1 week)
	assert.GreaterOrEqual(t, JWTExpirationHours, 1)
	assert.LessOrEqual(t, JWTExpirationHours, 168)
}

func TestConstantTypes(t *testing.T) {
	// Verify types are correct
	assert.IsType(t, time.Duration(0), DefaultTimeout)
	assert.IsType(t, 0, MaxBatchSize)
	assert.IsType(t, 0, ValkeyTTLSeconds)
	assert.IsType(t, 0, JWTExpirationHours)
	assert.IsType(t, 0, MinHeartRate)
	assert.IsType(t, 0, MaxHeartRate)
	assert.IsType(t, 0, MinSpO2)
	assert.IsType(t, 0, MaxSpO2)
	assert.IsType(t, "", CorrelationIDHeader)
}
