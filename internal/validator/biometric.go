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

// GetMetricRules returns validation rules for a given biometric metric type.
func GetMetricRules(metricType string) (MetricRules, bool) {
	rules := map[string]MetricRules{
		"heart_rate":               {30, 220, "heart_rate"},
		"resting_heart_rate":       {20, 220, "resting_heart_rate"},
		"spo2":                     {70, 100, "spo2"},
		"temperature":              {35.5, 38.5, "temperature"},
		"body_temperature":         {35.5, 38.5, "body_temperature"},
		"blood_pressure_systolic":  {80, 200, "blood_pressure_systolic"},
		"blood_pressure_diastolic": {50, 130, "blood_pressure_diastolic"},
		"steps":                    {0, 100000, "steps"},
		"hrv":                      {0, 200, "hrv"},
		"hrv_sdnn":                 {0, 300, "hrv_sdnn"},
		"hrv_rmssd":                {0, 300, "hrv_rmssd"},
		"blood_glucose":            {20, 600, "blood_glucose"},
		"active_energy":            {0, 10000, "active_energy"},
		"basal_energy":             {0, 10000, "basal_energy"},
		"bmi":                      {10, 60, "bmi"},
		"lean_body_mass":           {20, 200, "lean_body_mass"},
		"flights_climbed":          {0, 1000, "flights_climbed"},
		"water_intake":             {0, 10000, "water_intake"},
		"vo2_max":                  {10, 100, "vo2_max"},
		"respiratory_rate":         {5, 80, "respiratory_rate"},
		"weight":                   {20, 500, "weight"},
		"sleep_stages":             {0, 1000000, "sleep_stages"},
		"workout":                  {0, 1000000, "workout"},
		"menstrual_cycle":          {0, 1000000, "menstrual_cycle"},
		"body_composition":         {0, 1000000, "body_composition"},
	}
	r, ok := rules[metricType]
	return r, ok
}

// ValidateBiometricRequest validates an AddRecordRequest for biometric data.
func ValidateBiometricRequest(req *pb.AddRecordRequest) error {
	if req == nil {
		return NilRequestError()
	}

	if err := RequireString(req.UserId, "user_id", ErrUserIDRequired); err != nil {
		return err
	}

	return ValidateBiometricRecord(req)
}

// ValidateBiometricRecord validates the biometric record fields.
func ValidateBiometricRecord(req *pb.AddRecordRequest) error {
	if req == nil {
		return NilRequestError()
	}

	if err := RequireString(req.MetricType, "metric_type", ErrMetricTypeRequired); err != nil {
		return err
	}
	if req.Value < 0 {
		return status.Error(codes.InvalidArgument, ErrValueNegative.Error())
	}

	if rules, ok := GetMetricRules(req.MetricType); ok {
		if req.Value < rules.Min || req.Value > rules.Max {
			return status.Error(codes.InvalidArgument, rules.Name+" out of valid range")
		}
	}

	return nil
}
