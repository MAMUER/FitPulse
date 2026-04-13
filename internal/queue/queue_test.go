// internal/queue/queue_test.go
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testRabbitURL = "amqp://guest:guest@localhost:5672/"
	testQueueName = "test_queue"
)

// ✅ В тестах используем интерфейс через переменную типа Publisher
func TestNewPublisher(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	url := testRabbitURL
	queueName := testQueueName

	pub, err := NewPublisher(url, queueName, zap.NewNop())
	if err != nil {
		t.Skip("RabbitMQ not available")
	}
	defer func() { _ = pub.Close() }()

	assert.NotNil(t, pub)
}

func TestNewPublisherInvalidURL(t *testing.T) {
	pub, err := NewPublisher("amqp://invalid:5672/", "test_queue", zap.NewNop())
	assert.Error(t, err)
	assert.Nil(t, pub)
}

func TestPublish(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test - use -short flag to skip")
	}

	url := testRabbitURL
	queueName := testQueueName

	pub, err := NewPublisher(url, queueName, zap.NewNop())
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}
	defer func() { _ = pub.Close() }()

	tests := []struct {
		name  string
		event interface{}
	}{
		{"simple map", map[string]interface{}{"test": "message", "id": 123}},
		{"struct", struct{ Name string }{"test"}},
		{"array", []string{"item1", "item2"}},
		{"complex nested", map[string]interface{}{
			"user_id": "user-123",
			"metrics": map[string]interface{}{
				"heart_rate": 72,
				"spo2":       98,
			},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := pub.Publish(ctx, tt.event)
			assert.NoError(t, err)
		})
	}
}

func TestNewConsumer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test - use -short flag to skip")
	}

	url := testRabbitURL
	queueName := testQueueName

	// Создаем publisher, чтобы очередь существовала
	pub, err := NewPublisher(url, queueName, zap.NewNop())
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}
	_ = pub.Close()

	consumer, err := NewConsumer(url, queueName, zap.NewNop())
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}
	defer func() { _ = consumer.Close() }()

	assert.NotNil(t, consumer)
	// ✅ Проверяем через публичный метод, а не внутреннее поле
	// Если в Consumer есть метод Messages(), проверяем его
}

func TestPublishAndConsume(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test - use -short flag to skip")
	}

	url := testRabbitURL
	queueName := testQueueName

	// Создаем publisher
	pub, err := NewPublisher(url, queueName, zap.NewNop())
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}
	defer func() { _ = pub.Close() }()

	// Создаем consumer
	consumer, err := NewConsumer(url, queueName, zap.NewNop())
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}
	defer func() { _ = consumer.Close() }()

	// Канал для получения сообщений через публичный метод
	// ✅ Используем публичный метод Messages() вместо доступа к полю msgs
	received := make(chan map[string]interface{}, 1)

	go func() {
		for msg := range consumer.Messages() {
			var data map[string]interface{}
			if umErr := json.Unmarshal(msg.Body, &data); umErr == nil {
				received <- data
				_ = msg.Ack(false)
			}
		}
	}()

	// Публикуем сообщение
	event := map[string]interface{}{
		"test": "consume",
		"id":   12345,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = pub.Publish(ctx, event)
	require.NoError(t, err)

	// Ждем сообщение
	select {
	case receivedEvent := <-received:
		assert.Equal(t, "consume", receivedEvent["test"])
		assert.Equal(t, float64(12345), receivedEvent["id"])
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestPublisherClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test - use -short flag to skip")
	}

	url := testRabbitURL
	queueName := testQueueName

	pub, err := NewPublisher(url, queueName, zap.NewNop())
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}

	err = pub.Close()
	assert.NoError(t, err)

	// Повторный close не должен вызывать ошибку
	err = pub.Close()
	assert.NoError(t, err)
}

func TestConsumerClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test - use -short flag to skip")
	}

	url := testRabbitURL
	queueName := testQueueName

	// Создаем publisher, чтобы очередь существовала
	pub, err := NewPublisher(url, queueName, zap.NewNop())
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}
	_ = pub.Close()

	consumer, err := NewConsumer(url, queueName, zap.NewNop())
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}

	err = consumer.Close()
	assert.NoError(t, err)

	// Повторный close не должен вызывать ошибку
	err = consumer.Close()
	assert.NoError(t, err)
}

// --- Unit tests for isClosedError (no RabbitMQ required) ---

func TestIsClosedErrorWithEOF(t *testing.T) {
	assert.True(t, isClosedError(io.EOF))
}

func TestIsClosedErrorWithAmqpErrClosed(t *testing.T) {
	assert.True(t, isClosedError(amqp.ErrClosed))
}

func TestIsClosedErrorWithWrappedEOF(t *testing.T) {
	wrapped := fmt.Errorf("wrapped: %w", io.EOF)
	assert.True(t, isClosedError(wrapped))
}

func TestIsClosedErrorWithWrappedAmqpErrClosed(t *testing.T) {
	wrapped := fmt.Errorf("wrapped: %w", amqp.ErrClosed)
	assert.True(t, isClosedError(wrapped))
}

func TestIsClosedErrorWithRegularError(t *testing.T) {
	assert.False(t, isClosedError(errors.New("regular error")))
}

func TestIsClosedErrorWithNil(t *testing.T) {
	assert.False(t, isClosedError(nil))
}

func TestPublisherInterface(t *testing.T) {
	var _ Publisher = (*rabbitPublisher)(nil)
}

func TestConsumerInterface(t *testing.T) {
	var _ Consumer = (*rabbitConsumer)(nil)
}
