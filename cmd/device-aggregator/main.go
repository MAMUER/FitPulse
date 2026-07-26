package main

import (
	"context"
	"database/sql"
	"net/http"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/MAMUER/project/cmd/device-aggregator/providers"
	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/crypto"
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/metrics"
	"github.com/MAMUER/project/internal/middleware"
)

func main() {
	log := logger.New("device-aggregator")
	defer func() { _ = log.Sync() }()

	port := config.GetEnv("DEVICE_AGGREGATOR_PORT", "8083")
	metricsPort := config.GetEnv("DEVICE_AGGREGATOR_METRICS_PORT", "9093")

	metricsSrv := createMetricsServer(metricsPort)
	database := connectDatabase(log)
	defer func() { _ = database.Close() }()

	deviceEncryptor := initDeviceEncryptor(log)
	fitbit := providers.NewFitbitProvider(database, log.Logger, deviceEncryptor)
	garmin := providers.NewGarminProvider(database, log.Logger, deviceEncryptor)
	withings := providers.NewWithingsProvider(database, log.Logger, deviceEncryptor)
	agg := newAggregator(database, log, fitbit, garmin, withings)

	r := setupRouter(log, agg)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("Starting metrics server", zap.String("port", metricsPort))
		if err := metricsSrv.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "Server closed") {
			log.Fatal("Metrics server failed", zap.Error(err))
		}
	}()

	go func() {
		log.Info("Device aggregator starting", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "Server closed") {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("Shutting down device aggregator")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = srv.Shutdown(shutdownCtx)
	}()
	go func() {
		defer wg.Done()
		_ = metricsSrv.Shutdown(shutdownCtx)
	}()
	wg.Wait()
	log.Info("Device aggregator stopped")
}

func createMetricsServer(metricsPort string) *http.Server {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	return &http.Server{
		Addr:    ":" + metricsPort,
		Handler: metricsMux,
	}
}

func connectDatabase(log *logger.Logger) *sql.DB {
	dbCfg := db.Config{
		Host:     config.GetEnv("DB_HOST"),
		Port:     config.GetEnv("DB_PORT"),
		User:     config.GetEnv("POSTGRES_USER"),
		Password: config.GetEnv("POSTGRES_PASSWORD"),
		DBName:   config.GetEnv("POSTGRES_DB"),
		SSLMode:  config.GetEnv("DB_SSLMODE"),
	}

	database, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	return database
}

func initDeviceEncryptor(log *logger.Logger) *crypto.AESGCMEncryptor {
	deviceEncryptionKey := config.GetEnv("DEVICE_TOKEN_ENCRYPTION_KEY")
	if deviceEncryptionKey == "" {
		log.Fatal("DEVICE_TOKEN_ENCRYPTION_KEY environment variable is required")
	}

	deviceEncryptor, initErr := crypto.NewAESGCMEncryptor(deviceEncryptionKey)
	if initErr != nil {
		log.Fatal("Failed to initialize device token encryption", zap.Error(initErr))
	}
	return deviceEncryptor
}

func setupRouter(log *logger.Logger, agg *aggregator) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RecoveryMiddleware(log.Logger))
	r.Use(middleware.RequestID)
	r.Use(middleware.CorrelationIDHTTP)
	r.Use(middleware.LoggingMiddleware(log.Logger, metrics.RequestDuration, metrics.RequestsTotal, metrics.ErrorTotal))

	r.Get("/health", agg.healthHandler)
	r.Get("/api/v1/devices/fitbit/auth", agg.fitbitAuthHandler)
	r.Get("/api/v1/devices/fitbit/callback", agg.fitbitCallbackHandler)
	r.Post("/api/v1/devices/fitbit/webhook", fitbitWebhookHandler)
	r.Post("/api/v1/devices/fitbit/disconnect", agg.fitbitDisconnectHandler)

	r.Get("/api/v1/devices/garmin/auth", agg.garminAuthHandler)
	r.Get("/api/v1/devices/garmin/callback", agg.garminCallbackHandler)
	r.Post("/api/v1/devices/garmin/disconnect", agg.garminDisconnectHandler)

	r.Get("/api/v1/devices/withings/auth", agg.withingsAuthHandler)
	r.Get("/api/v1/devices/withings/callback", agg.withingsCallbackHandler)
	r.Post("/api/v1/devices/withings/disconnect", agg.withingsDisconnectHandler)
	r.Post("/api/v1/devices/withings/webhook", withingsWebhookHandler)

	r.Get("/api/v1/devices/providers", agg.listProvidersHandler)

	return r
}
