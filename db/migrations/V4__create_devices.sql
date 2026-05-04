-- V4__create_devices.sql
-- Devices registered by device-connector

-- Devices
CREATE TABLE IF NOT EXISTS devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_type     VARCHAR(50) NOT NULL,
    device_name     VARCHAR(100),
    token           VARCHAR(255) UNIQUE NOT NULL,
    is_connected    BOOLEAN NOT NULL DEFAULT TRUE,
    last_sync       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_devices_user ON devices(user_id);