# ADR 0004: Реализация наблюдаемости — структурированное логирование, метрики Prometheus и алертинг

## Статус

Принято

## Контекст

Система требует всесторонней наблюдаемости для мониторинга здоровья, производительности и безопасности микросервисов. Включает структурированное логирование, сбор метрик и автоматизированный алертинг для обеспечения надёжности и быстрого реагирования на инциденты.

## Решение

Реализовать наблюдаемость с:

1. **Структурированное JSON-логирование**: все Go-сервисы должны логировать в JSON с обязательными полями:
   - timestamp (ISO8601 UTC)
   - level (DEBUG/INFO/WARN/ERROR/FATAL)
   - service (имя микросервиса)
   - correlationId (UUID для трейсинга запросов)
   - userId (string|null)
   - action (семантическое имя в UPPER_SNAKE_CASE)
   - дополнительные контекстные поля по необходимости.

2. **Метрики Prometheus**: обязательный набор метрик:
   - `http_request_duration_seconds` (Histogram)
   - `error_total` (Counter)
   - `classification_confidence` (Gauge для ML)
   - `db_connection_pool_usage` (Gauge)
   - `notification_queue_depth` (Gauge)
   - `biometric_sync_lag_seconds` (Gauge)

3. **Правила алертинга**: критические и предупреждающие алерты с политиками эскалации:
   - SEV-1: ServiceDown, DBConnectionPoolExhausted, BackupFailed
   - SEV-3: HighErrorRate, HighLatency, LowMLConfidence
   - Эскалация: Telegram-уведомления для SEV-1; Slack/PagerDuty/Grafana OnCall не настроены.

## Последствия

- Обеспечивает полную видимость в поведение и производительность системы.
- Позволяет проактивный мониторинг и быстрое реагирование на инциденты.
- Поддерживает compliance и операционные требования.

## Реализация

- **Структурированное JSON-логирование**: Go-сервисы используют `internal/logger/logger.go` на базе zap с JSON-кодированием, ISO8601-таймстемпами и полем `service`. Gateway-middleware добавляет `correlationId`, `userId`, `action` через `internal/middleware/middleware.go`.
- **Prometheus-экспортёры и Grafana-дашборды**: реализованы core и доменные метрики (`internal/metrics/metrics.go`, `internal/metrics/extended.go`).
- **Alertmanager**: развёрнут с Telegram-интеграцией (`telegram-notifications` receiver) в `configs/k8s/base/monitoring/alertmanager-config.yaml`. Slack/PagerDuty/Grafana OnCall receivers добавлены в конфиг, но не настроены (требуют credentials).
- **Propagation correlation ID**: реализована в gateway-middleware через `internal/middleware/middleware.go` и `internal/middleware/context_keys.go`.

## Рассмотренные альтернативы

- Неструктурированное логирование: сложнее для поиска и анализа.
- Меньше метрик: сниженная наблюдаемость.
- Ручной алертинг: более медленное реагирование.
