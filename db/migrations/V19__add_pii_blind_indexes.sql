-- V19__add_pii_blind_indexes.sql
-- Добавляем blind-index колонки и nonce для рандомизированного AEAD.
-- Инфраструктура для email (email_hash) уже существует; здесь дополняем full_name и nickname.

ALTER TABLE users ADD COLUMN IF NOT EXISTS full_name_hash VARCHAR(64);
ALTER TABLE users ADD COLUMN IF NOT EXISTS full_name_nonce  BYTEA;
ALTER TABLE users ADD COLUMN IF NOT EXISTS nickname_hash  VARCHAR(64);
ALTER TABLE users ADD COLUMN IF NOT EXISTS nickname_nonce   BYTEA;

CREATE INDEX IF NOT EXISTS idx_users_full_name_hash ON users(full_name_hash);
CREATE INDEX IF NOT EXISTS idx_users_nickname_hash  ON users(nickname_hash);
