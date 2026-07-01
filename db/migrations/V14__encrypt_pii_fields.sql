-- V14__encrypt_pii_fields.sql
-- Add pgcrypto-encrypted columns for PII fields

-- Users: encrypted PII columns
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_encrypted BYTEA;
ALTER TABLE users ADD COLUMN IF NOT EXISTS full_name_encrypted BYTEA;
ALTER TABLE users ADD COLUMN IF NOT EXISTS nickname_encrypted BYTEA;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_hash VARCHAR(64) UNIQUE;

-- Email verifications: encrypted columns
ALTER TABLE email_verifications ADD COLUMN IF NOT EXISTS email_encrypted BYTEA;
ALTER TABLE email_verifications ADD COLUMN IF NOT EXISTS token_encrypted BYTEA;

-- Backfill existing data using pgcrypto with a placeholder key.
-- WARNING: The key below is a TEMPORARY backfill key. Production MUST set
-- DB_ENCRYPTION_KEY; the application backfillEncryptedPII() will re-encrypt
-- any remaining plaintext rows with the real key on startup.
UPDATE users
SET
    email_encrypted = pgp_sym_encrypt(email, '00000000000000000000000000000000'),
    email_hash = encode(digest(lower(email), 'sha256'), 'hex'),
    full_name_encrypted = pgp_sym_encrypt(full_name, '00000000000000000000000000000000'),
    nickname_encrypted = pgp_sym_encrypt(nickname, '00000000000000000000000000000000')
WHERE email_encrypted IS NULL;

UPDATE email_verifications
SET
    email_encrypted = pgp_sym_encrypt(email, '00000000000000000000000000000000'),
    token_encrypted = pgp_sym_encrypt(token, '00000000000000000000000000000000')
WHERE email_encrypted IS NULL;

-- Indexes for encrypted lookups
CREATE INDEX IF NOT EXISTS idx_users_email_hash ON users(email_hash);
CREATE INDEX IF NOT EXISTS idx_email_verifications_user ON email_verifications(user_id);
