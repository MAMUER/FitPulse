package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/MAMUER/project/cmd/device-aggregator/providers"
	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/crypto"
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/middleware"
)

// Package devices aggregates wearable device providers and syncs data.
type server struct {
	agg *aggregator
}

func newServer(agg *aggregator) *server {
	return &server{agg: agg}
}

func main() {
	log := logger.New("device-aggregator")
	defer func() { _ = log.Sync() }()

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
	defer func() { _ = database.Close() }()

	deviceEncryptionKey := config.GetEnv("DEVICE_TOKEN_ENCRYPTION_KEY")
	if deviceEncryptionKey == "" {
		log.Fatal("DEVICE_TOKEN_ENCRYPTION_KEY environment variable is required")
	}

	deviceEncryptor, initErr := crypto.NewAESGCMEncryptor(deviceEncryptionKey)
	if initErr != nil {
		log.Fatal("Failed to initialize device token encryption", zap.Error(initErr))
	}

	fitbit := providers.NewFitbitProvider(database, log.Logger, deviceEncryptor)
	garmin := providers.NewGarminProvider(database, log.Logger, deviceEncryptor)
	withings := providers.NewWithingsProvider(database, log.Logger, deviceEncryptor)
	agg := newAggregator(database, log, fitbit, garmin, withings)
	s := newServer(agg)

	r := chi.NewRouter()

	// Apply middleware
	r.Use(middleware.CorrelationIDHTTP)

	r.Get("/health", s.agg.healthHandler)
	r.Get("/api/v1/devices/fitbit/auth", s.agg.fitbitAuthHandler)
	r.Get("/api/v1/devices/fitbit/callback", s.agg.fitbitCallbackHandler)
	r.Post("/api/v1/devices/fitbit/webhook", fitbitWebhookHandler)
	r.Post("/api/v1/devices/fitbit/disconnect", s.agg.fitbitDisconnectHandler)

	r.Get("/api/v1/devices/garmin/auth", s.agg.garminAuthHandler)
	r.Get("/api/v1/devices/garmin/callback", s.agg.garminCallbackHandler)
	r.Post("/api/v1/devices/garmin/disconnect", s.agg.garminDisconnectHandler)

	r.Get("/api/v1/devices/withings/auth", s.agg.withingsAuthHandler)
	r.Get("/api/v1/devices/withings/callback", s.agg.withingsCallbackHandler)
	r.Post("/api/v1/devices/withings/disconnect", s.agg.withingsDisconnectHandler)

	r.Get("/api/v1/devices/providers", s.agg.listProvidersHandler)

	srv := &http.Server{
		Addr:         ":8083",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Info("Device aggregator starting", zap.String("port", "8083"))
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}
}
