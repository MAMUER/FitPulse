package main

import (
	"context"
	"testing"
	"time"

	"github.com/MAMUER/project/internal/db"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestDataProcessor_Integration_DatabaseConnection(t *testing.T) {
	ctx := context.Background()

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
		t.Skipf("Skipping data-processor integration test: could not start PostgreSQL: %v", err)
	}
	defer func() { _ = pgContainer.Terminate(ctx) }()

	host, _ := pgContainer.Host(ctx)
	port, _ := pgContainer.MappedPort(ctx, "5432")

	cfg := db.Config{
		Host:     host,
		Port:     port.Port(),
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	database, err := db.NewConnection(cfg)
	require.NoError(t, err)
	defer func() { _ = database.Close() }()

	var result int
	err = database.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	require.NoError(t, err)
	require.Equal(t, 1, result)

	t.Logf("Data Processor successfully connected to PostgreSQL at %s:%s", host, port.Port())
}
