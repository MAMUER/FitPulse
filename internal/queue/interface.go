// Package queue provides message queue interfaces and implementations.
package queue

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher — интерфейс для отправки сообщений в очередь
type Publisher interface {
	Publish(ctx context.Context, event interface{}) error
	Close() error
}

// Consumer — интерфейс для получения сообщений из очереди
type Consumer interface {
	Messages() <-chan amqp.Delivery
	Ack(tag uint64, multiple bool) error
	Nack(tag uint64, multiple, requeue bool) error
	Close() error
}
