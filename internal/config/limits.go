// Package config provides configuration constants and limits.
package config

import "time"

const (
	DefaultTimeout      = 5 * time.Second
	MaxBatchSize        = 100
	RedisTTLSeconds     = 3600
	JWTExpirationHours  = 24
	MinHeartRate        = 30
	MaxHeartRate        = 220
	MinSpO2             = 70
	MaxSpO2             = 100
	CorrelationIDHeader = "X-Correlation-ID"
)
