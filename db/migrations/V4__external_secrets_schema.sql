-- External Secrets Operator — PostgreSQL backend schema
-- Free, open-source secrets backend using existing PostgreSQL
-- Run this migration after the main schema is applied

CREATE TABLE IF NOT EXISTS external_secrets (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    value       TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_external_secrets_name
    ON external_secrets (name);

CREATE OR REPLACE FUNCTION update_external_secrets_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_external_secrets_updated_at ON external_secrets;

CREATE TRIGGER trg_external_secrets_updated_at
    BEFORE UPDATE ON external_secrets
    FOR EACH ROW
    EXECUTE FUNCTION update_external_secrets_updated_at();
