package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/MAMUER/project/internal/biometric/domain"
)

func TestDefaultAdapterConfig(t *testing.T) {
	cfg := DefaultAdapterConfig()

	assert.NotEmpty(t, cfg.Vendors)
	assert.Len(t, cfg.Vendors, 6)

	vendors := []string{"fitbit", "garmin", "withings", "health_connect", "terra", "rook"}
	for _, vendor := range vendors {
		t.Run(vendor, func(t *testing.T) {
			v, ok := cfg.Vendors[vendor]
			assert.True(t, ok, "vendor %s should exist", vendor)
			assert.Equal(t, vendor, v.DeviceType)
			assert.NotEmpty(t, v.BaseURL)
			assert.True(t, v.Enabled)
			assert.Greater(t, v.RateLimit, time.Duration(0))
			assert.Greater(t, v.Timeout, time.Duration(0))
		})
	}
}

func TestDefaultAdapterConfigFitbit(t *testing.T) {
	cfg := DefaultAdapterConfig()
	v := cfg.Vendors["fitbit"]

	assert.Equal(t, "https://api.fitbit.com/1", v.BaseURL)
	assert.Equal(t, 100*time.Millisecond, v.RateLimit)
	assert.Equal(t, 15*time.Second, v.Timeout)
	assert.True(t, v.Enabled)
	assert.Equal(t, "fitbit", v.DeviceType)
}

func TestDefaultAdapterConfigGarmin(t *testing.T) {
	cfg := DefaultAdapterConfig()
	v := cfg.Vendors["garmin"]

	assert.Equal(t, "https://connectapi.garmin.com", v.BaseURL)
	assert.True(t, v.Enabled)
}

func TestDefaultAdapterConfigWithings(t *testing.T) {
	cfg := DefaultAdapterConfig()
	v := cfg.Vendors["withings"]

	assert.Equal(t, "https://wbsapi.withings.net", v.BaseURL)
	assert.True(t, v.Enabled)
}

func TestDefaultVendorCapabilities(t *testing.T) {
	caps := DefaultVendorCapabilities()

	assert.NotEmpty(t, caps.Vendors)
	assert.Len(t, caps.Vendors, 3)

	vendors := []string{"fitbit", "garmin", "withings"}
	for _, vendor := range vendors {
		t.Run(vendor, func(t *testing.T) {
			v, ok := caps.Vendors[vendor]
			assert.True(t, ok, "vendor %s should exist", vendor)
			assert.NotNil(t, v)
		})
	}
}

func TestDefaultVendorCapabilitiesFitbit(t *testing.T) {
	caps := DefaultVendorCapabilities()
	fitbit := caps.Vendors["fitbit"]

	assert.True(t, fitbit[domain.MetricHeartRate])
	assert.True(t, fitbit[domain.MetricHRV])
	assert.True(t, fitbit[domain.MetricSpO2])
	assert.False(t, fitbit[domain.MetricTemperature])
	assert.False(t, fitbit[domain.MetricBloodPressureSys])
	assert.False(t, fitbit[domain.MetricECG])
	assert.True(t, fitbit[domain.MetricSleepStage])
	assert.True(t, fitbit[domain.MetricSteps])
}

func TestDefaultVendorCapabilitiesGarmin(t *testing.T) {
	caps := DefaultVendorCapabilities()
	garmin := caps.Vendors["garmin"]

	assert.True(t, garmin[domain.MetricHeartRate])
	assert.True(t, garmin[domain.MetricHRV])
	assert.True(t, garmin[domain.MetricSpO2])
	assert.True(t, garmin[domain.MetricTemperature])
	assert.False(t, garmin[domain.MetricBloodPressureSys])
	assert.False(t, garmin[domain.MetricECG])
	assert.True(t, garmin[domain.MetricSleepStage])
	assert.True(t, garmin[domain.MetricSteps])
}

func TestDefaultVendorCapabilitiesWithings(t *testing.T) {
	caps := DefaultVendorCapabilities()
	withings := caps.Vendors["withings"]

	assert.True(t, withings[domain.MetricHeartRate])
	assert.True(t, withings[domain.MetricHRV])
	assert.True(t, withings[domain.MetricSpO2])
	assert.True(t, withings[domain.MetricTemperature])
	assert.True(t, withings[domain.MetricBloodPressureSys])
	assert.False(t, withings[domain.MetricECG])
	assert.True(t, withings[domain.MetricSleepStage])
	assert.True(t, withings[domain.MetricSteps])
}

func TestVendorConfigStruct(t *testing.T) {
	v := VendorConfig{
		BaseURL:    "https://api.example.com",
		APIKey:     "test-key",
		DeviceType: "test",
		RateLimit:  100 * time.Millisecond,
		Timeout:    15 * time.Second,
		Enabled:    true,
	}

	assert.Equal(t, "https://api.example.com", v.BaseURL)
	assert.Equal(t, "test-key", v.APIKey)
	assert.Equal(t, "test", v.DeviceType)
	assert.Equal(t, 100*time.Millisecond, v.RateLimit)
	assert.Equal(t, 15*time.Second, v.Timeout)
	assert.True(t, v.Enabled)
}

func TestAdapterConfigStruct(t *testing.T) {
	cfg := AdapterConfig{
		Vendors: map[string]VendorConfig{
			"test": {
				BaseURL:    "https://api.example.com",
				RateLimit:  100 * time.Millisecond,
				Timeout:    15 * time.Second,
				Enabled:    true,
				DeviceType: "test",
			},
		},
	}

	assert.NotEmpty(t, cfg.Vendors)
	assert.Len(t, cfg.Vendors, 1)
	assert.Equal(t, "https://api.example.com", cfg.Vendors["test"].BaseURL)
}

func TestVendorCapabilitiesConfigStruct(t *testing.T) {
	caps := VendorCapabilitiesConfig{
		Vendors: map[string]map[domain.MetricType]bool{
			"test": {
				domain.MetricHeartRate: true,
				domain.MetricSteps:     true,
			},
		},
	}

	assert.NotEmpty(t, caps.Vendors)
	assert.Len(t, caps.Vendors, 1)
	assert.True(t, caps.Vendors["test"][domain.MetricHeartRate])
	assert.True(t, caps.Vendors["test"][domain.MetricSteps])
	assert.False(t, caps.Vendors["test"][domain.MetricECG])
}
