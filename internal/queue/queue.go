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
	"go.uber.org/zap"

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

const defaultReconnectBackoff = 2 * time.Second

func ensureLogger(log *logger.Logger) *logger.Logger {
	if log == nil {
		return logger.New("queue")
	}
	return log
}

// rabbitPublisher — реализация Publisher
type rabbitPublisher struct {
	conn             *amqp.Connection
	channel          *amqp.Channel
	queue            string
	log              *logger.Logger
	metrics          *QueueMetrics
	mu               sync.RWMutex
	closed           bool
	url              string
	reconnectBackoff time.Duration
}

// rabbitConsumer — реализация Consumer
type rabbitConsumer struct {
	conn             *amqp.Connection
	channel          *amqp.Channel
	queue            string
	msgs             <-chan amqp.Delivery
	log              *logger.Logger
	mu               sync.RWMutex
	closed           bool
	url              string
	reconnectBackoff time.Duration
}

func dialAndDeclare(url, queueName string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err := DeclareQueueWithDLQ(ch, queueName); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return conn, ch, nil
}

// NewPublisher создаёт нового издателя
func NewPublisher(url, queueName string, log *logger.Logger, opts ...PublisherOption) (Publisher, error) {
	log = ensureLogger(log)

	conn, ch, err := dialAndDeclare(url, queueName)
	if err != nil {
		return nil, err
	}

	o := &publisherOptions{priority: "default"}
	for _, opt := range opts {
		opt(o)
	}

	return &rabbitPublisher{
		conn:             conn,
		channel:          ch,
		queue:            queueName,
		log:              log,
		metrics:          registerQueueMetrics(queueName, o.priority),
		url:              url,
		reconnectBackoff: defaultReconnectBackoff,
	}, nil
}

func (p *rabbitPublisher) reconnect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return errors.New("publisher is closed")
	}

	closeResourcesKeepClosedFlag(&p.mu, p.conn, p.channel)

	conn, ch, err := dialAndDeclare(p.url, p.queue)
	if err != nil {
		return fmt.Errorf("failed to reconnect publisher: %w", err)
	}

	p.conn = conn
	p.channel = ch
	p.log.Info("publisher reconnected", zap.String("queue", p.queue))
	return nil
}

func (c *rabbitConsumer) reconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errors.New("consumer is closed")
	}

	closeResourcesKeepClosedFlag(&c.mu, c.conn, c.channel)

	conn, ch, err := dialAndDeclare(c.url, c.queue)
	if err != nil {
		return fmt.Errorf("failed to reconnect consumer: %w", err)
	}

	if qosErr := ch.Qos(1, 0, false); qosErr != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("failed to set QoS on reconnect: %w", qosErr)
	}

	msgs, err := ch.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("failed to consume on reconnect: %w", err)
	}

	c.conn = conn
	c.channel = ch
	c.msgs = msgs
	c.log.Info("consumer reconnected", zap.String("queue", c.queue))
	return nil
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
		if isClosedError(err) {
			if reconnectErr := p.reconnect(); reconnectErr == nil {
				p.mu.RLock()
				ch = p.channel
				p.mu.RUnlock()
				if ch != nil {
					err = ch.PublishWithContext(ctx, "", p.queue, false, false, amqp.Publishing{
						ContentType:  "application/json",
						Body:         body,
						DeliveryMode: amqp.Persistent,
					})
				}
			}
		}
		if err != nil {
			return fmt.Errorf("failed to publish: %w", err)
		}
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
		if err := p.reconnect(); err != nil {
			return fmt.Errorf("publisher ping failed: reconnect error: %w", err)
		}
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

func closeResourcesKeepClosedFlag(mu *sync.RWMutex, conn *amqp.Connection, channel *amqp.Channel) {
	mu.Lock()
	defer mu.Unlock()
	if channel != nil {
		_ = channel.Close()
	}
	if conn != nil {
		_ = conn.Close()
	}
}

// NewConsumer создаёт нового потребителя
func NewConsumer(url, queueName string, log *logger.Logger, opts ...ConsumerOption) (Consumer, error) {
	log = ensureLogger(log)

	conn, ch, err := dialAndDeclare(url, queueName)
	if err != nil {
		return nil, err
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
		conn:             conn,
		channel:          ch,
		queue:            queueName,
		msgs:             msgs,
		log:              log,
		url:              url,
		reconnectBackoff: defaultReconnectBackoff,
	}, nil
}

func (c *rabbitConsumer) Messages() <-chan amqp.Delivery {
	c.mu.RLock()
	if c.msgs == nil {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	if c.channel != nil && c.channel.IsClosed() {
		go func() {
			if err := c.reconnect(); err != nil {
				c.log.Error("failed to reconnect consumer", zap.Error(err))
			}
		}()
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.msgs
}

func (c *rabbitConsumer) Ack(tag uint64, multiple bool) error {
	c.mu.RLock()
	if c.closed || c.channel == nil {
		c.mu.RUnlock()
		return errors.New("consumer is closed")
	}
	ch := c.channel
	c.mu.RUnlock()

	if ch.IsClosed() {
		return errors.New("consumer channel is closed")
	}
	return ch.Ack(tag, multiple)
}

func (c *rabbitConsumer) Nack(tag uint64, multiple, requeue bool) error {
	c.mu.RLock()
	if c.closed || c.channel == nil {
		c.mu.RUnlock()
		return errors.New("consumer is closed")
	}
	ch := c.channel
	c.mu.RUnlock()

	if ch.IsClosed() {
		return errors.New("consumer channel is closed")
	}
	return ch.Nack(tag, multiple, requeue)
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

func (c *rabbitConsumer) Channel() *amqp.Channel {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.channel
}
