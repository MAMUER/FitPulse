-- V17__add_user_menstrual_cycles.sql
-- 3NF: каждый цикл — отдельная запись с нормализованными справочниками симптомов/настроения.
CREATE TABLE IF NOT EXISTS user_menstrual_cycles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cycle_start_date DATE NOT NULL,
    cycle_end_date   DATE,
    flow_intensity   VARCHAR(50) CHECK (flow_intensity IS NULL OR flow_intensity IN ('none','light','medium','heavy')),
    notes            TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT chk_cycle_dates CHECK (cycle_end_date IS NULL OR cycle_end_date >= cycle_start_date)
);

CREATE TABLE IF NOT EXISTS user_menstrual_symptoms (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cycle_id    UUID NOT NULL REFERENCES user_menstrual_cycles(id) ON DELETE CASCADE,
    symptom     VARCHAR(100) NOT NULL,
    severity    VARCHAR(50) CHECK (severity IS NULL OR severity IN ('mild','moderate','severe')),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_menstrual_moods (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cycle_id    UUID NOT NULL REFERENCES user_menstrual_cycles(id) ON DELETE CASCADE,
    mood        VARCHAR(100) NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_menstrual_cycles_user ON user_menstrual_cycles(user_id, cycle_start_date DESC);
CREATE INDEX IF NOT EXISTS idx_user_menstrual_symptoms_cycle ON user_menstrual_symptoms(cycle_id);
CREATE INDEX IF NOT EXISTS idx_user_menstrual_moods_cycle ON user_menstrual_moods(cycle_id);
