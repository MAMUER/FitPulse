// Package adapters provides biometric data source adapters and aggregators.
package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/MAMUER/project/internal/biometric/domain"
)

// CompositeBiometricSource aggregates multiple biometric sources, providing a single normalized view.
type CompositeBiometricSource struct {
	sources []domain.BiometricSource
}

// NewCompositeBiometricSource creates a source that merges data from multiple adapters.
func NewCompositeBiometricSource(sources ...domain.BiometricSource) domain.BiometricSource {
	return &CompositeBiometricSource{sources: sources}
}

// Fetch collects data from all available sources and merges duplicate metrics by quality/confidence.
func (c *CompositeBiometricSource) Fetch(ctx context.Context, userID string, metricTypes []string) ([]domain.BiometricSample, error) {
	if len(c.sources) == 0 {
		return nil, domain.ErrSourceUnavailable
	}

	var allSamples []domain.BiometricSample
	var errs []string

	for _, source := range c.sources {
		if source == nil {
			continue
		}

		supportedMetrics := make([]string, 0, len(metricTypes))
		for _, metricType := range metricTypes {
			if source.Supports(metricType) {
				supportedMetrics = append(supportedMetrics, metricType)
			}
		}
		if len(supportedMetrics) == 0 {
			continue
		}

		samples, err := source.Fetch(ctx, userID, supportedMetrics)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", source.DeviceType(), err))
			continue
		}
		for _, sample := range samples {
			if sample.Metadata == nil {
				sample.Metadata = make(map[string]interface{})
			}
			sample.Metadata["source_platform"] = source.DeviceType()
			allSamples = append(allSamples, sample)
		}
	}

	if len(allSamples) == 0 {
		if len(errs) > 0 {
			return nil, fmt.Errorf("all sources failed: %s", strings.Join(errs, "; "))
		}
		return []domain.BiometricSample{}, nil
	}

	return mergeSamples(allSamples), nil
}

func (c *CompositeBiometricSource) Supports(metricType string) bool {
	for _, source := range c.sources {
		if source != nil && source.Supports(metricType) {
			return true
		}
	}
	return false
}

func (c *CompositeBiometricSource) DeviceType() string {
	return "composite"
}

func (c *CompositeBiometricSource) HealthCheck(ctx context.Context) error {
	var errs []string
	healthy := false
	for _, source := range c.sources {
		if source == nil {
			continue
		}
		if err := source.HealthCheck(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", source.DeviceType(), err))
		} else {
			healthy = true
		}
	}
	if healthy {
		return nil
	}
	return fmt.Errorf("no healthy biometric source: %s", strings.Join(errs, "; "))
}

func mergeSamples(samples []domain.BiometricSample) []domain.BiometricSample {
	score := func(sample domain.BiometricSample) int {
		switch strings.ToLower(sample.Quality) {
		case "high":
			return 3
		case "fair":
			return 2
		case "low":
			return 1
		default:
			return 2
		}
	}

	merged := make(map[string]domain.BiometricSample)
	for _, sample := range samples {
		key := fmt.Sprintf("%s|%s|%d", sample.UserID, sample.MetricType, sample.Timestamp.UnixNano())
		existing, found := merged[key]
		if !found {
			merged[key] = sample
			continue
		}
		if score(sample) > score(existing) || (score(sample) == score(existing) && sample.Confidence > existing.Confidence) {
			merged[key] = sample
		}
	}

	result := make([]domain.BiometricSample, 0, len(merged))
	for _, sample := range merged {
		result = append(result, sample)
	}
	return result
}
