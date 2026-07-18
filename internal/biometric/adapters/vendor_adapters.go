// Package adapters provides biometric data source adapters, aggregators, and decorators.
package adapters

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/MAMUER/project/internal/biometric/config"
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
	rateLimit   time.Duration
	lastRequest time.Time
	mu          sync.Mutex
	userID      string
	deviceID    string
}

// newBaseAdapter creates a new base adapter with shared HTTP client and rate limiting state.
func newBaseAdapter(deviceType, baseURL, apiKey, userID, deviceID string, rateLimit time.Duration) *baseAdapter {
	if rateLimit <= 0 {
		rateLimit = 100 * time.Millisecond
	}
	return &baseAdapter{
		client:     &http.Client{Timeout: 15 * time.Second},
		baseURL:    baseURL,
		apiKey:     apiKey,
		deviceType: deviceType,
		userID:     userID,
		deviceID:   deviceID,
		sourceID:   fmt.Sprintf("%s-%d", deviceType, time.Now().UnixNano()),
		rateLimit:  rateLimit,
	}
}

// waitForRateLimit blocks until the rate limit window has passed.
// This is safe for concurrent use.
func (b *baseAdapter) waitForRateLimit() {
	b.mu.Lock()
	defer b.mu.Unlock()

	elapsed := time.Since(b.lastRequest)
	if elapsed < b.rateLimit {
		time.Sleep(b.rateLimit - elapsed)
	}
	b.lastRequest = time.Now()
}

// HealthCheck verifies API accessibility without fetching data.
func (b *baseAdapter) HealthCheck(ctx context.Context) error {
	b.waitForRateLimit()
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
// Note: Fetch is a stub implementation. Replace with actual API calls for production use.
type PlatformAdapter struct {
	*baseAdapter
	supportedMetrics map[domain.MetricType]bool
}

// NewPlatformAdapter creates a new platform adapter for the given vendor configuration.
func NewPlatformAdapter(platform, baseURL, apiKey, userID, deviceID string, supportedMetrics map[domain.MetricType]bool, rateLimit time.Duration) *PlatformAdapter {
	base := newBaseAdapter(platform, baseURL, apiKey, userID, deviceID, rateLimit)
	return &PlatformAdapter{
		baseAdapter:      base,
		supportedMetrics: supportedMetrics,
	}
}

// Fetch retrieves biometric data from the platform.
// This is a stub implementation that respects rate limiting.
// TODO: Implement actual API calls for production use.
func (p *PlatformAdapter) Fetch(ctx context.Context, userID string, metricTypes []string) ([]domain.BiometricSample, error) {
	p.waitForRateLimit()
	return []domain.BiometricSample{}, nil
}

// Supports reports whether the adapter can provide the requested metric type.
func (p *PlatformAdapter) Supports(metricType string) bool {
	return p.supportedMetrics[domain.MetricType(metricType)]
}

// DeviceType returns the vendor device type identifier.
func (p *PlatformAdapter) DeviceType() string {
	return p.deviceType
}

// ==========================================
// Adapter Factory
// ==========================================

// AdapterFactory creates configured biometric adapters for known device types.
type AdapterFactory struct {
	apiKeys map[string]string
	config  config.AdapterConfig
	caps    config.VendorCapabilitiesConfig
}

// NewAdapterFactory creates a factory using the provided API keys and default vendor configurations.
func NewAdapterFactory(apiKeys map[string]string) *AdapterFactory {
	return &AdapterFactory{
		apiKeys: apiKeys,
		config:  config.DefaultAdapterConfig(),
		caps:    config.DefaultVendorCapabilities(),
	}
}

// CreateAdapter builds a BiometricSource for the requested device type.
func (f *AdapterFactory) CreateAdapter(deviceType, userID, deviceID string) (domain.BiometricSource, error) {
	vendorConfig, ok := f.config.Vendors[deviceType]
	if !ok {
		return nil, fmt.Errorf("no configuration for device type: %s", deviceType)
	}
	if !vendorConfig.Enabled {
		return nil, fmt.Errorf("device type disabled: %s", deviceType)
	}

	apiKey, ok := f.apiKeys[deviceType]
	if !ok {
		return nil, fmt.Errorf("no API key configured for device type: %s", deviceType)
	}

	caps, ok := f.caps.Vendors[deviceType]
	if !ok {
		caps = map[domain.MetricType]bool{}
	}

	return NewPlatformAdapter(deviceType, vendorConfig.BaseURL, apiKey, userID, deviceID, caps, vendorConfig.RateLimit), nil
}

// Supports reports whether the given device type supports the requested metric.
func (f *AdapterFactory) Supports(deviceType, metricType string) bool {
	caps, ok := f.caps.Vendors[deviceType]
	if !ok {
		return false
	}
	return caps[domain.MetricType(metricType)]
}
