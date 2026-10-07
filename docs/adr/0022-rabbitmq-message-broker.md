# ADR 0022: Выбор RabbitMQ 4 как Message Broker

## Статус

Принято

## Контекст

FitPulse требует асинхронной коммуникации между микросервисами для:

- Уведомлений (email, push)
- Фоновой обработки биометрических данных
- Событийной синхронизации между сервисами
- ML-очередей (генерация планов, классификация)

Доступные варианты:

1. **RabbitMQ** — классический message broker с поддержкой AMQP 0.9.1, DLQ, publisher confirms.
2. **Apache Kafka** — distributed streaming platform, высокий throughput, сложнее эксплуатация.
3. **NATS** — лёгкий broker, но меньше экосистемы и community.

## Решение

Использовать **RabbitMQ 4** с классическими очередями (quorum queues запланированы на Phase 2):

1. **Durability**: persistent queues + mirrored queues для отказоустойчивости.
2. **DLQ**: `<queue-name>.dlq` для анализа ошибок.
3. **TTL**: 24 часа для сообщений уведомлений.
4. **Мониторинг**: queue depth, consumer lag, message rates через Prometheus.

## Последствия

- **Плюсы**: простая эксплуатация, богатый ecosystem, совместимость с AMQP 0.9.1, встроенные инструменты мониторинга.
- **Нейтрально**: требуется управление кластером RabbitMQ (в Phase 1 — single node).
- **Риски**: single node — single point of failure. Митигация: переход на quorum queues и clustered deployment в Phase 2.

## Рассмотренные альтернативы

- **Kafka**: слишком сложная конфигурация для текущего объёма данных, избыточный функционал.
- **NATS**: меньше community, меньше готовых решений для DLQ и мониторинга.

## Реализация

- `internal/queue/queue.go` — publisher/consumer abstraction.
- `internal/queue/dlq.go` — dead letter queue handling.
- `configs/k8s/base/deployments/rabbitmq.yaml` — Deployment + Service + PersistentVolumeClaim.
