// Package testcontainers provides helpers for spinning up
// ephemeral infrastructure containers in integration and smoke tests.
package testcontainers

import (
	"context"
	"fmt"
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

// retryStart runs fn until it succeeds or the attempts are exhausted.
// Container startup on shared CI runners can fail transit because the
// testcontainers reaper (Ryuk) takes longer than its connection timeout
// on the very first request of a session.
func retryStart[T any](t *testing.T, attempts int, name string, fn func() (T, error)) (T, error) {
	t.Helper()
	var zero T
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		res, err := fn()
		if err == nil {
			return res, nil
		}
		lastErr = err
		t.Logf("attempt %d/%d to start %s container failed: %v", attempt, attempts, name, err)
		if attempt < attempts {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
	}
	return zero, lastErr
}

// StartInfrastructure starts PostgreSQL, Valkey and RabbitMQ containers
// and returns a Container struct with connection details.
// Containers are automatically terminated when the test ends.
func StartInfrastructure(t *testing.T) *Container {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := retryStart(t, 3, "postgres", func() (*postgres.PostgresContainer, error) {
		return postgres.Run(ctx, "postgres:18-alpine",
			testcontainers.WithEnv(map[string]string{
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "testpass",
				"POSTGRES_DB":       "testdb",
			}),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).WithStartupTimeout(60*time.Second)),
		)
	})
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	valkeyContainer, err := retryStart(t, 3, "valkey", func() (*valkey.ValkeyContainer, error) {
		return valkey.Run(ctx, "valkey/valkey:9-alpine",
			testcontainers.WithWaitStrategy(
				wait.ForLog("Ready to accept connections").WithStartupTimeout(30*time.Second)),
		)
	})
	if err != nil {
		t.Fatalf("failed to start valkey container: %v", err)
	}

	rabbitContainer, err := retryStart(t, 3, "rabbitmq", func() (*rabbitmq.RabbitMQContainer, error) {
		return rabbitmq.Run(ctx, "rabbitmq:4.3-management-alpine",
			testcontainers.WithEnv(map[string]string{
				"RABBITMQ_DEFAULT_USER": "testuser",
				"RABBITMQ_DEFAULT_PASS": "testpass",
			}),
			testcontainers.WithWaitStrategy(
				wait.ForLog("Server startup complete").WithStartupTimeout(60*time.Second)),
		)
	})
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
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
		if err := valkeyContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate valkey container: %v", err)
		}
		if err := rabbitContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate rabbitmq container: %v", err)
		}
	})

	return &Container{
		Postgres:     pgContainer,
		Valkey:       valkeyContainer,
		RabbitMQ:     rabbitContainer,
		PostgresHost: pgHost,
		PostgresPort: mustParsePort(t, pgPort.Port()),
		ValkeyHost:   valkeyHost,
		ValkeyPort:   mustParsePort(t, valkeyPort.Port()),
		RabbitMQHost: rabbitHost,
		RabbitMQPort: mustParsePort(t, rabbitPort.Port()),
	}
}

// parsePort преобразует строковый порт контейнера в число.
func parsePort(s string) (int, error) {
	port, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid port %q: %w", s, err)
	}
	return port, nil
}

// mustParsePort возвращает порт, завершая тест при некорректном значении.
func mustParsePort(t *testing.T, s string) int {
	t.Helper()
	port, err := parsePort(s)
	if err != nil {
		t.Fatalf("%v", err)
	}
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
