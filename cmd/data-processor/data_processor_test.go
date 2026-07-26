package main

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseBiometricEvent(t *testing.T) {
	t.Run("valid event", func(t *testing.T) {
		body := `{"user_id":"user-1","metric_type":"heart_rate","value":140.0,"timestamp":"2024-01-01T00:00:00Z"}`
		event, err := parseBiometricEvent([]byte(body))
		assert.NoError(t, err)
		assert.Equal(t, "user-1", event.UserID)
		assert.Equal(t, "heart_rate", event.MetricType)
		assert.Equal(t, 140.0, event.Value)
	})

	t.Run("missing user_id", func(t *testing.T) {
		body := `{"metric_type":"heart_rate","value":140.0}`
		_, err := parseBiometricEvent([]byte(body))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id is empty")
	})

	t.Run("missing metric_type", func(t *testing.T) {
		body := `{"user_id":"user-1","value":140.0}`
		_, err := parseBiometricEvent([]byte(body))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metric_type is empty")
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parseBiometricEvent([]byte(`{invalid`))
		assert.Error(t, err)
	})

	t.Run("zero timestamp gets current time", func(t *testing.T) {
		before := time.Now().UTC()
		body := `{"user_id":"user-1","metric_type":"heart_rate","value":140.0,"timestamp":"0001-01-01T00:00:00Z"}`
		event, err := parseBiometricEvent([]byte(body))
		assert.NoError(t, err)
		assert.False(t, event.Timestamp.IsZero())
		assert.True(t, event.Timestamp.After(before) || event.Timestamp.Equal(before))
	})
}

func TestValidateBiometricEvent(t *testing.T) {
	t.Run("valid heart_rate", func(t *testing.T) {
		err := validateBiometricEvent(biometricEvent{UserID: "user-1", MetricType: "heart_rate", Value: 140})
		assert.NoError(t, err)
	})

	t.Run("valid spo2", func(t *testing.T) {
		err := validateBiometricEvent(biometricEvent{UserID: "user-1", MetricType: "spo2", Value: 98})
		assert.NoError(t, err)
	})

	t.Run("negative value", func(t *testing.T) {
		err := validateBiometricEvent(biometricEvent{UserID: "user-1", MetricType: "heart_rate", Value: -10})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "value cannot be negative")
	})

	t.Run("heart_rate out of range", func(t *testing.T) {
		err := validateBiometricEvent(biometricEvent{UserID: "user-1", MetricType: "heart_rate", Value: 10})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "out of valid range")
	})

	t.Run("unknown metric_type", func(t *testing.T) {
		err := validateBiometricEvent(biometricEvent{UserID: "user-1", MetricType: "unknown", Value: 10})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown metric_type")
	})

	t.Run("spo2 out of range", func(t *testing.T) {
		err := validateBiometricEvent(biometricEvent{UserID: "user-1", MetricType: "spo2", Value: 50})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "spo2 out of valid range")
	})
}

func TestGetMetricRules(t *testing.T) {
	t.Run("known metric", func(t *testing.T) {
		rules, ok := getMetricRules("heart_rate")
		assert.True(t, ok)
		assert.Equal(t, 30.0, rules.Min)
		assert.Equal(t, 220.0, rules.Max)
		assert.Equal(t, "heart_rate", rules.Name)
	})

	t.Run("unknown metric", func(t *testing.T) {
		_, ok := getMetricRules("unknown_metric")
		assert.False(t, ok)
	})
}

func TestInsertBiometricRecord(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping database test")
	}

	if os.Getenv("POSTGRES_HOST") == "" {
		t.Skip("POSTGRES_HOST not set, skipping database test")
	}

	ctx := context.Background()
	database, err := sql.Open("postgres", os.Getenv("POSTGRES_URL"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = database.Close() }()

	if pingErr := database.PingContext(ctx); pingErr != nil {
		t.Skipf("database not available: %v", pingErr)
	}

	_, err = database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS biometric_data_test (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			metric_type VARCHAR(100) NOT NULL,
			value DOUBLE PRECISION NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			device_type VARCHAR(100),
			created_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}
	defer func() { _, _ = database.ExecContext(ctx, `DROP TABLE IF EXISTS biometric_data_test`) }()

	event := biometricEvent{
		UserID:     "test-user",
		MetricType: "heart_rate",
		Value:      140.0,
		Timestamp:  time.Now().UTC(),
	}

	err = insertBiometricRecord(ctx, database, event)
	assert.NoError(t, err)

	var count int
	err = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM biometric_data_test WHERE user_id = $1", "test-user").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}
