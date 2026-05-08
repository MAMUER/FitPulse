package adapters

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MAMUER/Project/internal/biometric/domain"
)

type stubSource struct {
	deviceType string
	supports   map[domain.MetricType]bool
	samples    []domain.BiometricSample
	err        error
}

func (s *stubSource) Fetch(ctx context.Context, userID string, metricTypes []string) ([]domain.BiometricSample, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.samples, nil
}

func (s *stubSource) Supports(metricType string) bool {
	return s.supports[domain.MetricType(metricType)]
}

func (s *stubSource) DeviceType() string {
	return s.deviceType
}

func (s *stubSource) HealthCheck(ctx context.Context) error {
	return s.err
}

func TestCompositeBiometricSourceFetchMergesSources(t *testing.T) {
	ctx := context.Background()
	baseTime := time.Date(2026, 5, 5, 12, 30, 0, 0, time.UTC)

	sourceA := &stubSource{
		deviceType: "rook",
		supports: map[domain.MetricType]bool{
			domain.MetricHeartRate: true,
			domain.MetricSpO2:      true,
		},
		samples: []domain.BiometricSample{
			{UserID: "user-1", MetricType: "heart_rate", Value: 72, Unit: "bpm", Timestamp: baseTime, Quality: "high", Confidence: 0.9},
			{UserID: "user-1", MetricType: "spo2", Value: 98.4, Unit: "%", Timestamp: baseTime, Quality: "fair", Confidence: 0.7},
		},
	}

	sourceB := &stubSource{
		deviceType: "terra",
		supports: map[domain.MetricType]bool{
			domain.MetricHeartRate:   true,
			domain.MetricTemperature: true,
		},
		samples: []domain.BiometricSample{
			{UserID: "user-1", MetricType: "heart_rate", Value: 74, Unit: "bpm", Timestamp: baseTime, Quality: "fair", Confidence: 0.8},
			{UserID: "user-1", MetricType: "temperature", Value: 36.8, Unit: "celsius", Timestamp: baseTime, Quality: "high", Confidence: 0.95},
		},
	}

	source := NewCompositeBiometricSource(sourceA, sourceB)
	samples, err := source.Fetch(ctx, "user-1", []string{"heart_rate", "spo2", "temperature"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(samples) != 3 {
		t.Fatalf("expected 3 merged samples, got %d", len(samples))
	}

	have := map[string]domain.BiometricSample{}
	for _, sample := range samples {
		have[sample.MetricType] = sample
	}

	if hr, ok := have["heart_rate"]; !ok || hr.Value != 72 {
		t.Fatalf("expected heart_rate from rook with value 72, got %v", hr)
	}
	if spo2, ok := have["spo2"]; !ok || spo2.Value != 98.4 {
		t.Fatalf("expected spo2 from rook, got %v", spo2)
	}
	if temp, ok := have["temperature"]; !ok || temp.Value != 36.8 {
		t.Fatalf("expected temperature from terra, got %v", temp)
	}
}

func TestCompositeBiometricSourceSupports(t *testing.T) {
	source := NewCompositeBiometricSource(&stubSource{supports: map[domain.MetricType]bool{domain.MetricECG: true}})
	if !source.Supports("ecg") {
		t.Fatal("expected composite source to support ecg")
	}
	if source.Supports("spo2") {
		t.Fatal("expected composite source to not support spo2")
	}
}

func TestCompositeBiometricSourceHealthCheck(t *testing.T) {
	source := NewCompositeBiometricSource(
		&stubSource{deviceType: "rook", err: errors.New("unreachable")},
		&stubSource{deviceType: "terra", err: nil},
	)
	if err := source.HealthCheck(context.Background()); err != nil {
		t.Fatalf("unexpected healthy composite source: %v", err)
	}

	badSource := NewCompositeBiometricSource(
		&stubSource{deviceType: "rook", err: errors.New("unreachable")},
		&stubSource{deviceType: "terra", err: errors.New("timeout")},
	)
	if err := badSource.HealthCheck(context.Background()); err == nil {
		t.Fatal("expected unhealthy composite source")
	}
}

func TestNewCompositeBiometricSource(t *testing.T) {
	// Test with no sources
	source := NewCompositeBiometricSource()
	composite, ok := source.(*CompositeBiometricSource)
	if !ok {
		t.Fatal("expected CompositeBiometricSource")
	}
	if len(composite.sources) != 0 {
		t.Fatalf("expected 0 sources, got %d", len(composite.sources))
	}

	// Test with multiple sources
	sourceA := &stubSource{deviceType: "sourceA"}
	sourceB := &stubSource{deviceType: "sourceB"}
	source = NewCompositeBiometricSource(sourceA, sourceB)
	composite, ok = source.(*CompositeBiometricSource)
	if !ok {
		t.Fatal("expected CompositeBiometricSource")
	}
	if len(composite.sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(composite.sources))
	}
}

func TestCompositeBiometricSourceDeviceType(t *testing.T) {
	source := NewCompositeBiometricSource()
	if source.DeviceType() != "composite" {
		t.Fatalf("expected device type 'composite', got '%s'", source.DeviceType())
	}
}

func TestCompositeBiometricSourceFetchEmptySources(t *testing.T) {
	ctx := context.Background()
	source := NewCompositeBiometricSource()

	_, err := source.Fetch(ctx, "user-1", []string{"heart_rate"})
	if !errors.Is(err, domain.ErrSourceUnavailable) {
		t.Fatalf("expected ErrSourceUnavailable, got %v", err)
	}
}

func TestCompositeBiometricSourceFetchAllSourcesFail(t *testing.T) {
	ctx := context.Background()
	source := NewCompositeBiometricSource(
		&stubSource{
			deviceType: "source1",
			supports:   map[domain.MetricType]bool{domain.MetricHeartRate: true},
			err:        errors.New("connection failed"),
		},
		&stubSource{
			deviceType: "source2",
			supports:   map[domain.MetricType]bool{domain.MetricHeartRate: true},
			err:        errors.New("timeout"),
		},
	)

	_, err := source.Fetch(ctx, "user-1", []string{"heart_rate"})
	if err == nil {
		t.Fatal("expected error when all sources fail")
	}
	if !strings.Contains(err.Error(), "all sources failed") {
		t.Fatalf("expected 'all sources failed' in error, got %v", err)
	}
}

func TestCompositeBiometricSourceFetchNilSources(t *testing.T) {
	ctx := context.Background()
	source := NewCompositeBiometricSource(
		nil,
		&stubSource{
			deviceType: "source1",
			supports:   map[domain.MetricType]bool{domain.MetricHeartRate: true},
			samples: []domain.BiometricSample{
				{UserID: "user-1", MetricType: "heart_rate", Value: 72, Unit: "bpm", Timestamp: time.Now()},
			},
		},
		nil,
	)

	samples, err := source.Fetch(ctx, "user-1", []string{"heart_rate"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("expected 1 sample, got %d", len(samples))
	}
}

func TestCompositeBiometricSourceFetchNoSupportedMetrics(t *testing.T) {
	ctx := context.Background()
	source := NewCompositeBiometricSource(
		&stubSource{
			deviceType: "source1",
			supports:   map[domain.MetricType]bool{domain.MetricHeartRate: false},
		},
	)

	samples, err := source.Fetch(ctx, "user-1", []string{"heart_rate"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(samples) != 0 {
		t.Fatalf("expected 0 samples, got %d", len(samples))
	}
}

func TestCompositeBiometricSourceSupportsNoSources(t *testing.T) {
	source := NewCompositeBiometricSource()
	if source.Supports("heart_rate") {
		t.Fatal("expected no support when no sources")
	}
}

func TestCompositeBiometricSourceSupportsNilSources(t *testing.T) {
	source := NewCompositeBiometricSource(nil, nil)
	if source.Supports("heart_rate") {
		t.Fatal("expected no support when all sources are nil")
	}
}

func TestCompositeBiometricSourceHealthCheckNoSources(t *testing.T) {
	source := NewCompositeBiometricSource()
	err := source.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected error when no sources")
	}
	if !strings.Contains(err.Error(), "no healthy biometric source") {
		t.Fatalf("expected 'no healthy biometric source' in error, got %v", err)
	}
}

func TestCompositeBiometricSourceHealthCheckNilSources(t *testing.T) {
	source := NewCompositeBiometricSource(nil, nil)
	err := source.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected error when all sources are nil")
	}
}

func TestCompositeBiometricSourceHealthCheckAllNilSources(t *testing.T) {
	source := NewCompositeBiometricSource(nil, nil, nil)
	err := source.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected error when all sources are nil")
	}
}

func TestMergeSamples(t *testing.T) {
	baseTime := time.Date(2026, 5, 5, 12, 30, 0, 0, time.UTC)

	tests := []struct {
		name      string
		samples   []domain.BiometricSample
		expected  int
		checkFunc func(t *testing.T, samples []domain.BiometricSample)
	}{
		{
			name:     "empty slice",
			samples:  []domain.BiometricSample{},
			expected: 0,
		},
		{
			name: "no duplicates",
			samples: []domain.BiometricSample{
				{UserID: "user1", MetricType: "heart_rate", Value: 72, Timestamp: baseTime},
				{UserID: "user1", MetricType: "spo2", Value: 98, Timestamp: baseTime},
			},
			expected: 2,
		},
		{
			name: "quality-based selection - high beats fair",
			samples: []domain.BiometricSample{
				{UserID: "user1", MetricType: "heart_rate", Value: 72, Timestamp: baseTime, Quality: "fair", Confidence: 0.8},
				{UserID: "user1", MetricType: "heart_rate", Value: 74, Timestamp: baseTime, Quality: "high", Confidence: 0.9},
			},
			expected: 1,
			checkFunc: func(t *testing.T, samples []domain.BiometricSample) {
				if len(samples) != 1 {
					return
				}
				if samples[0].Value != 74 || samples[0].Quality != "high" {
					t.Errorf("expected high quality sample with value 74, got value %f quality %s", samples[0].Value, samples[0].Quality)
				}
			},
		},
		{
			name: "confidence-based selection when quality equal",
			samples: []domain.BiometricSample{
				{UserID: "user1", MetricType: "heart_rate", Value: 72, Timestamp: baseTime, Quality: "high", Confidence: 0.8},
				{UserID: "user1", MetricType: "heart_rate", Value: 74, Timestamp: baseTime, Quality: "high", Confidence: 0.9},
			},
			expected: 1,
			checkFunc: func(t *testing.T, samples []domain.BiometricSample) {
				if len(samples) != 1 {
					return
				}
				if samples[0].Value != 74 || samples[0].Confidence != 0.9 {
					t.Errorf("expected higher confidence sample with value 74, got value %f confidence %f", samples[0].Value, samples[0].Confidence)
				}
			},
		},
		{
			name: "low quality defaults to fair score",
			samples: []domain.BiometricSample{
				{UserID: "user1", MetricType: "heart_rate", Value: 72, Timestamp: baseTime, Quality: "low", Confidence: 0.8},
				{UserID: "user1", MetricType: "heart_rate", Value: 74, Timestamp: baseTime, Quality: "unknown", Confidence: 0.9},
			},
			expected: 1,
			checkFunc: func(t *testing.T, samples []domain.BiometricSample) {
				if len(samples) != 1 {
					return
				}
				// Both should default to score 2, so higher confidence wins
				if samples[0].Confidence != 0.9 {
					t.Errorf("expected higher confidence sample to win, got confidence %f", samples[0].Confidence)
				}
			},
		},
		{
			name: "different timestamps treated as separate",
			samples: []domain.BiometricSample{
				{UserID: "user1", MetricType: "heart_rate", Value: 72, Timestamp: baseTime},
				{UserID: "user1", MetricType: "heart_rate", Value: 74, Timestamp: baseTime.Add(time.Minute)},
			},
			expected: 2,
		},
		{
			name: "different users treated as separate",
			samples: []domain.BiometricSample{
				{UserID: "user1", MetricType: "heart_rate", Value: 72, Timestamp: baseTime},
				{UserID: "user2", MetricType: "heart_rate", Value: 74, Timestamp: baseTime},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeSamples(tt.samples)
			if len(result) != tt.expected {
				t.Fatalf("expected %d samples, got %d", tt.expected, len(result))
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}
