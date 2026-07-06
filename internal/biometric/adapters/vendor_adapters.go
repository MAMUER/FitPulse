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
	userID      string // Added: user identifier
	deviceID    string // Added: device identifier
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
// Apple HealthKit Adapter
// ==========================================

// AppleHealthAdapter implements BiometricSource for Apple Watch / HealthKit.
type AppleHealthAdapter struct {
	*baseAdapter
	userID           string
	deviceID         string
	supportedMetrics map[domain.MetricType]bool
}

func NewAppleHealthAdapter(apiKey, userID, deviceID string) *AppleHealthAdapter {
	base := newBaseAdapter("apple", "https://api.apple-healthkit.example.com/v1", apiKey, userID, deviceID)
	return &AppleHealthAdapter{
		baseAdapter: base,
	}
}

// Fetch retrieves biometric data from Apple HealthKit.
func (a *AppleHealthAdapter) Fetch(ctx context.Context, userID string, metricTypes []string) ([]domain.BiometricSample, error) {
	<-a.rateLimiter.C
	return []domain.BiometricSample{}, nil
}

func (a *AppleHealthAdapter) Supports(metricType string) bool {
	return a.supportedMetrics[domain.MetricType(metricType)]
}

func (a *AppleHealthAdapter) DeviceType() string {
	return "apple"
}

// ==========================================
// Samsung Health Adapter
// ==========================================

// SamsungHealthAdapter implements BiometricSource for Samsung Galaxy Watch.
type SamsungHealthAdapter struct {
	*baseAdapter
	userID           string
	deviceID         string
	supportedMetrics map[domain.MetricType]bool
}

func NewSamsungHealthAdapter(apiKey, userID, deviceID string) *SamsungHealthAdapter {
	base := newBaseAdapter("samsung", "https://api.samsung-health.example.com/v1", apiKey, userID, deviceID)
	return &SamsungHealthAdapter{
		baseAdapter: base,
	}
}

func (s *SamsungHealthAdapter) Fetch(ctx context.Context, userID string, metricTypes []string) ([]domain.BiometricSample, error) {
	<-s.rateLimiter.C
	return []domain.BiometricSample{}, nil
}

func (s *SamsungHealthAdapter) Supports(metricType string) bool {
	return s.supportedMetrics[domain.MetricType(metricType)]
}

func (s *SamsungHealthAdapter) DeviceType() string {
	return "samsung"
}

// ==========================================
// Huawei Health Kit Adapter
// ==========================================

// HuaweiHealthAdapter implements BiometricSource for Huawei Watch D2.
type HuaweiHealthAdapter struct {
	*baseAdapter
	userID           string
	deviceID         string
	supportedMetrics map[domain.MetricType]bool
}

func NewHuaweiHealthAdapter(apiKey, userID, deviceID string) *HuaweiHealthAdapter {
	base := newBaseAdapter("huawei", "https://api.huawei-health.example.com/v1", apiKey, userID, deviceID)
	return &HuaweiHealthAdapter{
		baseAdapter: base,
	}
}

func (h *HuaweiHealthAdapter) Fetch(ctx context.Context, userID string, metricTypes []string) ([]domain.BiometricSample, error) {
	<-h.rateLimiter.C
	return []domain.BiometricSample{}, nil
}

func (h *HuaweiHealthAdapter) Supports(metricType string) bool {
	return h.supportedMetrics[domain.MetricType(metricType)]
}

func (h *HuaweiHealthAdapter) DeviceType() string {
	return "huawei"
}

// ==========================================
// Amazfit Zepp Adapter
// ==========================================

// AmazfitAdapter implements BiometricSource for Amazfit devices via Zepp API.
type AmazfitAdapter struct {
	*baseAdapter
	userID           string
	deviceID         string
	supportedMetrics map[domain.MetricType]bool
}

func NewAmazfitAdapter(apiKey, userID, deviceID string) *AmazfitAdapter {
	base := newBaseAdapter("amazfit", "https://api.zepp-life.example.com/v1", apiKey, userID, deviceID)
	return &AmazfitAdapter{
		baseAdapter: base,
	}
}

func (a *AmazfitAdapter) Fetch(ctx context.Context, userID string, metricTypes []string) ([]domain.BiometricSample, error) {
	<-a.rateLimiter.C
	return []domain.BiometricSample{}, nil
}

func (a *AmazfitAdapter) Supports(metricType string) bool {
	return a.supportedMetrics[domain.MetricType(metricType)]
}

func (a *AmazfitAdapter) DeviceType() string {
	return "amazfit"
}

// ==========================================
// Unified Platform Adapters
// ==========================================

// PlatformAdapter implements BiometricSource for unified aggregator platforms such as ROOK, Terra, Health Connect.
type PlatformAdapter struct {
	*baseAdapter
	userID           string
	deviceID         string
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
	case "apple":
		return NewAppleHealthAdapter(apiKey, userID, deviceID), nil
	case "samsung":
		return NewSamsungHealthAdapter(apiKey, userID, deviceID), nil
	case "huawei":
		return NewHuaweiHealthAdapter(apiKey, userID, deviceID), nil
	case "amazfit":
		return NewAmazfitAdapter(apiKey, userID, deviceID), nil
	case "healthkit":
		return NewPlatformAdapter("healthkit", "https://api.apple-healthkit.example.com/v1", apiKey, userID, deviceID, domain.VendorCapabilities()["apple"]), nil
	case "health_connect":
		return NewPlatformAdapter("health_connect", "https://api.healthconnect.example.com/v1", apiKey, userID, deviceID, map[domain.MetricType]bool{
			domain.MetricHeartRate:        true,
			domain.MetricHRV:              true,
			domain.MetricSpO2:             true,
			domain.MetricTemperature:      true,
			domain.MetricBloodPressureSys: true,
			domain.MetricBloodPressureDia: true,
			domain.MetricECG:              true,
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
			domain.MetricECG:              true,
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
			domain.MetricECG:              true,
			domain.MetricSleepStage:       true,
			domain.MetricSteps:            true,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported device type: %s", deviceType)
	}
}
