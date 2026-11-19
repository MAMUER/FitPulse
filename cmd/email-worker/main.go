// Command email-worker consumes email tasks from RabbitMQ and sends them via
// SMTP. It decouples email delivery from user-service: user-service publishes
// tasks to the "emails" queue and returns immediately, while this worker
// performs the actual SMTP send (with retries via dead-lettering).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/email"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/queue"
	"github.com/MAMUER/project/internal/sanitize"
	"github.com/MAMUER/project/internal/server"
	"github.com/MAMUER/project/internal/telemetry"
)

const serviceName = "email-worker"

func main() {
	log := logger.New(serviceName)
	defer func() { _ = log.Sync() }()

	shutdownTraces := telemetry.InitTracer()
	defer func() {
		if err := shutdownTraces(context.Background()); err != nil {
			log.Warn("Failed to shutdown traces", zap.Error(err))
		}
	}()

	config.InitViper("email-worker")

	port := config.GetEnv("EMAIL_WORKER_PORT", "8086")
	metricsPort := config.GetEnv("EMAIL_WORKER_METRICS_PORT", "9097")

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"status":"healthy"}`)); err != nil {
			log.Warn("Failed to write health response", zap.Error(err))
		}
	})

	cfg := server.DefaultConfig(port, metricsPort)

	server.ServeWithWork(log, cfg, metricsMux, healthMux, healthMux, func(ctx context.Context) error {
		return run(ctx, log)
	})
}

func run(ctx context.Context, log *logger.Logger) error {
	log.Info("Email worker starting")

	rabbitURL := config.GetEnv("RABBITMQ_URL", "")
	if rabbitURL == "" {
		return errors.New("RABBITMQ_URL is required")
	}

	consumer, err := queue.NewConsumer(rabbitURL, email.QueueName, log)
	if err != nil {
		return fmt.Errorf("connect rabbitmq: %w", err)
	}
	defer func() {
		if closeErr := consumer.Close(); closeErr != nil {
			log.Warn("Failed to close queue consumer", zap.Error(closeErr))
		}
	}()

	sender := email.NewSMTPClient(email.LoadConfig())

	msgs := consumer.Messages()
	for {
		select {
		case <-ctx.Done():
			log.Info("Email worker shutting down")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				log.Warn("RabbitMQ message channel closed")
				return nil
			}
			processEmailTask(ctx, sender, consumer, log, msg)
		}
	}
}

// processEmailTask parses an email task and sends it. On success the message is
// acked; on any failure it is nacked (requeue=false) so it is dead-lettered to
// the .dlq queue for inspection/replay instead of looping forever.
func processEmailTask(ctx context.Context, sender *email.SMTPClient, consumer queue.Consumer, log *logger.Logger, msg amqp.Delivery) {
	var task email.Task
	if err := json.Unmarshal(msg.Body, &task); err != nil {
		log.Error("Failed to parse email task", zap.Error(err), zap.String("body", sanitize.LogString(string(msg.Body))))
		nack(consumer, msg, log)
		return
	}

	var sendErr error
	switch task.Type {
	case email.TypeVerification:
		sendCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		sendErr = sender.SendVerificationEmail(sendCtx, task.To, task.Token, task.BaseURL)
		cancel()
	default:
		sendErr = fmt.Errorf("unknown email type %q", task.Type)
	}

	if sendErr != nil {
		log.Error("Failed to send email (dead-lettering)",
			zap.Error(sendErr),
			zap.String("type", sanitize.LogString(task.Type)),
			zap.String("to", sanitize.LogString(task.To)))
		nack(consumer, msg, log)
		return
	}

	log.Info("Email sent", zap.String("type", sanitize.LogString(task.Type)), zap.String("to", sanitize.LogString(task.To)))
	if ackErr := consumer.Ack(msg.DeliveryTag, false); ackErr != nil {
		log.Error("Failed to ack message", zap.Error(ackErr))
	}
}

func nack(consumer queue.Consumer, msg amqp.Delivery, log *logger.Logger) {
	if nackErr := consumer.Nack(msg.DeliveryTag, false, false); nackErr != nil {
		log.Error("Failed to nack message", zap.Error(nackErr))
	}
}
