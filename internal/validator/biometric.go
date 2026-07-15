// Package validator provides input validation utilities for API requests.
package validator

import (
	"errors"
	"fmt"

	pb "github.com/MAMUER/project/api/gen/biometric"
)

var (
	ErrUserIDRequired        = errors.New("user_id is required")
	ErrMetricTypeRequired    = errors.New("metric_type is required")
	ErrValueNegative         = errors.New("value cannot be negative")
	ErrHeartRateOutOfRange   = errors.New("heart_rate out of valid range")
	ErrSpO2OutOfRange        = errors.New("spo2 out of valid range")
	ErrTemperatureOutOfRange = errors.New("temperature out of valid range")
	ErrBPSystolicOutOfRange  = errors.New("blood_pressure_systolic out of valid range")
	ErrBPDiastolicOutOfRange = errors.New("blood_pressure_diastolic out of valid range")
	ErrStepsOutOfRange       = errors.New("steps out of valid range")
	ErrHRVOutOfRange         = errors.New("hrv out of valid range")
)

type MetricRules struct {
	Min, Max float64
	Name     string
}

func getMetricRules(metricType string) (MetricRules, bool) {
	rules := map[string]MetricRules{
		"heart_rate":               {30, 220, "heart_rate"},
		"spo2":                     {70, 100, "spo2"},
		"temperature":              {35.5, 38.5, "temperature"},
		"blood_pressure_systolic":  {80, 200, "blood_pressure_systolic"},
		"blood_pressure_diastolic": {50, 130, "blood_pressure_diastolic"},
		"steps":                    {0, 100000, "steps"},
		"hrv":                      {0, 200, "hrv"},
	}
	r, ok := rules[metricType]
	return r, ok
}

func ValidateBiometricRequest(req *pb.AddRecordRequest) error {
	if req == nil {
		return errors.New("request is nil")
	}

	if req.UserId == "" {
		return ErrUserIDRequired
	}

	return ValidateBiometricRecord(req)
}

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

	if rules, ok := getMetricRules(req.MetricType); ok {
		if req.Value < rules.Min || req.Value > rules.Max {
			return fmt.Errorf("%s out of valid range", rules.Name)
		}
	}

	return nil
}
