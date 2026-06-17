-- V13__add_totp_2fa.sql
-- Adds TOTP two-factor authentication support

ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret_encrypted BYTEA;

ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_backup_codes_hash TEXT[];

ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_backup_codes_remaining INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_users_totp_enabled
    ON users(totp_enabled)
    WHERE totp_enabled = TRUE;

COMMENT ON COLUMN users.totp_secret_encrypted IS 'TOTP secret encrypted with AES-256-GCM using TOTP_ENCRYPTION_KEY';
COMMENT ON COLUMN users.totp_backup_codes_hash IS 'Array of SHA-256 hashes of backup codes (10 codes)';
