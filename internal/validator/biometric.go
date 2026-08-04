// Package validator provides input validation utilities for API requests.
package validator

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/MAMUER/project/api/gen/biometric"
)

var (
	ErrUserIDRequired     = errors.New("user_id is required")
	ErrMetricTypeRequired = errors.New("metric_type is required")
	ErrValueNegative      = errors.New("value cannot be negative")
)

type MetricRules struct {
	Min, Max float64
	Name     string
}

// getMetricRules returns validation rules for a given biometric metric type.
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

// ValidateBiometricRequest validates an AddRecordRequest for biometric data.
func ValidateBiometricRequest(req *pb.AddRecordRequest) error {
	if req == nil {
		return NilRequestError()
	}

	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, ErrUserIDRequired.Error())
	}

	return ValidateBiometricRecord(req)
}

// ValidateBiometricRecord validates the biometric record fields.
func ValidateBiometricRecord(req *pb.AddRecordRequest) error {
	if req == nil {
		return NilRequestError()
	}

	if req.MetricType == "" {
		return status.Error(codes.InvalidArgument, ErrMetricTypeRequired.Error())
	}
	if req.Value < 0 {
		return status.Error(codes.InvalidArgument, ErrValueNegative.Error())
	}

	if rules, ok := getMetricRules(req.MetricType); ok {
		if req.Value < rules.Min || req.Value > rules.Max {
			return status.Error(codes.InvalidArgument, rules.Name+" out of valid range")
		}
	}

	return nil
}
