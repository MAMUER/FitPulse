package main

import (
	"net/http"
	"time"

	"github.com/MAMUER/project/cmd/device-aggregator/providers"
	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
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

	fitbit := providers.NewFitbitProvider(database, log.Logger)
	agg := newAggregator(database, log, fitbit)
	s := newServer(agg)

	r := chi.NewRouter()

	// Apply middleware
	r.Use(middleware.CorrelationIDHTTP)

	r.Get("/health", s.agg.healthHandler)
	r.Get("/api/v1/devices/fitbit/auth", s.agg.fitbitAuthHandler)
	r.Get("/api/v1/devices/fitbit/callback", s.agg.fitbitCallbackHandler)
	r.Post("/api/v1/devices/fitbit/webhook", fitbitWebhookHandler)
	r.Post("/api/v1/devices/fitbit/disconnect", s.agg.fitbitDisconnectHandler)
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
