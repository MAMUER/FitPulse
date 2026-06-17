-- V12__oauth_providers.sql
-- OAuth providers support (Google OAuth)

-- Make password optional for OAuth users (local users still set password)
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- OAuth provider fields
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS provider VARCHAR(50) NOT NULL DEFAULT 'local',
    ADD COLUMN IF NOT EXISTS external_id VARCHAR(255);

-- Unique index for OAuth lookups (google + google_sub)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_provider_external
    ON users(provider, external_id)
    WHERE external_id IS NOT NULL;