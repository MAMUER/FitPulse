# ADR 0015: Инфраструктура тестового окружения — TLS, health-check, защищённый HTTPS

## Статус

Принято

## Контекст

Тестовое окружение (`docker-compose.test.yml`) падало при старте Gateway из-за обязательной проверки TLS-переменных. Отсутствовали сертификаты, health-check использовал HTTP, хотя основной сервис ожидал HTTPS.

## Решение

Привести тестовое окружение к требованиям production-grade, но с self-signed инфраструктурой:

1. **Self-signed сертификаты** в `deploy/tls/certs/`:
   - генерация через OpenSSL (`make certs`);
   - CN=localhost, SAN=DNS:localhost,IP:127.0.0.1;
   - короткий TTL (1 день) чтобы случайно не попало в production.

2. **Docker Compose Test**:
   - монтировать директорию с сертификатами в `/etc/tls/certs:ro`;
   - передавать `TLS_CERT_FILE` и `TLS_KEY_FILE`;
   - открывать оба порта: `8080` (HTTP redirect) и `8443` (HTTPS);
   - health-check через `curl -f -k https://localhost:8443/health`.

3. **Graceful TLS skip**:
   - Gateway продолжит стартовать, даже если переменных нет (логирует warn), что позволяет запускать unit-тесты без Docker.

## Последствия

- **Плюсы**: тестовый стенд стабильно поднимается одной командой; разработчики не зависят от внешних CA.
- **Нейтрально**: self-signed certs не годятся для production; их TTL предотвращает утечку в production.
- **Риски**: при ротации certs нужно не забывать обновить и volume в docker-compose.test.yml.

## Реализация

- `Makefile` — цель `certs`.
- `docker-compose.test.yml` — volumes/env/ports/healthcheck изменения.
- `cmd/gateway/main.go` — логика fallback’а.
