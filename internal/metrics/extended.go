package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ClassificationConfidence = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "classification_confidence",
			Help: "ML model confidence score for training type classification",
		},
		[]string{"model_version", "class"},
	)

	DBConnectionPoolUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_connection_pool_usage",
			Help: "Fraction of database connection pool currently in use",
		},
		[]string{"service", "pool_name"},
	)

	NotificationQueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "notification_queue_depth",
			Help: "Number of pending messages in notification queues",
		},
		[]string{"queue_name", "priority"},
	)

	BiometricSyncLagSeconds = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "biometric_sync_lag_seconds",
			Help: "Delay in seconds between device data receipt and processing completion",
		},
		[]string{"device_type", "user_segment"},
	)

	BackupSuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backup_success",
			Help: "Indicates whether the latest backup job succeeded (1 = success, 0 = failure)",
		},
		[]string{"type", "job"},
	)
)
