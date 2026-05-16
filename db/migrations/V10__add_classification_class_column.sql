-- V10__add_classification_class_column.sql
-- Add classification_class column to training_plans (was missing from V6)

ALTER TABLE training_plans
    ADD COLUMN IF NOT EXISTS classification_class VARCHAR(255);

COMMENT ON COLUMN training_plans.classification_class IS 'ML-классификация типа тренировки (например endurance_e1e2)';
