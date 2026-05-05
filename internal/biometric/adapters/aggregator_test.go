package adapters

import (
	"context"
	"errors"
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
