-- V15__add_user_health_conditions.sql
-- 3NF: отдельная таблица состояний здоровья, независимая от user_goals/user_contraindications.
-- Каждая запись — один факт: аллергия/заболевание/инвалидность.
CREATE TABLE IF NOT EXISTS user_health_conditions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    condition_type  VARCHAR(50) NOT NULL CHECK (condition_type IN ('allergy','disease','disability','other')),
    condition_name  VARCHAR(255) NOT NULL,
    severity        VARCHAR(50) CHECK (severity IS NULL OR severity IN ('mild','moderate','severe')),
    diagnosed_at    DATE,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    notes           TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, condition_type, condition_name)
);

CREATE INDEX IF NOT EXISTS idx_user_health_conditions_user ON user_health_conditions(user_id);
