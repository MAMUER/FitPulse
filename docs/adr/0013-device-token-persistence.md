# ADR 0013: Token Persistence и Audit Trail для носимых устройств

## Статус

Принято

## Контекст

Обеспечить персистентное хранение OAuth-токенов, учёт состояния синхронизации и audit trail операций с устройствами. Прямое хранение токенов в Redis/Valkey недостаточно надёжно и не покрывает сценарии rollover, отзыва доступа и аудита.

## Решение

Добавить в схему БД миграцию `V11__device_providers.sql` с тремя таблицами:

1. **`oauth_states`** — временные CSRF-токены OAuth, `expires_at + 10 мин`, с индексом по `expires_at`.
2. **`device_provider_accounts`** — учётные записи OAuth-провайдеров:
   - `user_id`, `provider`, `provider_user_id`
   - `access_token`, `refresh_token`, `token_expires_at`
   - `scopes TEXT[]`, `webhook_subscription_id`
   - `last_sync_at`, `is_active`, мета-поля `created_at/updated_at`
   - уникальность `(user_id, provider)`, `ON CONFLICT... DO UPDATE`
3. **`device_sync_log`** — лог синхронизаций:
   - `provider_account_id`, `sync_type`, `records_count`
   - `started_at`, `completed_at`, `status`, `error_message`

## Последствия

- **Плюсы**: атомарное обновление токенов без дублирования записей, возможность отладки синхронизаций, compliance-аудит.
- **Нейтрально**: требуется миграция БД при деплое.
- **Риски**: хранение refresh-токенов в открытом виде нарушает OWASP.
- **Митигация**: TOTP-секреты и OAuth refresh-токены — envelope encryption AES-256-GCM на уровне приложения (internal/crypto)

## Реализация

- Миграция: `db/migrations/V11__device_providers.sql`
- Репозитории/сервисы в `cmd/device-aggregator/providers/`.
- Шифрование `refresh_token` перед INSERT/UPDATE через `internal/crypto` (AES-256-GCM, `DEVICE_TOKEN_ENCRYPTION_KEY`), `Decrypt` — при необходимости использования токена для обновления `access_token`.
- Использование `ON CONFLICT (user_id, provider) DO UPDATE`.
