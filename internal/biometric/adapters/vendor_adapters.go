package adapters

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/MAMUER/project/internal/biometric/domain"
)

// ==========================================
// Base Adapter
// ==========================================

// baseAdapter common fields for all vendor adapters.
type baseAdapter struct {
	client      *http.Client
	baseURL     string
	apiKey      string
	deviceType  string
	sourceID    string
	rateLimiter *time.Ticker
	userID      string
	deviceID    string
}

func newBaseAdapter(deviceType, baseURL, apiKey, userID, deviceID string) *baseAdapter {
	return &baseAdapter{
		client:      &http.Client{Timeout: 15 * time.Second},
		baseURL:     baseURL,
		apiKey:      apiKey,
		deviceType:  deviceType,
		userID:      userID,
		deviceID:    deviceID,
		sourceID:    fmt.Sprintf("%s-%d", deviceType, time.Now().UnixNano()),
		rateLimiter: time.NewTicker(100 * time.Millisecond),
	}
}

// HealthCheck verifies API accessibility without fetching data.
func (b *baseAdapter) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", b.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("create health check request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+b.apiKey)
	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute health check: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: %s", resp.Status)
	}
	return nil
}

// ==========================================
// Platform Adapters
// ==========================================

// PlatformAdapter implements BiometricSource for unified aggregator platforms such as ROOK, Terra, Health Connect.
type PlatformAdapter struct {
	*baseAdapter
	supportedMetrics map[domain.MetricType]bool
}

func NewPlatformAdapter(platform, baseURL, apiKey, userID, deviceID string, supportedMetrics map[domain.MetricType]bool) *PlatformAdapter {
	base := newBaseAdapter(platform, baseURL, apiKey, userID, deviceID)
	return &PlatformAdapter{
		baseAdapter:      base,
		supportedMetrics: supportedMetrics,
	}
}

func (p *PlatformAdapter) Fetch(ctx context.Context, userID string, metricTypes []string) ([]domain.BiometricSample, error) {
	<-p.rateLimiter.C
	return []domain.BiometricSample{}, nil
}

func (p *PlatformAdapter) Supports(metricType string) bool {
	return p.supportedMetrics[domain.MetricType(metricType)]
}

func (p *PlatformAdapter) DeviceType() string {
	return p.deviceType
}

// ==========================================
// Adapter Factory
// ==========================================

type AdapterFactory struct {
	apiKeys map[string]string
}

func NewAdapterFactory(apiKeys map[string]string) *AdapterFactory {
	return &AdapterFactory{apiKeys: apiKeys}
}

func (f *AdapterFactory) CreateAdapter(deviceType, userID, deviceID string) (domain.BiometricSource, error) {
	apiKey, ok := f.apiKeys[deviceType]
	if !ok {
		return nil, fmt.Errorf("no API key configured for device type: %s", deviceType)
	}
	switch deviceType {
	case "fitbit":
		return NewPlatformAdapter("fitbit", "https://api.fitbit.com/1", apiKey, userID, deviceID, map[domain.MetricType]bool{
			domain.MetricHeartRate:  true,
			domain.MetricHRV:        true,
			domain.MetricSpO2:       true,
			domain.MetricSleepStage: true,
			domain.MetricSteps:      true,
		}), nil
	case "garmin":
		return NewPlatformAdapter("garmin", "https://connectapi.garmin.com", apiKey, userID, deviceID, map[domain.MetricType]bool{
			domain.MetricHeartRate:   true,
			domain.MetricHRV:         true,
			domain.MetricSpO2:        true,
			domain.MetricTemperature: true,
			domain.MetricSleepStage:  true,
			domain.MetricSteps:       true,
		}), nil
	case "withings":
		return NewPlatformAdapter("withings", "https://wbsapi.withings.net", apiKey, userID, deviceID, map[domain.MetricType]bool{
			domain.MetricHeartRate:        true,
			domain.MetricHRV:              true,
			domain.MetricSpO2:             true,
			domain.MetricTemperature:      true,
			domain.MetricBloodPressureSys: true,
			domain.MetricBloodPressureDia: true,
			domain.MetricSleepStage:       true,
			domain.MetricSteps:            true,
		}), nil
	case "health_connect":
		return NewPlatformAdapter("health_connect", "https://healthconnect.example.com/v1", apiKey, userID, deviceID, map[domain.MetricType]bool{
			domain.MetricHeartRate:        true,
			domain.MetricHRV:              true,
			domain.MetricSpO2:             true,
			domain.MetricTemperature:      true,
			domain.MetricBloodPressureSys: true,
			domain.MetricBloodPressureDia: true,
			domain.MetricSleepStage:       true,
			domain.MetricSteps:            true,
		}), nil
	case "terra":
		return NewPlatformAdapter("terra", "https://api.terra.example.com/v1", apiKey, userID, deviceID, map[domain.MetricType]bool{
			domain.MetricHeartRate:        true,
			domain.MetricHRV:              true,
			domain.MetricSpO2:             true,
			domain.MetricTemperature:      true,
			domain.MetricBloodPressureSys: true,
			domain.MetricBloodPressureDia: true,
			domain.MetricSleepStage:       true,
			domain.MetricSteps:            true,
		}), nil
	case "rook":
		return NewPlatformAdapter("rook", "https://api.rook.example.com/v1", apiKey, userID, deviceID, map[domain.MetricType]bool{
			domain.MetricHeartRate:        true,
			domain.MetricHRV:              true,
			domain.MetricSpO2:             true,
			domain.MetricTemperature:      true,
			domain.MetricBloodPressureSys: true,
			domain.MetricBloodPressureDia: true,
			domain.MetricSleepStage:       true,
			domain.MetricSteps:            true,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported device type: %s", deviceType)
	}
}
