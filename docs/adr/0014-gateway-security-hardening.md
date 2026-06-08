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
   - В `docker-compose.test.yml` смонтированы self-signed сертификаты:
     - `deploy/tls/certs/server.crt` и `server.key` (1 день, CN=localhost, SAN=localhost,127.0.0.1)
   - Добавлены env-переменные `TLS_CERT_FILE`/`TLS_KEY_FILE`.
   - Health-check использован через HTTPS с флагом `-k`.

## Последствия

- **Плюсы**: устранены два gosec-предупреждения критического уровня, тестовый стенд поднимается без ручного вмешательства.
- **Нейтрально**: self-signed сертификаты только для dev/test; production продолжает использовать доверенные.
- **Риски**: self-signed certs не годятся для production; нужно явно отделить окружения.

## Реализация

- `cmd/gateway/main.go` — `ReadHeaderTimeout`, whitelist-host redirect, TLS fallback.
- `docker-compose.test.yml` — volumes, env vars, HTTPS healthcheck.
- `deploy/tls/certs/` — сгенерированные тестовые ключи/сертификаты.
