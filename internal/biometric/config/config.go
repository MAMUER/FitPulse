// Package config provides configuration types for biometric sources and medical constraints.
package config

import (
	"time"

	"github.com/MAMUER/project/internal/biometric/domain"
)

// VendorConfig holds configuration for a single biometric vendor.
type VendorConfig struct {
	BaseURL    string
	APIKey     string
	DeviceType string
	RateLimit  time.Duration
	Timeout    time.Duration
	Enabled    bool
}

// AdapterConfig holds configuration for the adapter factory.
type AdapterConfig struct {
	Vendors map[string]VendorConfig
}

// DefaultAdapterConfig returns default configuration for known vendors.
func DefaultAdapterConfig() AdapterConfig {
	return AdapterConfig{
		Vendors: map[string]VendorConfig{
			"fitbit": {
				BaseURL:    "https://api.fitbit.com/1",
				RateLimit:  100 * time.Millisecond,
				Timeout:    15 * time.Second,
				Enabled:    true,
				DeviceType: "fitbit",
			},
			"garmin": {
				BaseURL:    "https://connectapi.garmin.com",
				RateLimit:  100 * time.Millisecond,
				Timeout:    15 * time.Second,
				Enabled:    true,
				DeviceType: "garmin",
			},
			"withings": {
				BaseURL:    "https://wbsapi.withings.net",
				RateLimit:  100 * time.Millisecond,
				Timeout:    15 * time.Second,
				Enabled:    true,
				DeviceType: "withings",
			},
			"health_connect": {
				BaseURL:    "https://healthconnect.example.com/v1",
				RateLimit:  100 * time.Millisecond,
				Timeout:    15 * time.Second,
				Enabled:    true,
				DeviceType: "health_connect",
			},
			"terra": {
				BaseURL:    "https://api.terra.example.com/v1",
				RateLimit:  100 * time.Millisecond,
				Timeout:    15 * time.Second,
				Enabled:    true,
				DeviceType: "terra",
			},
			"rook": {
				BaseURL:    "https://api.rook.example.com/v1",
				RateLimit:  100 * time.Millisecond,
				Timeout:    15 * time.Second,
				Enabled:    true,
				DeviceType: "rook",
			},
		},
	}
}

// VendorCapabilitiesConfig holds supported metrics per vendor.
type VendorCapabilitiesConfig struct {
	Vendors map[string]map[domain.MetricType]bool
}

// DefaultVendorCapabilities returns default capabilities for known vendors.
func DefaultVendorCapabilities() VendorCapabilitiesConfig {
	return VendorCapabilitiesConfig{
		Vendors: map[string]map[domain.MetricType]bool{
			"fitbit": {
				domain.MetricHeartRate:        true,
				domain.MetricHRV:              true,
				domain.MetricSpO2:             true,
				domain.MetricTemperature:      false,
				domain.MetricBloodPressureSys: false,
				domain.MetricECG:              false,
				domain.MetricSleepStage:       true,
				domain.MetricSteps:            true,
			},
			"garmin": {
				domain.MetricHeartRate:        true,
				domain.MetricHRV:              true,
				domain.MetricSpO2:             true,
				domain.MetricTemperature:      true,
				domain.MetricBloodPressureSys: false,
				domain.MetricECG:              false,
				domain.MetricSleepStage:       true,
				domain.MetricSteps:            true,
			},
			"withings": {
				domain.MetricHeartRate:        true,
				domain.MetricHRV:              true,
				domain.MetricSpO2:             true,
				domain.MetricTemperature:      true,
				domain.MetricBloodPressureSys: true,
				domain.MetricECG:              false,
				domain.MetricSleepStage:       true,
				domain.MetricSteps:            true,
			},
		},
	}
}
