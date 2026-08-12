package queue

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

type mockChannel struct {
	declaredQueues []string
	declareError   error
	errorOnQueue   string
}

func (m *mockChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	if m.errorOnQueue != "" && name == m.errorOnQueue {
		return amqp.Queue{}, m.declareError
	}
	m.declaredQueues = append(m.declaredQueues, name)
	return amqp.Queue{Name: name}, nil
}

func (m *mockChannel) QueueDeclarePassive(name string) (amqp.Queue, bool, error) {
	return amqp.Queue{}, false, nil
}

func (m *mockChannel) QueueDelete(name string, ifUnused, ifEmpty bool) (int, error) {
	return 0, nil
}

func (m *mockChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return nil
}

func (m *mockChannel) QueueUnbind(name, key, exchange string, noWait bool) error {
	return nil
}

func (m *mockChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	return nil
}

func (m *mockChannel) ExchangeDeclarePassive(name, kind string) error {
	return nil
}

func (m *mockChannel) ExchangeDelete(name string, ifUnused bool) error {
	return nil
}

func (m *mockChannel) ExchangeBind(dest, key, source string, noWait bool, args amqp.Table) error {
	return nil
}

func (m *mockChannel) ExchangeUnbind(dest, key, source string, noWait bool) error {
	return nil
}

func (m *mockChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return nil
}

func (m *mockChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return nil, nil
}

func (m *mockChannel) Get(queue string, autoAck bool) (msg amqp.Delivery, ok bool, err error) {
	return amqp.Delivery{}, false, nil
}

func (m *mockChannel) Ack(tag uint64, multiple bool) error {
	return nil
}

func (m *mockChannel) Nack(tag uint64, multiple, requeue bool) error {
	return nil
}

func (m *mockChannel) Reject(tag uint64, requeue bool) error {
	return nil
}

func (m *mockChannel) Tx() error {
	return nil
}

func (m *mockChannel) Commit() error {
	return nil
}

func (m *mockChannel) Rollback() error {
	return nil
}

func (m *mockChannel) Confirm(noWait bool) error {
	return nil
}

func (m *mockChannel) NotifyPublish(chan amqp.Confirmation) chan amqp.Confirmation {
	return nil
}

func (m *mockChannel) NotifyReturn(chan amqp.Return) chan amqp.Return {
	return nil
}

func (m *mockChannel) NotifyCancel(chan string) chan string {
	return nil
}

func (m *mockChannel) NotifyClose(chan *amqp.Error) chan *amqp.Error {
	return nil
}

func (m *mockChannel) NotifyFlow(chan bool) chan bool {
	return nil
}

func (m *mockChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	return nil
}

func (m *mockChannel) Cancel(consumer string, noWait bool) error {
	return nil
}

func (m *mockChannel) Close() error {
	return nil
}

func TestDeclareQueueWithDLQ_Success(t *testing.T) {
	ch := &mockChannel{}
	err := DeclareQueueWithDLQ(ch, "test-queue")
	assert.NoError(t, err)
	assert.Len(t, ch.declaredQueues, 2)
	assert.Contains(t, ch.declaredQueues, "test-queue.dlq")
	assert.Contains(t, ch.declaredQueues, "test-queue")
}

func TestDeclareQueueWithDLQ_DLQDeclareError(t *testing.T) {
	ch := &mockChannel{errorOnQueue: "test-queue.dlq", declareError: assert.AnError}
	err := DeclareQueueWithDLQ(ch, "test-queue")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "declare DLQ")
}

func TestDeclareQueueWithDLQ_MainQueueDeclareError(t *testing.T) {
	ch := &mockChannel{errorOnQueue: "test-queue", declareError: assert.AnError}
	err := DeclareQueueWithDLQ(ch, "test-queue")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "declare main queue")
}

func TestDeclareQueueWithDLQ_DLQNaming(t *testing.T) {
	ch := &mockChannel{}
	err := DeclareQueueWithDLQ(ch, "my.nested.queue")
	assert.NoError(t, err)
	assert.Len(t, ch.declaredQueues, 2)
	assert.Equal(t, "my.nested.queue.dlq", ch.declaredQueues[0])
	assert.Equal(t, "my.nested.queue", ch.declaredQueues[1])
}
