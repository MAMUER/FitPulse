package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/server"
)

var (
	webhookRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "device_aggregator_webhook_requests_total",
			Help: "Total number of webhook requests received by device-aggregator",
		},
		[]string{"source", "status"},
	)
)

func init() {
	prometheus.MustRegister(webhookRequestsTotal)
}

func main() {
	log := logger.New("device-aggregator")
	defer func() { _ = log.Sync() }()

	config.InitViper("device-aggregator")
	_ = config.GetViper()

	port := config.GetEnv("DEVICE_AGGREGATOR_PORT", "8083")
	metricsPort := config.GetEnv("DEVICE_AGGREGATOR_METRICS_PORT", "9093")

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"device-aggregator"}`))
	})
	mux.HandleFunc("/api/v1/integrations/open-wearables/webhook", openWearablesWebhookHandler)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	cfg := server.DefaultConfig(port, metricsPort)
	server.Serve(log, cfg, metricsMux, mux, mux)
}
