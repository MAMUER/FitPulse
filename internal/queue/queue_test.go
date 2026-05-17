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

// MockAMQPConnection is a mock for amqp.Connection
type MockAMQPConnection struct {
	shouldFail bool
	closed     bool
}

func (m *MockAMQPConnection) Channel() (*amqp.Channel, error) {
	if m.shouldFail {
		return nil, errors.New("mock connection failed")
	}
	return &amqp.Channel{}, nil
}

func (m *MockAMQPConnection) Close() error {
	if m.closed {
		return amqp.ErrClosed
	}
	m.closed = true
	return nil
}

func (m *MockAMQPConnection) IsClosed() bool {
	return m.closed
}

// MockAMQPChannel is a mock for amqp.Channel
type MockAMQPChannel struct {
	shouldFailQueueDeclare bool
	shouldFailQos          bool
	shouldFailConsume      bool
	shouldFailPublish      bool
	shouldFailClose        bool
	closed                 bool
}

func (m *MockAMQPChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	if m.shouldFailQueueDeclare {
		return amqp.Queue{}, errors.New("mock queue declare failed")
	}
	return amqp.Queue{Name: name}, nil
}

func (m *MockAMQPChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	if m.shouldFailQos {
		return errors.New("mock qos failed")
	}
	return nil
}

func (m *MockAMQPChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	if m.shouldFailConsume {
		return nil, errors.New("mock consume failed")
	}
	ch := make(chan amqp.Delivery, 1)
	return ch, nil
}

func (m *MockAMQPChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	if m.shouldFailPublish {
		return errors.New("mock publish failed")
	}
	return nil
}

func (m *MockAMQPChannel) Ack(tag uint64, multiple bool) error {
	return nil
}

func (m *MockAMQPChannel) Nack(tag uint64, multiple, requeue bool) error {
	return nil
}

func (m *MockAMQPChannel) Close() error {
	if m.shouldFailClose {
		return errors.New("mock channel close failed")
	}
	if m.closed {
		return amqp.ErrClosed
	}
	m.closed = true
	return nil
}

// Test NewPublisher with nil logger
func TestNewPublisherNilLogger(t *testing.T) {
	// This test will fail without RabbitMQ, but demonstrates the nil logger handling
	pub, err := NewPublisher("amqp://invalid", "test", nil)
	assert.Error(t, err)
	assert.Nil(t, pub)
}

// Test publisher Publish method with closed publisher
func TestPublisherPublishClosed(t *testing.T) {
	// Create a publisher instance (will fail, but we can test the closed logic)
	pub := &rabbitPublisher{closed: true}
	err := pub.Publish(context.Background(), map[string]string{"test": "data"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publisher is closed")
}

// Test publisher Publish method with marshal error
func TestPublisherPublishMarshalError(t *testing.T) {
	pub := &rabbitPublisher{
		channel: &amqp.Channel{}, // mock channel that doesn't fail
		queue:   "test",
		closed:  false,
	}

	// Use a type that can't be marshaled
	err := pub.Publish(context.Background(), make(chan int))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal event")
}

// Test consumer Messages method
func TestConsumerMessages(t *testing.T) {
	consumer := &rabbitConsumer{
		msgs: make(<-chan amqp.Delivery, 1),
	}
	ch := consumer.Messages()
	assert.NotNil(t, ch)
}

// Test consumer Ack method (requires real channel, will be integration test)
func TestConsumerAck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	consumer, err := NewConsumer(testRabbitURL, testQueueName, zap.NewNop())
	if err != nil {
		t.Skip("RabbitMQ not available")
	}
	defer func() { _ = consumer.Close() }()

	err = consumer.Ack(1, false)
	// Ack might fail if no message, but shouldn't panic
	assert.NoError(t, err)
}

// Test consumer Nack method (requires real channel, will be integration test)
func TestConsumerNack(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	consumer, err := NewConsumer(testRabbitURL, testQueueName, zap.NewNop())
	if err != nil {
		t.Skip("RabbitMQ not available")
	}
	defer func() { _ = consumer.Close() }()

	err = consumer.Nack(1, false, true)
	// Nack might fail if no message, but shouldn't panic
	assert.NoError(t, err)
}

// Test publisher Close with various error conditions
func TestPublisherCloseErrors(t *testing.T) {
	pub := &rabbitPublisher{
		closed: false,
	}

	// Close without channel/connection - should not error
	err := pub.Close()
	assert.NoError(t, err)

	// Close again - should not error
	err = pub.Close()
	assert.NoError(t, err)
}

// Test consumer Close with various error conditions
func TestConsumerCloseErrors(t *testing.T) {
	consumer := &rabbitConsumer{
		closed: false,
	}

	// Close without channel/connection - should not error
	err := consumer.Close()
	assert.NoError(t, err)

	// Close again - should not error
	err = consumer.Close()
	assert.NoError(t, err)
}

// Test publisher Publish after close
func TestPublisherPublishAfterClose(t *testing.T) {
	pub := &rabbitPublisher{
		closed: true,
	}
	err := pub.Publish(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publisher is closed")
}

// Test consumer methods on closed consumer
func TestConsumerMethodsOnClosed(t *testing.T) {
	consumer := &rabbitConsumer{
		closed: true,
		msgs:   make(<-chan amqp.Delivery, 1), // Initialize with a channel
	}

	// These methods don't check closed status, so they should not error
	ch := consumer.Messages()
	assert.NotNil(t, ch)
}

// Test NewConsumer with nil logger
func TestNewConsumerNilLogger(t *testing.T) {
	consumer, err := NewConsumer("amqp://invalid", "test", nil)
	assert.Error(t, err)
	assert.Nil(t, consumer)
}

// Additional unit test for publish error path simulation
func TestPublisherPublishErrorPaths(t *testing.T) {
	pub := &rabbitPublisher{
		closed: false,
	}
	// Force error by using invalid internal state (simulates channel failure)
	err := pub.Publish(context.Background(), nil)
	// Expect error due to nil channel
	assert.Error(t, err)
}

// Cover more Close branches
func TestPublisherCloseWithPartialState(t *testing.T) {
	pub := &rabbitPublisher{
		closed:  false,
		channel: nil,
		conn:    nil,
	}
	err := pub.Close()
	assert.NoError(t, err)
}

// Test Ack/Nack on nil channel would panic in real lib - covered by integration paths instead

func TestNewPublisher_NilLogger(t *testing.T) {
	_, err := NewPublisher(testRabbitURL, testQueueName, nil)
	assert.Error(t, err)
}
