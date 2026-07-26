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

	"github.com/MAMUER/project/internal/logger"
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

	pub, err := NewPublisher(url, queueName, logger.New("test"), WithPublisherPriority("default"))
	if err != nil {
		t.Skip("RabbitMQ not available")
	}
	defer func() { _ = pub.Close() }()

	assert.NotNil(t, pub)
}

func TestNewPublisherInvalidURL(t *testing.T) {
	pub, err := NewPublisher("amqp://invalid:5672/", "test_queue", logger.New("test"))
	assert.Error(t, err)
	assert.Nil(t, pub)
}

func TestPublish(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test - use -short flag to skip")
	}

	url := testRabbitURL
	queueName := testQueueName

	pub, err := NewPublisher(url, queueName, logger.New("test"))
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

	pub, err := NewPublisher(url, queueName, logger.New("test"))
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}
	_ = pub.Close()

	consumer, err := NewConsumer(url, queueName, logger.New("test"))
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

	pub, err := NewPublisher(url, queueName, logger.New("test"))
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}
	defer func() { _ = pub.Close() }()

	consumer, err := NewConsumer(url, queueName, logger.New("test"))
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

	event := map[string]interface{}{
		"test": "consume",
		"id":   12345,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = pub.Publish(ctx, event)
	require.NoError(t, err)

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

	pub, err := NewPublisher(url, queueName, logger.New("test"))
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}

	err = pub.Close()
	assert.NoError(t, err)

	err = pub.Close()
	assert.NoError(t, err)
}

func TestConsumerClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test - use -short flag to skip")
	}

	url := testRabbitURL
	queueName := testQueueName

	pub, err := NewPublisher(url, queueName, logger.New("test"))
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}
	_ = pub.Close()

	consumer, err := NewConsumer(url, queueName, logger.New("test"))
	if err != nil {
		t.Skip("RabbitMQ not available, skipping test")
	}

	err = consumer.Close()
	assert.NoError(t, err)

	err = consumer.Close()
	assert.NoError(t, err)
}

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

func TestNewPublisherNilLogger(t *testing.T) {
	pub, err := NewPublisher("amqp://invalid", "test", nil)
	assert.Error(t, err)
	assert.Nil(t, pub)
}

func TestPublisherPublishClosed(t *testing.T) {
	pub := &rabbitPublisher{closed: true}
	err := pub.Publish(context.Background(), map[string]string{"test": "data"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publisher is closed")
}

func TestPublisherPublishMarshalError(t *testing.T) {
	pub := &rabbitPublisher{
		channel: &amqp.Channel{},
		queue:   "test",
		closed:  false,
	}

	err := pub.Publish(context.Background(), make(chan int))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal event")
}

func TestConsumerMessages(t *testing.T) {
	consumer := &rabbitConsumer{
		msgs: make(<-chan amqp.Delivery, 1),
	}
	ch := consumer.Messages()
	assert.NotNil(t, ch)
}

func TestConsumerAck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	consumer, err := NewConsumer(testRabbitURL, testQueueName, logger.New("test"))
	if err != nil {
		t.Skip("RabbitMQ not available")
	}
	defer func() { _ = consumer.Close() }()

	err = consumer.Ack(1, false)
	assert.NoError(t, err)
}

func TestConsumerNack(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	consumer, err := NewConsumer(testRabbitURL, testQueueName, logger.New("test"))
	if err != nil {
		t.Skip("RabbitMQ not available")
	}
	defer func() { _ = consumer.Close() }()

	err = consumer.Nack(1, false, true)
	assert.NoError(t, err)
}

func TestPublisherCloseErrors(t *testing.T) {
	pub := &rabbitPublisher{
		closed: false,
	}

	err := pub.Close()
	assert.NoError(t, err)

	err = pub.Close()
	assert.NoError(t, err)
}

func TestConsumerCloseErrors(t *testing.T) {
	consumer := &rabbitConsumer{
		closed: false,
	}

	err := consumer.Close()
	assert.NoError(t, err)

	err = consumer.Close()
	assert.NoError(t, err)
}

func TestPublisherPublishAfterClose(t *testing.T) {
	pub := &rabbitPublisher{
		closed: true,
	}
	err := pub.Publish(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publisher is closed")
}

func TestConsumerMethodsOnClosed(t *testing.T) {
	consumer := &rabbitConsumer{
		closed: true,
		msgs:   make(<-chan amqp.Delivery, 1),
	}

	ch := consumer.Messages()
	assert.NotNil(t, ch)
}

func TestNewConsumerNilLogger(t *testing.T) {
	consumer, err := NewConsumer("amqp://invalid", "test", nil)
	assert.Error(t, err)
	assert.Nil(t, consumer)
}

func TestPublisherPublishErrorPaths(t *testing.T) {
	pub := &rabbitPublisher{
		closed: false,
	}
	err := pub.Publish(context.Background(), nil)
	assert.Error(t, err)
}

func TestPublisherCloseWithPartialState(t *testing.T) {
	pub := &rabbitPublisher{
		closed:  false,
		channel: nil,
		conn:    nil,
	}
	err := pub.Close()
	assert.NoError(t, err)
}

func TestNewPublisher_NilLogger(t *testing.T) {
	_, err := NewPublisher(testRabbitURL, testQueueName, nil)
	assert.Error(t, err)
}

func TestIsClosedError_MoreCases(t *testing.T) {
	assert.False(t, isClosedError(errors.New("random error")))
	assert.False(t, isClosedError(nil))
}

func TestPublisherInterface_MoreCoverage(t *testing.T) {
	var _ Publisher = (*rabbitPublisher)(nil)
}

func TestConsumerInterface_MoreCoverage(t *testing.T) {
	var _ Consumer = (*rabbitConsumer)(nil)
}
