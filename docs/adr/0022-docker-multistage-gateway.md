# ADR 0022: Docker multi-stage сборка и Gateway как единая точка входа

## Статус

Принято

## Контекст

Проект состоит из Go backend (gateway) и React frontend. Для production требуется:
- единый production-образ без dev-зависимостей;
- минимальный размер итогового образа;
- сборка фронтенда внутри CI/CD, а не на хост-машине;
- раздача статики и SPA fallback через один gateway-сервис;
- поддержка `/confirm` страницы email-подтверждения через React SPA.

## Решение

Использовать Docker multi-stage build в `cmd/gateway/Dockerfile`:

1. **Stage `node-builder`**: `node:26-alpine` — клонирует репозиторий, устанавливает зависимости, собирает фронтенд (`npm run build`), артефакт — `web/dist/`.
2. **Stage `go-builder`**: `golang:1.26-alpine` — копирует `go.mod/go.sum`, качает зависимости, компилирует Go backend.
3. **Stage `runtime`**: `alpine:3.24` — содержит только runtime-зависимости (ca-certificates, tzdata), копирует бинарный файл gateway и `web/dist/` из builder-стадий.
4. **Gateway** слушает на порту 8080, раздаёт `web/dist/` через `http.FileServer`, все `/api/v1/*` запросы проксирует в backend-сервисы.
5. **Email confirm**: `/confirm` отдаёт `web/dist/index.html` для hydrated React SPA, а не отдельный HTML-шаблон.

## Последствия

- **Плюсы**: единый production-образ без node_modules; минимальный размер; фронтенд и бэкенд собираются изолированно.
- **Плюсы**: SPA fallback работает через один gateway; не нужен отдельный nginx/static-сервер.
- **Нейтрально**: Dockerfile сложнее, чем single-stage; требуется git в node-builder stage.
- **Риски**: при изменении только фронтенда пересобирается node-builder stage; можно оптимизировать кэшированием слоёв.

## Рассмотренные альтернативы

- **Отдельный nginx для статики**: больше компонент, отдельный деплой, дополнительные health-checks.
- **Сборка фронтенда на CI, загрузка артефакта**: сложнее артефакт-менеджмент.
- **Single-stage Dockerfile с node + go**: огромный образ, лишние dev-зависимости в runtime.

## Реализация

- `cmd/gateway/Dockerfile` — multi-stage build (node-builder → go-builder → runtime).
- `cmd/gateway/main.go` — `http.FileServer` для `./web/dist/`, rewrite SPA routes.
- `cmd/gateway/handlers_auth.go` — `/confirm` handler возвращает `web/dist/index.html`.
- `.github/workflows/ci.yml` — `docker` job собирает и публикует образ.
- `Makefile` — `docker-build`, `docker-push` targets.
