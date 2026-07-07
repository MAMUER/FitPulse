# ADR 0014: Усиление безопасности Gateway и TLS-окружения

## Статус

Принято

## Контекст

После аудитов и SAST-сканирования (gosec) были выявлены уязвимости в конфигурации HTTP-серверов Gateway и подготовке окружения. Проблемы: отсутствие `ReadHeaderTimeout`, неподтверждённый редирект, отсутствие TLS в тестовом окружении (остановка контейнера).

## Решение

1. **HTTP-редиректер**:
   - Добавлен `ReadHeaderTimeout: 15 * time.Second` для защиты от Slowloris (G112).
   - Whitelist-проверка `r.URL.Host` при построении редиректа: только исходный `Host` + фиксированный порт 8443. Это нивелирует G710 (Open Redirect).

2. **HTTPS-сервер**:
   - Значения `ReadTimeout/WriteTimeout/IdleTimeout` выровнены.
   - При отсутствии `TLS_CERT_FILE`/`TLS_KEY_FILE` сервис переходит на `ListenAndServe()` без паники (test-friendly), но пишет warn-лог.

3. **Тестовое окружение**:
   - `testcontainers-go` для интеграционных и smoke-тестов.
   - Gateway в тестах стартует с TLS-сертификатом из памяти или graceful TLS fallback при отсутствии переменных.
   - Health-check в CI выполняется через `go test ./internal/testcontainers/...`.

## Последствия

- **Плюсы**: устранены два gosec-предупреждения критического уровня, тестовый стенд поднимается автоматически через testcontainers, не требуется ручной Docker Compose.
- **Нейтрально**: self-signed сертификаты только для dev/test; production продолжает использовать доверенные.
- **Риски**: self-signed certs не годятся для production; нужно явно отделить окружения.

## Реализация

- `cmd/gateway/main.go` — `ReadHeaderTimeout`, whitelist-host redirect, TLS fallback.
- `internal/testcontainers/` — helpers для PostgreSQL, Valkey, RabbitMQ.
