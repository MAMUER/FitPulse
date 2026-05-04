-- V4__create_biometric_data.sql
-- Biometric data and device ingestion

-- Biometric data
CREATE TABLE IF NOT EXISTS biometric_data (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL,
    value       DOUBLE PRECISION NOT NULL CHECK (value >= 0),
    timestamp   TIMESTAMPTZ NOT NULL,
    device_type VARCHAR(50),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_biometric_user_metric_time ON biometric_data(user_id, metric_type, timestamp);
CREATE INDEX IF NOT EXISTS idx_biometric_timestamp ON biometric_data(timestamp);

-- Device ingestion log (deduplication)
CREATE TABLE IF NOT EXISTS device_ingest_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL,
    quality     VARCHAR(20) DEFAULT 'good',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ingest_log_device_time ON device_ingest_log(device_id, timestamp);