-- V18__enable_pgsodium.sql
-- Заменяем pgcrypto (pgp_sym_encrypt) на pgsodium (libsodium) для PII-полей.
-- Ключ импортируется в keyring (pgsodium.key) при старте user-service из DB_ENCRYPTION_KEY.

CREATE EXTENSION IF NOT EXISTS pgsodium;
