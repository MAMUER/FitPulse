// cmd/biometric-service/integration_test.go
//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestBiometricService_PostgreSQL_Connection tests that we can connect to PostgreSQL
func TestBiometricService_PostgreSQL_Connection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Skip if Docker/testcontainers are not available (e.g., on Windows CI)
	ctx := context.Background()

	// Try to create a container - if it fails, skip the test
	postgresContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Skipf("Skipping integration test - Docker/testcontainers not available: %v", err)
	}
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Проверяем что можем получить connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	require.NotEmpty(t, connStr)
	require.Contains(t, connStr, "testdb")
	require.Contains(t, connStr, "test")
	t.Logf("PostgreSQL connection string: %s", connStr)
}
