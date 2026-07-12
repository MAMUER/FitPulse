//go:build smoke

package testcontainers_test

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/testcontainers"
)

func TestInfrastructureSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping smoke test")
	}

	infra := testcontainers.StartInfrastructure(t)
	require.NotNil(t, infra)
	require.NotEmpty(t, infra.PostgresHost)
	require.NotEmpty(t, infra.ValkeyHost)
	require.NotEmpty(t, infra.RabbitMQHost)

	t.Run("PostgreSQL is reachable", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		conn, err := infra.Postgres.ConnectionString(ctx)
		require.NoError(t, err)
		require.Contains(t, conn, "postgres://")
	})

	t.Run("Valkey is reachable", func(t *testing.T) {
		host := testcontainers.ResolveHost(t, infra.ValkeyHost)
		address := host + ":" + strconv.Itoa(infra.ValkeyPort)

		conn, err := net.DialTimeout("tcp", address, 5*time.Second)
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	})

	t.Run("RabbitMQ is reachable", func(t *testing.T) {
		address := infra.RabbitMQHost + ":" + strconv.Itoa(infra.RabbitMQPort)
		conn, err := net.DialTimeout("tcp", address, 5*time.Second)
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	})
}
