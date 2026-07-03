package queue

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DeclareQueueWithDLQ создаёт очередь с настройкой Dead Letter Exchange
func DeclareQueueWithDLQ(ch *amqp.Channel, queueName string) error {
	dlqName := queueName + ".dlq"

	// 1. Создаём DLQ
	if _, err := ch.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare DLQ: %w", err)
	}

	// 2. Аргументы для основной очереди
	args := amqp.Table{
		"x-dead-letter-exchange":    "",       // Используем default exchange
		"x-dead-letter-routing-key": dlqName,  // Маршрутизация в DLQ
		"x-message-ttl":             86400000, // 24 часа в мс
		"x-max-length":              10000,    // Лимит сообщений (защита от переполнения)
	}

	// 3. Создаём основную очередь с DLX
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare main queue: %w", err)
	}

	return nil
}
