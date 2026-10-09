# ADR 0012: Инфраструктура тестового окружения — testcontainers-go, graceful TLS, изоляция окружений

## Статус

Принято

## Контекст

Локальный запуск сервисов поддерживается через `docker-compose.yml` для разработки. Production deploy происходит на VPS. Необходимо обеспечить:

## Решение

Локальный runtime поддерживается через `docker-compose.yml`. Интеграционные тесты используют `testcontainers-go` — библиотеку, которая поднимает эфемерные контейнеры прямо из тестов. Локальная разработка идёт через docker-compose, а не через testcontainers.

1. **Локальный development через docker-compose**:
    - `docker-compose.yml` поднимает все сервисы: gateway, user-service, biometric-service, training-service, classifier, device-aggregator, data-processor, ml-generator, Valkey, RabbitMQ, PostgreSQL, Jaeger, Prometheus, Grafana, frontend.
    - Gateway в docker-compose слушает HTTP на порту 8080, TLS не используется в локальной разработке.

2. **Testcontainers для зависимостей**:
    - PostgreSQL, Valkey, RabbitMQ запускаются автоматически через `testcontainers-go` в каждом integration-тесте.
    - Контейнеры живут ровно столько, сколько живёт тест; после завершения — удаляются.

3. **TLS в тестах**:
    - Self-signed сертификаты генерируются на лету (через `crypto/x509` + `crypto/rand`), без файлов на диске.
    - Gateway в integration-тестах стартует с TLS-сертификатом из памяти.
    - Health-check выполняется по HTTPS с отключённой проверкой CA (`InsecureSkipVerify` или кастомный `tls.Config`).

4. **Graceful TLS skip для unit-тестов**:
    - Gateway стартует без паники при отсутствии TLS-переменных, что позволяет запускать unit-тесты без контейнеров вообще.

5. **Изоляция окружений**:
    - Production/VPS окружение остаётся единственным целевым для production runtime с полноценным TLS.
    - Docker-compose используется для локальной разработки без TLS.
    - Testcontainers используется исключительно для автоматизированных integration-тестов.

## Последствия

- **Плюсы**: локальная разработка через docker-compose без TLS упрощает отладку; integration-тесты самодостаточны с self-signed certs; сертификаты не попадают в репозиторий.
- **Нейтрально**: требуется Docker Daemon для запуска тестов; локальный docker-compose работает по HTTP, production использует TLS.
- **Риски**: при изменении схемы БД нужно синхронизировать миграции в `db/migrations/V1__full_schema.sql` с тестовыми контейнерами.

## Реализация

- `docker-compose.yml` — локальный development runtime без TLS, HTTP на порту 8080.
- `go.mod` — добавлена зависимость `github.com/testcontainers/testcontainers-go`.
- `internal/testcontainers/` — helpers для поднятия PostgreSQL, Valkey, RabbitMQ контейнеров.
- Integration-тесты (`*_integration_test.go`) используют `testcontainers-go` с self-signed TLS.
- `cmd/gateway/main.go` — логика graceful TLS fallback реализована для unit-тестов без TLS-переменных.
