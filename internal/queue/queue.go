package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/metrics"
)

// Prometheus метрики для очереди
var (
	queueMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queue_messages_total",
			Help: "Total number of messages published to queue",
		},
		[]string{"queue", "status"},
	)
)

// QueueMetrics ties queue/priority labels for depth reporting.
type QueueMetrics struct {
	queue    string
	priority string
}

func (m *QueueMetrics) Set(depth int) {
	if m == nil {
		return
	}
	metrics.NotificationQueueDepth.WithLabelValues(m.queue, m.priority).Set(float64(depth))
}

var queueMetricsRegistry sync.Map

func registerQueueMetrics(queue, priority string) *QueueMetrics {
	key := queue + "|" + priority
	if v, ok := queueMetricsRegistry.Load(key); ok {
		return v.(*QueueMetrics)
	}
	m := &QueueMetrics{queue: queue, priority: priority}
	queueMetricsRegistry.Store(key, m)
	return m
}

// ExportQueueDepth exports queue depth for consumer-side tracking.
func ExportQueueDepth(queue, priority string, depth int) {
	registerQueueMetrics(queue, priority).Set(depth)
}

// PublisherOption настраивает Publisher при создании.
type PublisherOption func(*publisherOptions)

type publisherOptions struct {
	priority string
}

// WithPublisherPriority задаёт приоритет очереди для метрик.
func WithPublisherPriority(priority string) PublisherOption {
	return func(o *publisherOptions) {
		o.priority = priority
	}
}

// ConsumerOption настраивает Consumer при создании.
type ConsumerOption func(*consumerOptions)

type consumerOptions struct {
	priority string
}

// WithConsumerPriority задаёт приоритет очереди для метрик.
func WithConsumerPriority(priority string) ConsumerOption {
	return func(o *consumerOptions) {
		o.priority = priority
	}
}

func ensureLogger(log *logger.Logger) *logger.Logger {
	if log == nil {
		return logger.New("queue")
	}
	return log
}

// rabbitPublisher — реализация Publisher
type rabbitPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
	log     *logger.Logger
	metrics *QueueMetrics
	mu      sync.RWMutex
	closed  bool
}

// rabbitConsumer — реализация Consumer
type rabbitConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
	msgs    <-chan amqp.Delivery
	log     *logger.Logger
	mu      sync.RWMutex
	closed  bool
}

// NewPublisher создаёт нового издателя
func NewPublisher(url, queueName string, log *logger.Logger, opts ...PublisherOption) (Publisher, error) {
	log = ensureLogger(log)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = DeclareQueueWithDLQ(ch, queueName)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	o := &publisherOptions{priority: "default"}
	for _, opt := range opts {
		opt(o)
	}

	return &rabbitPublisher{
		conn:    conn,
		channel: ch,
		queue:   queueName,
		log:     log,
		metrics: registerQueueMetrics(queueName, o.priority),
	}, nil
}

func (p *rabbitPublisher) Publish(ctx context.Context, event interface{}) error {
	p.mu.RLock()
	if p.closed || p.channel == nil {
		p.mu.RUnlock()
		return errors.New("publisher is closed")
	}
	ch := p.channel
	p.mu.RUnlock()

	body, err := json.Marshal(event)
	if err != nil {
		queueMessagesTotal.WithLabelValues(p.queue, "marshal_error").Inc()
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = ch.PublishWithContext(ctx, "", p.queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
	})

	if err != nil {
		queueMessagesTotal.WithLabelValues(p.queue, "publish_error").Inc()
		return fmt.Errorf("failed to publish: %w", err)
	}

	queueMessagesTotal.WithLabelValues(p.queue, "success").Inc()
	return nil
}

func (p *rabbitPublisher) Ping() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed || p.channel == nil {
		return errors.New("publisher is closed")
	}
	if p.conn.IsClosed() {
		return errors.New("connection is closed")
	}
	return nil
}

func (p *rabbitPublisher) Close() error {
	return closeResources(&p.mu, &p.closed, p.conn, p.channel)
}

func (c *rabbitConsumer) Close() error {
	return closeResources(&c.mu, &c.closed, c.conn, c.channel)
}

func closeResources(mu *sync.RWMutex, closed *bool, conn *amqp.Connection, channel *amqp.Channel) error {
	mu.Lock()
	if *closed {
		mu.Unlock()
		return nil
	}
	*closed = true
	mu.Unlock()

	var errs []error
	if channel != nil {
		if err := channel.Close(); err != nil && !isClosedError(err) {
			errs = append(errs, fmt.Errorf("channel: %w", err))
		}
	}
	if conn != nil {
		if err := conn.Close(); err != nil && !isClosedError(err) {
			errs = append(errs, fmt.Errorf("conn: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// NewConsumer создаёт нового потребителя
func NewConsumer(url, queueName string, log *logger.Logger, opts ...ConsumerOption) (Consumer, error) {
	log = ensureLogger(log)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = DeclareQueueWithDLQ(ch, queueName)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	if qosErr := ch.Qos(1, 0, false); qosErr != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", qosErr)
	}

	msgs, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to consume: %w", err)
	}

	o := &consumerOptions{priority: "default"}
	for _, opt := range opts {
		opt(o)
	}

	return &rabbitConsumer{
		conn:    conn,
		channel: ch,
		queue:   queueName,
		msgs:    msgs,
		log:     log,
	}, nil
}

func (c *rabbitConsumer) Messages() <-chan amqp.Delivery {
	return c.msgs
}

func (c *rabbitConsumer) Ack(tag uint64, multiple bool) error {
	return c.channel.Ack(tag, multiple)
}

func (c *rabbitConsumer) Nack(tag uint64, multiple, requeue bool) error {
	return c.channel.Nack(tag, multiple, requeue)
}

func isClosedError(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, amqp.ErrClosed)
}

// StartDepthReporter periodically updates NotificationQueueDepth for the consumer queue.
// It returns a stop function for graceful shutdown.
func StartDepthReporter(ctx context.Context, ch *amqp.Channel, queueName string, opts ...ConsumerOption) func() {
	if ch == nil || queueName == "" {
		return func() {
			// No-op stop function: channel or queue name is invalid,
			// no background goroutine was started.
		}
	}

	o := &consumerOptions{priority: "default"}
	for _, opt := range opts {
		opt(o)
	}

	m := registerQueueMetrics(queueName, o.priority)
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if ctx.Err() != nil {
					return
				}
				q, err := ch.QueueDeclarePassive(queueName, true, false, false, false, nil)
				if err != nil {
					continue
				}
				m.Set(int(q.Messages))
			case <-done:
				return
			}
		}
	}()

	return func() { close(done) }
}
