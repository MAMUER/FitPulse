// Package testcontainers provides helpers for spinning up
// ephemeral infrastructure containers in integration and smoke tests.
package testcontainers

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"github.com/testcontainers/testcontainers-go/modules/valkey"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Container holds connection info for started infrastructure containers.
type Container struct {
	Postgres *postgres.PostgresContainer
	Valkey   *valkey.ValkeyContainer
	RabbitMQ *rabbitmq.RabbitMQContainer

	PostgresHost string
	PostgresPort int
	ValkeyHost   string
	ValkeyPort   int
	RabbitMQHost string
	RabbitMQPort int
}

// StartInfrastructure starts PostgreSQL, Valkey and RabbitMQ containers
// and returns a Container struct with connection details.
// Containers are automatically terminated when the test ends.
func StartInfrastructure(t *testing.T) *Container {
	t.Helper()
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

	valkeyContainer, err := valkey.Run(ctx, "valkey/valkey:9-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start valkey container: %v", err)
	}

	rabbitContainer, err := rabbitmq.Run(ctx, "rabbitmq:4.3-management-alpine",
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

	pgHost, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get postgres host: %v", err)
	}
	pgPort, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get postgres port: %v", err)
	}

	valkeyHost, err := valkeyContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get valkey host: %v", err)
	}
	valkeyPort, err := valkeyContainer.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get valkey port: %v", err)
	}

	rabbitHost, err := rabbitContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get rabbitmq host: %v", err)
	}
	rabbitPort, err := rabbitContainer.MappedPort(ctx, "5672")
	if err != nil {
		t.Fatalf("failed to get rabbitmq port: %v", err)
	}

	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
		_ = valkeyContainer.Terminate(ctx)
		_ = rabbitContainer.Terminate(ctx)
	})

	return &Container{
		Postgres:     pgContainer,
		Valkey:       valkeyContainer,
		RabbitMQ:     rabbitContainer,
		PostgresHost: pgHost,
		PostgresPort: parsePort(pgPort.Port()),
		ValkeyHost:   valkeyHost,
		ValkeyPort:   parsePort(valkeyPort.Port()),
		RabbitMQHost: rabbitHost,
		RabbitMQPort: parsePort(rabbitPort.Port()),
	}
}

func parsePort(s string) int {
	port, _ := strconv.Atoi(s)
	return port
}

// ResolveHost resolves a container host to an IP address suitable for connecting
// from the host to a container. On Docker Desktop (Windows/macOS) the host may
// need special handling.
func ResolveHost(t *testing.T, host string) string {
	t.Helper()
	if host == "localhost" || host == "127.0.0.1" {
		return host
	}
	ips, err := net.LookupHost(host)
	if err == nil && len(ips) > 0 {
		return ips[0]
	}
	return host
}
