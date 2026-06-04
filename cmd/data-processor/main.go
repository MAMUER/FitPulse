package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/queue"
	"go.uber.org/zap"
)

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
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}
	database, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Error("Failed to connect to database", zap.Error(err))
		return err
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
			log.Info("Connected to RabbitMQ")
		}
	}

	<-stopCh

	log.Info("Data processor shutting down")
	return nil
}
