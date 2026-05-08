// Package domain provides core business domain models and types.
package domain

import "time"

type BiometricData struct {
	ID         string
	UserID     string
	MetricType string
	Value      float64
	Timestamp  time.Time
	DeviceType string
}
