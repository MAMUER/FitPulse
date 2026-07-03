# ADR 0015: Инфраструктура тестового окружения — TLS, health-check, защищённый HTTPS

## Статус

Принято

## Контекст

Тестовое окружение (`docker-compose.test.yml`) падало при старте Gateway из-за обязательной проверки TLS-переменных. Отсутствовали сертификаты, health-check использовал HTTP, хотя основной сервис ожидал HTTPS.

## Решение

Привести тестовое окружение к TLS-ready состоянию, но без реальных сертификатов в репозитории:

1. **Self-signed сертификаты**:
   - планировалась генерация через OpenSSL (`make certs`), но цель в Makefile не добавлена;
   - предполагаемый контур: CN=localhost, SAN=DNS:localhost,IP:127.0.0.1;
   - короткий TTL (1 день) чтобы случайно не попало в production.

2. **Docker Compose Test**:
   - планировался volume `/etc/tls/certs:ro`, но в текущем `docker-compose.test.yml` не смонтирован;
   - env-переменные `TLS_CERT_FILE`/`TLS_KEY_FILE` поддерживаются кодом Gateway, но не заданы в compose;
   - оба порта `8080` (HTTP redirect) и `8443` (HTTPS) задокументированы, но в текущем compose открыт только `8081:8080`;
   - health-check выполняется по HTTP (`http://localhost:8080/health`).

3. **Graceful TLS skip**:
   - Gateway стартует без паники при отсутствии TLS-переменных, что позволяет запускать unit-тесты без Docker.

## Последствия

- **Плюсы**: тестовый стенд стабильно поднимается одной командой; разработчики не зависят от внешних CA.
- **Нейтрально**: self-signed certs не годятся для production; их TTL предотвращает утечку в production.
- **Риски**: при ротации certs нужно не забывать обновить и volume в docker-compose.test.yml.

## Реализация

- `Makefile` — цель `certs` не добавлена.
- `docker-compose.test.yml` — env-переменные заданы, но volume и HTTPS-порт не открыты.
- `cmd/gateway/main.go` — логика graceful TLS fallback реализована.
