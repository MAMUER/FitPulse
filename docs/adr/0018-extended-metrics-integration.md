# ADR-0018: Интеграция расширенных Prometheus-метрик

## Статус

Принято

## Контекст

Система требовала единой наблюдаемости по ключевым бизнес-путиям: ML-классификация, пул PostgreSQL, RabbitMQ очереди и задержка синхронизации биометрических данных. Ранее метрики были разбросаны, частично отсутствовали, часть объявлений дублировалась между файлами. Это мешало и сборке, и операционному дашбордингу.

## Решение

Разделить метрики на два файла в пакете `internal/metrics`:

- `metrics.go` — core HTTP/Prometheus метрики (`RequestsTotal`, `RequestDuration`, `ActiveRequests`, `ErrorTotal`).
- `extended.go` — доменные метрики:
  - `ClassificationConfidence` (`model_version`, `class`)
  - `DBConnectionPoolUsage` (`service`, `pool_name`)
  - `NotificationQueueDepth` (`queue_name`, `priority`)
  - `BiometricSyncLagSeconds` (`device_type`, `user_segment`)

### ML-сервисы

В `cmd/classifier/main.go` и `cmd/ml_generator/main.py` добавлен Gauge-вектор `classification_confidence` через `prometheus_client`. После успешной классификации/генерации происходит `labels(model_version, class).set(confidence)`.

### База данных

В `internal/db/db.go` используется `prometheus.NewGaugeFunc` для ленивого сбора `db.Stats()` при скрейпинге Prometheus, что исключает фоновые горутины и таймеры.

```text
usage = InUse / max(MaxOpenConnections, 1)
```

### RabbitMQ

В `internal/queue/queue.go`:

- добавлен `QueueMetrics` + `registerQueueMetrics()` кэш по `(queue, priority)`;
- `StartDepthReporter()` раз в 10 секунд через `QueueDeclarePassive` обновляет gauge глубины очереди.

### Biometric service

В `cmd/biometric-service/main.go` в `AddRecord` замеряется `time.Now()` до и после INSERT, после записи вызывается:

```text
metrics.BiometricSyncLagSeconds.WithLabelValues(req.DeviceType, "default").Set(lag)
```

## Последствия

- Метрики начинают собираться сразу после старта, без ручной конфигурации.
- В Grafana/Prometheus появляются 4 новых временных ряда, которые покрывают 5 блоков требований.
- Дубли метрик устранены: одна метрика = одна переменная = один файл.
- Device Aggregator вынесен из `main.go`, добавлены standalone `aggregator.go` и `webhooks.go`.

## Реализация

- `internal/metrics/metrics.go`
- `internal/metrics/extended.go`
- `internal/db/db.go`
- `internal/queue/queue.go`
- `cmd/biometric-service/main.go`
- `cmd/classifier/main.go`
- `cmd/ml_generator/main.py`
- `cmd/device-aggregator/aggregator.go`
- `cmd/device-aggregator/webhooks.go`
- `cmd/device-aggregator/main.go`

## Рассмотренные альтернативы

- Оставить все метрики в одном файле — рост конфликтов имён при добавлении новых доменных метрик.
- Сделать отдельный пакет `metrics/ml`, `metrics/db` — избыточная модульность для текущего объёма (4 метрики).
- Мониторить lag через external exporter polling DB — выше latency, больше нагрузки на БД.
