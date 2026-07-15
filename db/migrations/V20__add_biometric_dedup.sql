-- V20__add_biometric_dedup.sql
-- Add unique constraint for biometric data deduplication

ALTER TABLE biometric_data
ADD CONSTRAINT uq_biometric_user_metric_time_device
UNIQUE (user_id, metric_type, timestamp, device_type);
