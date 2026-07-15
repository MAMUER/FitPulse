package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/metrics"
	"github.com/MAMUER/project/internal/queue"
)

type biometricEvent struct {
	UserID     string    `json:"user_id"`
	MetricType string    `json:"metric_type"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
	DeviceType *string   `json:"device_type,omitempty"`
}

func main() {
	log := logger.New("data-processor")
	defer func() { _ = log.Sync() }()

	port := config.GetEnv("DATA_PROCESSOR_PORT", "8084")
	metricsPort := config.GetEnv("DATA_PROCESSOR_METRICS_PORT", "9092")

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsSrv := &http.Server{
		Addr:    ":" + metricsPort,
		Handler: metricsMux,
	}

	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})
	healthSrv := &http.Server{
		Addr:    ":" + port,
		Handler: healthMux,
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
		log.Info("Starting health server", zap.String("port", port))
		if err := healthSrv.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "Server closed") {
			log.Fatal("Health server failed", zap.Error(err))
		}
	}()

	if err := run(ctx, log); err != nil {
		log.Error("Data processor failed", zap.Error(err))
	}

	log.Info("Shutting down data processor")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = healthSrv.Shutdown(shutdownCtx)
	}()
	go func() {
		defer wg.Done()
		_ = metricsSrv.Shutdown(shutdownCtx)
	}()
	wg.Wait()
}

func run(ctx context.Context, log *logger.Logger) error {
	log.Info("Data processor service starting")

	dbCfg := db.Config{
		Host:     config.GetEnv("DB_HOST"),
		Port:     config.GetEnv("DB_PORT"),
		User:     config.GetEnv("POSTGRES_USER"),
		Password: config.GetEnv("POSTGRES_PASSWORD"),
		DBName:   config.GetEnv("POSTGRES_DB"),
		SSLMode:  config.GetEnv("DB_SSLMODE", "disable"),
	}
	database, err := db.NewConnection(dbCfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer func() { _ = database.Close() }()

	rabbitURL := config.GetEnv("RABBITMQ_URL")
	if rabbitURL == "" {
		return errors.New("RABBITMQ_URL is required")
	}

	consumer, err := queue.NewConsumer(rabbitURL, "biometric_events", log.Logger)
	if err != nil {
		return fmt.Errorf("connect rabbitmq: %w", err)
	}
	defer func() { _ = consumer.Close() }()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		consumeBiometricEvents(ctx, database, consumer, log)
	}()

	<-ctx.Done()
	log.Info("Waiting for in-flight messages to finish")

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-done:
		log.Info("All messages processed")
	case <-time.After(30 * time.Second):
		log.Warn("Timeout waiting for messages, shutting down")
	}

	return nil
}

func processBiometricEvent(ctx context.Context, database *sql.DB, consumer queue.Consumer, log *logger.Logger, msg amqp.Delivery) {
	event, err := parseBiometricEvent(msg.Body)
	if err != nil {
		log.Error("Failed to parse biometric event", zap.Error(err), zap.String("body", string(msg.Body)))
		metrics.ErrorTotal.WithLabelValues("data-processor", "parse_error").Inc()
		if nackErr := consumer.Nack(msg.DeliveryTag, false, false); nackErr != nil {
			log.Error("Failed to nack message", zap.Error(nackErr))
		}
		return
	}

	if err := validateBiometricEvent(event); err != nil {
		log.Error("Invalid biometric event", zap.Error(err), zap.String("user_id", event.UserID))
		metrics.ErrorTotal.WithLabelValues("data-processor", "validation_error").Inc()
		if nackErr := consumer.Nack(msg.DeliveryTag, false, false); nackErr != nil {
			log.Error("Failed to nack message", zap.Error(nackErr))
		}
		return
	}

	if err := insertBiometricRecord(ctx, database, event); err != nil {
		log.Error("Failed to insert biometric record",
			zap.Error(err),
			zap.String("user_id", event.UserID),
			zap.String("metric_type", event.MetricType),
		)
		metrics.ErrorTotal.WithLabelValues("data-processor", "insert_error").Inc()
		if nackErr := consumer.Nack(msg.DeliveryTag, false, true); nackErr != nil {
			log.Error("Failed to nack message", zap.Error(nackErr))
		}
		return
	}

	if ackErr := consumer.Ack(msg.DeliveryTag, false); ackErr != nil {
		log.Error("Failed to ack message", zap.Error(ackErr))
	}
}

func consumeBiometricEvents(ctx context.Context, database *sql.DB, consumer queue.Consumer, log *logger.Logger) {
	msgs := consumer.Messages()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Warn("RabbitMQ messages channel closed")
				return
			}
			processBiometricEvent(ctx, database, consumer, log, msg)
		}
	}
}

func parseBiometricEvent(body []byte) (biometricEvent, error) {
	var event biometricEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return event, fmt.Errorf("unmarshal event: %w", err)
	}
	if event.UserID == "" {
		return event, errors.New("user_id is empty")
	}
	if event.MetricType == "" {
		return event, errors.New("metric_type is empty")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	return event, nil
}

func validateBiometricEvent(event biometricEvent) error {
	if event.UserID == "" {
		return errors.New("user_id is required")
	}
	if event.MetricType == "" {
		return errors.New("metric_type is required")
	}
	if event.Value < 0 {
		return errors.New("value cannot be negative")
	}

	rules, ok := getMetricRules(event.MetricType)
	if !ok {
		return fmt.Errorf("unknown metric_type: %s", event.MetricType)
	}
	if event.Value < rules.Min || event.Value > rules.Max {
		return fmt.Errorf("%s out of valid range (%g-%g)", rules.Name, rules.Min, rules.Max)
	}

	return nil
}

type MetricRules struct {
	Min, Max float64
	Name     string
}

func getMetricRules(metricType string) (MetricRules, bool) {
	rules := map[string]MetricRules{
		"heart_rate":               {30, 220, "heart_rate"},
		"spo2":                     {70, 100, "spo2"},
		"temperature":              {35.5, 38.5, "temperature"},
		"blood_pressure_systolic":  {80, 200, "blood_pressure_systolic"},
		"blood_pressure_diastolic": {50, 130, "blood_pressure_diastolic"},
		"steps":                    {0, 100000, "steps"},
		"hrv":                      {0, 200, "hrv"},
	}
	r, ok := rules[metricType]
	return r, ok
}

func insertBiometricRecord(ctx context.Context, database *sql.DB, event biometricEvent) error {
	var deviceType sql.NullString
	if event.DeviceType != nil && *event.DeviceType != "" {
		deviceType = sql.NullString{String: *event.DeviceType, Valid: true}
	}

	_, err := database.ExecContext(ctx,
		`INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.New().String(), event.UserID, event.MetricType, event.Value, event.Timestamp, deviceType, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert biometric_data: %w", err)
	}
	return nil
}
