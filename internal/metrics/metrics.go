package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	ActiveRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of in-flight HTTP requests",
		},
	)

	// Additional required metrics
	ErrorTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "error_total",
			Help: "Total number of errors",
		},
		[]string{"service", "error_type"},
	)

	ClassificationConfidence = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "classification_confidence",
			Help:    "Confidence scores for ML classifications",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
	)

	DBConnectionPoolUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_connection_pool_usage",
			Help: "Current database connection pool usage",
		},
		[]string{"db_name"},
	)

	NotificationQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "notification_queue_depth",
			Help: "Current depth of notification queue",
		},
	)

	BiometricSyncLagSeconds = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "biometric_sync_lag_seconds",
			Help: "Lag in seconds for biometric data synchronization",
		},
		[]string{"device_type"},
	)
)
