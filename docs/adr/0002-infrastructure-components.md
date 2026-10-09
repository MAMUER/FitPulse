# ADR 0002: Выбор инфраструктурных компонентов — RabbitMQ, ELK Stack и Prometheus

## Статус

Принято

## Контекст

Проект требует надёжного асинхронного обмена сообщениями, централизованного логирования и сбора метрик для микросервисной архитектуры с высокой наблюдаемостью и отказоустойчивостью. Необходимо выбрать компоненты, которые production-ready, масштабируемы и хорошо интегрируются с Kubernetes.

## Решение

Будем использовать:

1. **RabbitMQ 4** — брокер сообщений для асинхронного взаимодействия микросервисов.
    - Назначение: очереди уведомлений, фоновой обработки биометрии, межсервисной синхронизации.
    - Требования: classic queues с DLQ (`<queue-name>.dlq`) через `internal/queue/dlq.go`, persistent storage в StatefulSet, мониторинг глубины очередей через Prometheus alert `RabbitMQQueueDepthHigh`. Quorum queues запланированы на Phase 2.

2. **Логирование**: Fluent Bit как DaemonSet для сбора stdout-логов (JSON Lines). ELK Stack (Elasticsearch 8.12, Logstash, Kibana) развёрнут в namespace `elk`. Fluent Bit отправляет логи в Logstash через network policy.
    - Retention: Elasticsearch hot tier + archive in S3 (3 года для compliance).

3. **Prometheus + Grafana + Alertmanager** — сбор, хранение, визуализация метрик и алерты.
    - Назначение: мониторинг здоровья сервисов, производительности и бизнес-метрик.
    - Service discovery через Kubernetes annotations; recording rules для SLO метрик (`job:request_error_rate`, `job:request_duration_p95`, `job:slo_error_budget_remaining`).
    - Alertmanager развёрнут с Telegram интеграцией (`telegram-notifications` receiver). Slack/PagerDuty/Grafana OnCall не настроены.

## Последствия

- **Плюсы**: надёжный messaging через RabbitMQ с DLQ; централизованное логирование через ELK Stack; полный стек observability с Prometheus/Grafana/Alertmanager/Telegram.
- **Минусы**: увеличивает сложность деплоя и поддержки, требуется экспертиза в этих инструментах.
- **Риски**: RabbitMQ single node — single point of failure (митигация: quorum queues и clustered deployment в Phase 2).

## Реализация

- RabbitMQ: StatefulSet с persistent volumes и TLS (self-signed CA через init container), classic queues + DLQ в коде (`internal/queue/dlq.go`). Quorum queues запланированы на Phase 2.
- Логирование: Fluent Bit DaemonSet в namespace `elk` (image: `fluent/fluent-bit:3.0.0`) отправляет логи в Logstash. ELK Stack развёрнут: Elasticsearch 8.12 StatefulSet, Logstash Deployment, Kibana Deployment.
- Мониторинг: Prometheus + Grafana + Alertmanager в namespace `monitoring`. Recording rules для SLO метрик в `configs/k8s/base/monitoring/prometheus-configmap.yaml`. Alertmanager с Telegram интеграцией в `configs/k8s/base/monitoring/alertmanager-config.yaml`.

## Рассмотренные альтернативы

- Kafka вместо RabbitMQ: более масштабируема для high-throughput, но RabbitMQ проще для нашего сценария.
- Loki + Grafana вместо ELK: легче для логов, но ELK предоставляет лучшие возможности поиска и анализа.
- Другие стеки мониторинга: Datadog, New Relic — но Prometheus является open-source и хорошо интегрируется с Kubernetes.
