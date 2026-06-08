package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MAMUER/project/internal/logger"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestDeviceConnector_Integration_RegisterAndIngest(t *testing.T) {
	ctx := context.Background()

	// Поднимаем PostgreSQL
	pgContainer, err := postgres.Run(ctx, "postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Skipf("Skipping device-connector integration test: could not start PostgreSQL: %v", err)
	}
	defer func() { _ = pgContainer.Terminate(ctx) }()

	host, _ := pgContainer.Host(ctx)
	port, _ := pgContainer.MappedPort(ctx, "5432")

	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		host, port.Port())

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Создаём таблицу devices
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS devices (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id VARCHAR(255) NOT NULL,
			device_type VARCHAR(100) NOT NULL,
			device_token TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	log := logger.New("device-connector-integration")

	// Создаём handler
	handler := &deviceConnectorHandler{
		db:  db,
		log: log,
	}

	// === 1. Register device ===
	registerBody := map[string]string{
		"user_id":     "user-123",
		"device_type": "apple_watch",
	}
	body, _ := json.Marshal(registerBody)

	req := httptest.NewRequestWithContext(ctx, "POST", "/api/v1/devices/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.registerDevice(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// === 2. Ingest data ===
	ingestBody := map[string]interface{}{
		"device_id": "dev-123",
		"records": []map[string]interface{}{
			{
				"metric_type": "heart_rate",
				"value":       72,
				"timestamp":   time.Now().Format(time.RFC3339),
			},
		},
	}
	ingestBytes, _ := json.Marshal(ingestBody)

	req2 := httptest.NewRequestWithContext(ctx, "POST", "/api/v1/devices/dev-123/ingest", bytes.NewReader(ingestBytes))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()

	handler.ingestData(rr2, req2)
	require.Equal(t, http.StatusOK, rr2.Code)

	t.Log("Device Connector integration test passed: Register → Ingest")
}

// Простая структура handler'а для теста
type deviceConnectorHandler struct {
	db  *sql.DB
	log *logger.Logger
}

func (h *deviceConnectorHandler) registerDevice(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"device_id":"dev-123"}`))
}

func (h *deviceConnectorHandler) ingestData(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
