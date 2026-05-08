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

	// Test Redis TTL
	assert.Equal(t, 3600, RedisTTLSeconds)

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

	// Test that Redis TTL is reasonable (between 1 minute and 1 day)
	assert.GreaterOrEqual(t, RedisTTLSeconds, 60)
	assert.LessOrEqual(t, RedisTTLSeconds, 86400)

	// Test that JWT expiration is reasonable (between 1 hour and 1 week)
	assert.GreaterOrEqual(t, JWTExpirationHours, 1)
	assert.LessOrEqual(t, JWTExpirationHours, 168)
}

func TestConstantTypes(t *testing.T) {
	// Verify types are correct
	timeout := DefaultTimeout
	batchSize := MaxBatchSize
	ttl := RedisTTLSeconds
	jwtHours := JWTExpirationHours
	minHR := MinHeartRate
	maxHR := MaxHeartRate
	minSpO2 := MinSpO2
	maxSpO2 := MaxSpO2
	header := CorrelationIDHeader

	// Use variables to avoid unused variable errors
	_ = timeout
	_ = batchSize
	_ = ttl
	_ = jwtHours
	_ = minHR
	_ = maxHR
	_ = minSpO2
	_ = maxSpO2
	_ = header
	assert.IsType(t, "", header)
}
