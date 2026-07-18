package adapters

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/biometric/domain"
	"github.com/MAMUER/project/internal/logger"
)

// LoggerAdapter wraps a biometric source with structured logging.
type LoggerAdapter struct {
	source domain.BiometricSource
	log    *logger.Logger
}

// NewLoggerAdapter creates a new logging adapter for a biometric source.
func NewLoggerAdapter(source domain.BiometricSource, log *logger.Logger) *LoggerAdapter {
	return &LoggerAdapter{
		source: source,
		log:    log,
	}
}

// Fetch logs the fetch operation.
func (a *LoggerAdapter) Fetch(ctx context.Context, userID string, metricTypes []string) ([]domain.BiometricSample, error) {
	a.log.Debug("Fetching biometric data",
		zap.String("source", a.source.DeviceType()),
		zap.String("user_id", userID),
		zap.Strings("metric_types", metricTypes),
	)

	samples, err := a.source.Fetch(ctx, userID, metricTypes)
	if err != nil {
		a.log.Error("Failed to fetch biometric data",
			zap.String("source", a.source.DeviceType()),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}

	a.log.Debug("Biometric data fetched successfully",
		zap.String("source", a.source.DeviceType()),
		zap.String("user_id", userID),
		zap.Int("samples_count", len(samples)),
	)

	return samples, nil
}

// Supports logs the supported metrics check.
func (a *LoggerAdapter) Supports(metricType string) bool {
	supported := a.source.Supports(metricType)
	a.log.Debug("Checking metric support",
		zap.String("source", a.source.DeviceType()),
		zap.String("metric_type", metricType),
		zap.Bool("supported", supported),
	)
	return supported
}

// DeviceType returns the device type.
func (a *LoggerAdapter) DeviceType() string {
	return a.source.DeviceType()
}

// HealthCheck logs the health check operation.
func (a *LoggerAdapter) HealthCheck(ctx context.Context) error {
	a.log.Debug("Performing health check",
		zap.String("source", a.source.DeviceType()),
	)

	err := a.source.HealthCheck(ctx)
	if err != nil {
		a.log.Error("Health check failed",
			zap.String("source", a.source.DeviceType()),
			zap.Error(err),
		)
		return err
	}

	a.log.Debug("Health check passed",
		zap.String("source", a.source.DeviceType()),
	)

	return nil
}

// MetricsAdapter wraps a biometric source with Prometheus metrics.
type MetricsAdapter struct {
	source        domain.BiometricSource
	fetchCounter  *prometheus.CounterVec
	fetchDuration *prometheus.HistogramVec
}

// NewMetricsAdapter creates a new metrics adapter for a biometric source.
func NewMetricsAdapter(source domain.BiometricSource, fetchCounter *prometheus.CounterVec, fetchDuration *prometheus.HistogramVec) *MetricsAdapter {
	return &MetricsAdapter{
		source:        source,
		fetchCounter:  fetchCounter,
		fetchDuration: fetchDuration,
	}
}

// Fetch records metrics for the fetch operation.
func (a *MetricsAdapter) Fetch(ctx context.Context, userID string, metricTypes []string) ([]domain.BiometricSample, error) {
	start := time.Now()

	samples, err := a.source.Fetch(ctx, userID, metricTypes)
	duration := time.Since(start).Seconds()

	a.fetchCounter.WithLabelValues(a.source.DeviceType(), strconv.FormatBool(err == nil)).Inc()
	a.fetchDuration.WithLabelValues(a.source.DeviceType()).Observe(duration)

	return samples, err
}

// Supports checks if the source supports a metric type.
func (a *MetricsAdapter) Supports(metricType string) bool {
	return a.source.Supports(metricType)
}

// DeviceType returns the device type.
func (a *MetricsAdapter) DeviceType() string {
	return a.source.DeviceType()
}

// HealthCheck checks the health of the source.
func (a *MetricsAdapter) HealthCheck(ctx context.Context) error {
	return a.source.HealthCheck(ctx)
}
