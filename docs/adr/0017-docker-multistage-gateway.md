# ADR 0017: Docker multi-stage сборка и Gateway как единая точка входа

## Статус

Принято

## Контекст

Проект состоит из Go backend (gateway + микросервисы) и React frontend. Для production требуется:

- единые минимальные образы без dev-зависимостей;
- сборка фронтенда внутри CI/CD;
- раздача статики и SPA fallback через gateway.

## Решение

Использовать Docker multi-stage builds для всех сервисов:

1. **Gateway** (`cmd/gateway/Dockerfile`) — 3 stage:
   - `node-builder`: собирает фронтенд (`web/dist/`)
   - `go-builder`: компилирует Go backend
   - `runtime`: минимальный Alpine с бинарником + статикой

2. **Остальные Go-сервисы** (`cmd/*/Dockerfile`) — 2 stage:
   - `builder`: компиляция Go
   - `runtime`: минимальный Alpine

Gateway слушает на порту 8080, раздаёт `web/dist/` через `http.FileServer`, все `/api/v1/*` запросы проксирует в backend-сервисы. `/confirm` отдаёт `web/dist/index.html` для hydrated React SPA.

## Последствия

- **Плюсы**: единые минимальные образы; фронтенд и бэкенд собираются изолированно; SPA fallback работает через один gateway.
- **Нейтрально**: Dockerfile сложнее, чем single-stage; требуется git в builder stage.
- **Риски**: при изменении только фронтенда пересобирается node-builder stage; можно оптимизировать кэшированием слоёв.

## Рассмотренные альтернативы

- **Отдельный nginx для статики**: больше компонент, отдельный деплой.
- **Сборка фронтенда на CI, загрузка артефакта**: сложнее артефакт-менеджмент.
- **Single-stage Dockerfile с node + go**: огромный образ, лишние dev-зависимости в runtime.

## Реализация

- `cmd/gateway/Dockerfile` — 3-stage: node-builder → go-builder → runtime
- `cmd/user-service/Dockerfile`, `cmd/biometric-service/Dockerfile`, `cmd/training-service/Dockerfile`, `cmd/classifier/Dockerfile`, `cmd/ml_generator/Dockerfile`, `cmd/device-aggregator/Dockerfile`, `cmd/data-processor/Dockerfile` — 2-stage: builder → runtime
- `cmd/gateway/main.go` — `http.FileServer` для `./web/dist/`, rewrite SPA routes
- `cmd/gateway/handlers_auth.go` — `/confirm` handler возвращает `web/dist/index.html`
- `.github/workflows/ci.yml` — `docker` job собирает и публикует образы через Buildx
- `scripts/ci/build-push.sh` — скрипт сборки образов для всех сервисов
