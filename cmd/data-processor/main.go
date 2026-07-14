package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/queue"
	"github.com/google/uuid"
)

type biometricEvent struct {
	UserID     string    `json:"user_id"`
	MetricType string    `json:"metric_type"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
	DeviceType *string   `json:"device_type,omitempty"`
}

func main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	if err := run(quit); err != nil {
		os.Exit(1)
	}
}

// run starts the data processor. stopCh is used to gracefully stop the service (useful for testing).
func run(stopCh <-chan os.Signal) error {
	log := logger.New("data-processor")
	defer func() { _ = log.Sync() }()

	log.Info("Data processor service starting")

	dbCfg := db.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		DBName:   os.Getenv("POSTGRES_DB"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}
	database, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Error("Failed to connect to database", zap.Error(err))
		return fmt.Errorf("connect database: %w", err)
	}
	defer func() { _ = database.Close() }()

	rabbitURL := os.Getenv("RABBITMQ_URL")
	var consumer queue.Consumer
	if rabbitURL != "" {
		consumer, err = queue.NewConsumer(rabbitURL, "biometric_events", log.Logger)
		if err != nil {
			log.Warn("Failed to connect to RabbitMQ", zap.Error(err))
		} else {
			defer func() { _ = consumer.Close() }()
			log.Info("Connected to RabbitMQ", zap.String("queue", "biometric_events"))
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if consumer != nil {
		go func() {
			log.Info("Starting biometric_events consumer loop")
			consumeBiometricEvents(ctx, database, consumer, log)
		}()
	}

	<-stopCh
	log.Info("Data processor shutting down")
	return nil
}

func consumeBiometricEvents(ctx context.Context, database *sql.DB, consumer queue.Consumer, log *logger.Logger) {
	const insertQuery = `
		INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

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

			event, err := parseBiometricEvent(msg.Body)
			if err != nil {
				log.Error("Failed to parse biometric event", zap.Error(err), zap.String("body", string(msg.Body)))
				if nackErr := consumer.Nack(msg.DeliveryTag, false, false); nackErr != nil {
					log.Error("Failed to nack message", zap.Error(nackErr))
				}
				continue
			}

			if err := insertBiometricRecord(ctx, database, insertQuery, event); err != nil {
				log.Error("Failed to insert biometric record",
					zap.Error(err),
					zap.String("user_id", event.UserID),
					zap.String("metric_type", event.MetricType),
				)
				if nackErr := consumer.Nack(msg.DeliveryTag, false, true); nackErr != nil {
					log.Error("Failed to nack message", zap.Error(nackErr))
				}
				continue
			}

			if ackErr := consumer.Ack(msg.DeliveryTag, false); ackErr != nil {
				log.Error("Failed to ack message", zap.Error(ackErr))
			}
		}
	}
}

func parseBiometricEvent(body []byte) (biometricEvent, error) {
	var event biometricEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return event, fmt.Errorf("unmarshal event: %w", err)
	}
	if event.UserID == "" {
		return event, fmt.Errorf("user_id is empty")
	}
	if event.MetricType == "" {
		return event, fmt.Errorf("metric_type is empty")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	return event, nil
}

func insertBiometricRecord(ctx context.Context, database *sql.DB, query string, event biometricEvent) error {
	var deviceType sql.NullString
	if event.DeviceType != nil && *event.DeviceType != "" {
		deviceType = sql.NullString{String: *event.DeviceType, Valid: true}
	}

	_, err := database.ExecContext(ctx, query,
		uuid.New().String(),
		event.UserID,
		event.MetricType,
		event.Value,
		event.Timestamp,
		deviceType,
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert biometric_data: %w", err)
	}
	return nil
}
