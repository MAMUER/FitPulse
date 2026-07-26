# ADR 0005: Реализация наблюдаемости — структурированное логирование, метрики Prometheus и алертинг

## Контекст

Система требует всесторонней наблюдаемости для мониторинга здоровья, производительности и безопасности микросервисов. Включает структурированное логирование, сбор метрик и автоматизированный алертинг для обеспечения надёжности и быстрого реагирования на инциденты.

## Решение

Реализовать наблюдаемость с:

1. **Структурированное JSON-логирование**: все сервисы должны логировать в JSON с обязательными полями:
   - timestamp (ISO8601 UTC)
   - level (DEBUG/INFO/WARN/ERROR/FATAL)
   - service (имя микросервиса)
   - correlationId (UUID для трейсинга запросов)
   - userId (string|null)
   - action (семантическое имя в UPPER_SNAKE_CASE)
   - дополнительные контекстные поля по необходимости.

2. **Метрики Prometheus**: обязательный набор метрик:
   - request_duration_seconds (Histogram)
   - error_total (Counter)
   - classification_confidence (Gauge для ML)
   - db_connection_pool_usage (Gauge)
   - notification_queue_depth (Gauge)
   - biometric_sync_lag_seconds (Gauge)

3. **Правила алертинга**: критические и предупреждающие алерты с политиками эскалации:
   - SEV-1: ServiceDown, DBConnectionPoolExhausted, BackupFailed
   - SEV-3: HighErrorRate, HighLatency, LowMLConfidence
   - Эскалация: Telegram-уведомления (через CI/CD бот) для SEV-1; Slack/PagerDuty-интеграции запланированы на Phase 2.

## Последствия

- Обеспечивает полную видимость в поведение и производительность системы.
- Позволяет проактивный мониторинг и быстрое реагирование на инциденты.
- Поддерживает compliance и операционные требования.

## Реализация

- **Структурированное JSON-логирование**: Go-сервисы (user-service, biometric-service, training-service, gateway, device-connector, data-processor) используют `internal/logger/logger.go` на базе zap с JSON-кодированием, ISO8601-таймстемпами и полем `service`. Gateway-middleware добавляет `correlationId`, `userId`, `action`. Пробелы: `cmd/classifier/main.go` использует сырой `zap.NewProduction()` без поля `service` и middleware; Python-сервис (`cmd/ml_generator/main.py`) использует `structlog` с `ConsoleRenderer` (человекочитаемый текст, не JSON) и не гарантирует все обязательные поля.
- Настройка Prometheus-экспортёров и Grafana-дашбордов: реализованы core и доменные метрики (`internal/metrics/metrics.go`, `internal/metrics/extended.go`).
- Настройка Alertmanager: развёрнут с базовым webhook-ресивером (`localhost:9093`) и закомментированным Telegram-примером. Интеграция со Slack/PagerDuty/Grafana OnCall запланирована на Phase 2.
- Реализация propagation correlation ID через сервисы: реализована в gateway-middleware.

## Рассмотренные альтернативы

- Неструктурированное логирование: сложнее для поиска и анализа.
- Меньше метрик: сниженная наблюдаемость.
- Ручной алертинг: более медленное реагирование.
