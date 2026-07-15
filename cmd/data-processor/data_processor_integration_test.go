package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/MAMUER/project/internal/logger"
)

func TestDataProcessorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if !isDockerAvailable(t) {
		t.Skip("Docker is not available")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:18-alpine",
		testcontainers.WithEnv(map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	_ = pgContainer.Terminate(ctx)

	rabbitContainer, err := rabbitmq.Run(ctx, "rabbitmq:4-management-alpine",
		testcontainers.WithEnv(map[string]string{
			"RABBITMQ_DEFAULT_USER": "testuser",
			"RABBITMQ_DEFAULT_PASS": "testpass",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Server startup complete").WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start rabbitmq container: %v", err)
	}
	_ = rabbitContainer.Terminate(ctx)

	pgHost, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get postgres host: %v", err)
	}
	pgPort, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get postgres port: %v", err)
	}

	rabbitHost, err := rabbitContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get rabbitmq host: %v", err)
	}
	rabbitPort, err := rabbitContainer.MappedPort(ctx, "5672")
	if err != nil {
		t.Fatalf("failed to get rabbitmq port: %v", err)
	}

	rabbitURL := fmt.Sprintf("amqp://testuser:testpass@%s:%d/", rabbitHost, rabbitPort)

	t.Setenv("DB_HOST", pgHost)
	t.Setenv("DB_PORT", pgPort.Port())
	t.Setenv("POSTGRES_USER", "testuser")
	t.Setenv("POSTGRES_PASSWORD", "testpass")
	t.Setenv("POSTGRES_DB", "testdb")
	t.Setenv("DB_SSLMODE", "disable")
	t.Setenv("RABBITMQ_URL", rabbitURL)

	db, err := sql.Open("postgres", fmt.Sprintf("postgres://testuser:testpass@%s:%d/testdb?sslmode=disable", pgHost, pgPort))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS biometric_data (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			metric_type VARCHAR(100) NOT NULL,
			value DOUBLE PRECISION NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			device_type VARCHAR(100),
			created_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}
	defer func() { _, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS biometric_data") }()

	processCtx, processCancel := context.WithCancel(ctx)
	log := logger.New("data-processor-test")

	go func() {
		_ = run(processCtx, log)
	}()

	amqpConn, err := amqp.Dial(rabbitURL)
	if err != nil {
		t.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer func() { _ = amqpConn.Close() }()

	amqpCh, err := amqpConn.Channel()
	if err != nil {
		t.Fatalf("failed to open channel: %v", err)
	}
	defer func() { _ = amqpCh.Close() }()

	_, err = amqpCh.QueueDeclare("biometric_events", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("failed to declare queue: %v", err)
	}

	userID := "integration-user-" + uuid.New().String()
	event := biometricEvent{
		UserID:     userID,
		MetricType: "heart_rate",
		Value:      160.0,
		Timestamp:  time.Now().UTC(),
		DeviceType: ptrString("test-device"),
	}
	body, _ := json.Marshal(event)

	err = amqpCh.Publish("", "biometric_events", false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
	})
	if err != nil {
		t.Fatalf("failed to publish event: %v", err)
	}

	assert.Eventually(t, func() bool {
		var count int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM biometric_data WHERE user_id = $1", userID).Scan(&count)
		if err != nil {
			t.Logf("query error: %v", err)
			return false
		}
		return count == 1
	}, 15*time.Second, 500*time.Millisecond, "message should be processed and inserted")

	processCancel()
	time.Sleep(500 * time.Millisecond)
}

func ptrString(s string) *string {
	return &s
}

func isDockerAvailable(t *testing.T) bool {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:2375/version")
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK
}
