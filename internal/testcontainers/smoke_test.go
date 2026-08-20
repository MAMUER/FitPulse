//go:build smoke

package testcontainers_test

import (
	"context"
	"database/sql"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/testcontainers"
	_ "github.com/lib/pq"
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

	t.Run("PostgreSQL can execute queries", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		connStr, err := infra.Postgres.ConnectionString(ctx)
		require.NoError(t, err)

		db, err := sql.Open("postgres", connStr+"?sslmode=disable")
		require.NoError(t, err)
		defer db.Close()

		var result int
		err = db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
		require.NoError(t, err)
		require.Equal(t, 1, result)
	})

	t.Run("Valkey is reachable", func(t *testing.T) {
		host := testcontainers.ResolveHost(t, infra.ValkeyHost)
		address := host + ":" + strconv.Itoa(infra.ValkeyPort)

		conn, err := net.DialTimeout("tcp", address, 5*time.Second)
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	})

	t.Run("Valkey can set and get keys", func(t *testing.T) {
		host := testcontainers.ResolveHost(t, infra.ValkeyHost)
		address := host + ":" + strconv.Itoa(infra.ValkeyPort)

		client := redis.NewClient(&redis.Options{
			Addr:     address,
			Password: "",
			DB:       0,
		})
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := client.Set(ctx, "smoke-test-key", "smoke-test-value", 0).Err()
		require.NoError(t, err)

		val, err := client.Get(ctx, "smoke-test-key").Result()
		require.NoError(t, err)
		require.Equal(t, "smoke-test-value", val)
	})

	t.Run("RabbitMQ is reachable", func(t *testing.T) {
		address := infra.RabbitMQHost + ":" + strconv.Itoa(infra.RabbitMQPort)
		conn, err := net.DialTimeout("tcp", address, 5*time.Second)
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	})

	t.Run("RabbitMQ can publish and consume messages", func(t *testing.T) {
		address := infra.RabbitMQHost + ":" + strconv.Itoa(infra.RabbitMQPort)
		url := "amqp://testuser:testpass@" + address + "/"

		conn, err := amqp091.Dial(url)
		require.NoError(t, err)
		defer conn.Close()

		ch, err := conn.Channel()
		require.NoError(t, err)
		defer ch.Close()

		queue, err := ch.QueueDeclare("smoke-test-queue", true, false, false, false, nil)
		require.NoError(t, err)

		body := []byte("smoke-test-message")
		err = ch.Publish("", queue.Name, false, false, amqp091.Publishing{
			Body: body,
		})
		require.NoError(t, err)

		msgs, err := ch.Consume(queue.Name, "", true, false, false, false, nil)
		require.NoError(t, err)

		select {
		case msg := <-msgs:
			require.Equal(t, body, msg.Body)
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for message from RabbitMQ")
		}
	})
}
