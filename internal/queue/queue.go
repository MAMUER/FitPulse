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
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/metrics"
)

// Prometheus метрики для очереди
var (
	queueMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queue_messages_total",
			Help: "Total number of messages published to queue",
		},
		[]string{"queue", "status"},
	)
)

func init() {
	prometheus.MustRegister(queueMessagesTotal)
}

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

// rabbitPublisher — реализация Publisher
type rabbitPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
	log     *zap.Logger
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
	log     *zap.Logger
	mu      sync.RWMutex
	closed  bool
}

// NewPublisher создаёт нового издателя
func NewPublisher(url, queueName string, logger *zap.Logger) (Publisher, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

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

	return &rabbitPublisher{
		conn:    conn,
		channel: ch,
		queue:   queueName,
		log:     logger,
		metrics: registerQueueMetrics(queueName, "default"),
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
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	var errs []error
	if p.channel != nil {
		if err := p.channel.Close(); err != nil && !isClosedError(err) {
			errs = append(errs, fmt.Errorf("channel: %w", err))
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil && !isClosedError(err) {
			errs = append(errs, fmt.Errorf("conn: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// NewConsumer создаёт нового потребителя
func NewConsumer(url, queueName string, logger *zap.Logger) (Consumer, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

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

	return &rabbitConsumer{
		conn:    conn,
		channel: ch,
		queue:   queueName,
		msgs:    msgs,
		log:     logger,
	}, nil
}

func (c *rabbitConsumer) Messages() <-chan amqp.Delivery {
	return c.msgs
}

func (c *rabbitConsumer) Ack(tag uint64, multiple bool) error {
	return fmt.Errorf("ack message: %w", c.channel.Ack(tag, multiple))
}

func (c *rabbitConsumer) Nack(tag uint64, multiple, requeue bool) error {
	return fmt.Errorf("nack message: %w", c.channel.Nack(tag, multiple, requeue))
}

func (c *rabbitConsumer) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	if c.channel != nil {
		err := c.channel.Close()
		if err != nil && !isClosedError(err) {
			c.log.Error("Channel close failed", zap.Error(err))
		}
	}
	if c.conn != nil {
		err := c.conn.Close()
		if err != nil && !isClosedError(err) {
			c.log.Error("Conn close failed", zap.Error(err))
		}
	}
	return nil
}

func isClosedError(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, amqp.ErrClosed)
}

// StartDepthReporter periodically updates NotificationQueueDepth for the consumer queue.
func StartDepthReporter(ctx context.Context, ch *amqp.Channel, queueName string) {
	if ch == nil || queueName == "" {
		return
	}
	m := registerQueueMetrics(queueName, "default")

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if ctx.Err() != nil {
				return
			}
			q, err := ch.QueueDeclarePassive(queueName, true, false, false, false, nil)
			if err != nil {
				continue
			}
			m.Set(int(q.Messages))
		}
	}()
}
