// Package server provides HTTP server lifecycle management.

package server

import (
	"context"
	"net/http"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
)

// Config holds server configuration.

type Config struct {
	Port string

	MetricsPort string

	ReadTimeout time.Duration

	WriteTimeout time.Duration

	IdleTimeout time.Duration

	ShutdownTimeout time.Duration
}

// DefaultConfig returns default server configuration.

func DefaultConfig(port, metricsPort string) Config {

	return Config{

		Port: port,

		MetricsPort: metricsPort,

		ReadTimeout: 15 * time.Second,

		WriteTimeout: 30 * time.Second,

		IdleTimeout: 60 * time.Second,

		ShutdownTimeout: 10 * time.Second,
	}

}

// Serve starts the main and metrics servers and handles graceful shutdown.

// It blocks until a termination signal is received.

func Serve(log *logger.Logger, cfg Config, metricsMux, mainMux *http.ServeMux, mainHandler http.Handler) {

	metricsSrv := &http.Server{

		Addr: ":" + cfg.MetricsPort,

		Handler: metricsMux,

		ReadHeaderTimeout: 5 * time.Second,
	}

	srv := &http.Server{

		Addr: ":" + cfg.Port,

		Handler: mainHandler,

		ReadTimeout: cfg.ReadTimeout,

		WriteTimeout: cfg.WriteTimeout,

		IdleTimeout: cfg.IdleTimeout,

		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {

		log.Info("Starting metrics server", zap.String("port", cfg.MetricsPort))

		if err := metricsSrv.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "Server closed") {

			log.Fatal("Metrics server failed", zap.Error(err))

		}

	}()

	go func() {

		log.Info("Starting service", zap.String("port", cfg.Port))

		if err := srv.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "Server closed") {

			log.Fatal("Server failed", zap.Error(err))

		}

	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	<-ctx.Done()

	log.Info("Shutting down service")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)

	defer cancel()

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {

		defer wg.Done()

		if err := srv.Shutdown(shutdownCtx); err != nil {

			log.Error("HTTP server shutdown error", zap.Error(err))

		}

	}()

	go func() {

		defer wg.Done()

		if err := metricsSrv.Shutdown(shutdownCtx); err != nil {

			log.Error("Metrics server shutdown error", zap.Error(err))

		}

	}()

	wg.Wait()

	log.Info("Service stopped")

}
