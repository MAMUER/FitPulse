// Package validator provides input validation utilities for API requests.
package validator

import (
	"errors"
	"fmt"

	pb "github.com/MAMUER/project/api/gen/biometric"
)

var (
	ErrUserIDRequired      = errors.New("user_id is required")
	ErrMetricTypeRequired  = errors.New("metric_type is required")
	ErrValueNegative       = errors.New("value cannot be negative")
	ErrHeartRateOutOfRange = errors.New("heart_rate out of valid range")
	ErrSpO2OutOfRange      = errors.New("spo2 out of valid range")
)

type MetricRules struct {
	Min, Max float64
	Name     string
}

// getMetricRules returns validation rules for a metric type
func getMetricRules(metricType string) (MetricRules, bool) {
	rules := map[string]MetricRules{
		"heart_rate": {30, 220, "heart_rate"},
		"spo2":       {70, 100, "spo2"},
	}
	r, ok := rules[metricType]
	return r, ok
}

// ValidateBiometricRequest проверяет входные данные биометрии
func ValidateBiometricRequest(req *pb.AddRecordRequest) error {
	if req == nil {
		return errors.New("request is nil")
	}

	// Проверка UserId (дополнительно к общей валидации)
	if req.UserId == "" {
		return ErrUserIDRequired
	}

	// Используем общую валидацию записи
	return ValidateBiometricRecord(req)
}

// ValidateBiometricRecord проверяет отдельную запись без UserId
// Используется для batch операций, где UserId берётся из родительского запроса
func ValidateBiometricRecord(req *pb.AddRecordRequest) error {
	if req == nil {
		return errors.New("request is nil")
	}

	if req.MetricType == "" {
		return ErrMetricTypeRequired
	}
	if req.Value < 0 {
		return ErrValueNegative
	}

	// Специфичные правила для метрик
	if rules, ok := getMetricRules(req.MetricType); ok {
		if req.Value < rules.Min || req.Value > rules.Max {
			return fmt.Errorf("%s out of valid range", rules.Name)
		}
	}

	return nil
}
