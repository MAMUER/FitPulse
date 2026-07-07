# ADR 0002: Миграция с Redis на Valkey 9

## Статус

Принято

## Контекст

Redis 7.4+ перешёл на RSALv2 лицензии (Redis Source Available License), что ограничивает использование в open-source проектах. Valkey (valkey.io) — это fully open-source форк, совместимый с Redis API.

## Решение

Мигрировать с Redis на Valkey 9:

- Тот же протокол и API (redis-cli, go-redis клиент)
- Открытая лицензия (BSD-like)
- Активное сообщество и контрибуторы

## Последствия

- **Плюсы**: открытая лицензия, совместимость с существующим кодом
- **Риски**: новый дистрибутив (но от основателей Redis)

## Реализация

- `internal/testcontainers/`: `valkey/valkey:9-alpine` для smoke/integration тестов
- `configs/k8s/base/deployments`: `valkey/valkey:9-alpine` вместо `redis:7-alpine`
- `go.mod`: redis/go-redis совместим (Valkey — форк Redis без изменения протокола)
- Env vars (`REDIS_HOST`, `REDIS_PASSWORD`) сохранены для обратной совместимости с кодом