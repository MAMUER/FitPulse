-- V2__add_source_to_biometric_metrics.sql
-- Add source column support to biometric_data for Open Wearables integration

DO $body$
DECLARE
    v_default_source CONSTANT VARCHAR := 'unknown';
BEGIN
    -- Add source column if it does not exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'biometric_data' AND column_name = 'source'
    ) THEN
        ALTER TABLE biometric_data
            ADD COLUMN source VARCHAR(100) NOT NULL DEFAULT v_default_source;
    END IF;

    -- Backfill source from device_type for existing rows
    UPDATE biometric_data
    SET source = COALESCE(NULLIF(device_type, ''), v_default_source)
    WHERE source = v_default_source
      AND device_type IS NOT NULL
      AND device_type IS DISTINCT FROM '';

    -- Drop old unique constraint/index if exists
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'biometric_data_user_id_metric_type_timestamp_device_type_key'
    ) THEN
        ALTER TABLE biometric_data
            DROP CONSTRAINT biometric_data_user_id_metric_type_timestamp_device_type_key;
    END IF;
END $body$;

DROP INDEX IF EXISTS idx_biometric_user_metric_time;

-- Recreate unique index including source
CREATE UNIQUE INDEX IF NOT EXISTS idx_biometric_user_metric_time
    ON biometric_data(user_id, metric_type, timestamp, source);

-- Recreate timestamp index
CREATE INDEX IF NOT EXISTS idx_biometric_timestamp
    ON biometric_data(timestamp);

-- Add check constraint for source values
DO $body$
DECLARE
    v_default_source CONSTANT VARCHAR := 'unknown';
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'biometric_data_source_check'
    ) THEN
        ALTER TABLE biometric_data
            ADD CONSTRAINT biometric_data_source_check
            CHECK (source IN (
                'apple_health', 'garmin', 'health_connect', 'open_wearables',
                'fitbit', 'withings', 'okok', 'flo', v_default_source
            ));
    END IF;
END $body$;
