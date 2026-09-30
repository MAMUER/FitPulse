# ADR 0023: Выбор pgsodium (libsodium) для шифрования PII

## Статус

Принято

## Контекст

FitPulse обрабатывает специальные категории персональных данных (биометрические и медицинские). Требуется надёжное шифрование at rest для:

- Email, имён, никнеймов
- Фотографий профиля
- Медицинских данных (заболевания, менструальные циклы, состав тела)
- TOTP-секретов
- Refresh-токенов

PostgreSQL предлагает несколько вариантов шифрования: `pgcrypto` (встроенный), `pgp_sym_encrypt`/`pgp_pub_decrypt`, внешние extension (pgsodium, pg_tde).

## Решение

Использовать **pgsodium** (libsodium wrapper) для шифрования PII:

1. **Deteministic AEAD** для полей, по которым требуется поиск (`email_hash`, `full_name_hash`, `nickname_hash`).
2. **Envelope encryption** для секретов (`totp_secret_encrypted`, `refresh_token_encrypted`, `profile_photo_url_encrypted`).
3. **Nonce management**: pgsodium автоматически генерирует nonce для каждогоencrypt.
4. **Ключевое управление**: мастер-ключ хранится в `DB_ENCRYPTION_KEY` env var (Kubernetes Secret).

## Последствия

- **Плюсы**: детерминированный поиск по зашифрованным полям, защита от утечек через side-channel, совместимость с PostgreSQL 18.
- **Нейтрально**: требуется установка extension `pgsodium` в кластере PostgreSQL.
- **Риски**: потеря мастер-ключа = потеря доступа ко всем зашифрованным данным. Митигация: бэкапы ключа в Vault (Phase 2).

## Рассмотренные альтернативы

- **pgcrypto**: не поддерживает детерминированное шифрование, сложнее управление ключами.
- **pg_tde**: требуется PostgreSQL 16+, transparent encryption на уровне tablespace, не даёт детерминированный поиск.
- **Application-level encryption**: повышает сложность кода, риск ошибок в реализации.

## Реализация

- `internal/crypto/` — утилиты для envelope encryption.
- `db/migrations/V1__full_schema.sql` — столбцы `*_encrypted`, `*_hash`, `*_nonce` для email, full_name, nickname, profile_photo_url, totp_secret, refresh_tokens.token.
- `internal/db/pgsodium.go` — `PgsodiumRandomEncryptParam`, `PgsodiumDecryptParam`, `BlindIndex`, `GenerateNonce`.
- `internal/repository/postgres/pgsodium_user_repository.go` — репозиторий пользователей с шифрованием/дешифрованием PII.
- `internal/repository/postgres/refresh_token_repository.go` — шифрование refresh-токенов через pgsodium.
- `cmd/user-service/main.go` — инициализация pgsodium keyring, миграция pgcrypto → pgsodium при старте.
