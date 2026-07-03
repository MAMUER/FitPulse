-- V15__add_user_health_features.sql
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
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_health_conditions_user ON user_health_conditions(user_id);

-- V16__add_user_body_composition.sql
-- 3NF: исторические измерения веса/состава тела. Каждая строка — отдельное измерение.
CREATE TABLE IF NOT EXISTS user_body_composition (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recorded_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    weight_kg               NUMERIC(5,2) CHECK (weight_kg IS NULL OR (weight_kg >= 1 AND weight_kg <= 500)),
    height_cm               INT CHECK (height_cm IS NULL OR (height_cm >= 50 AND height_cm <= 300)),
    bmi                     NUMERIC(4,2),
    body_fat_percentage     NUMERIC(4,2) CHECK (body_fat_percentage IS NULL OR (body_fat_percentage >= 1 AND body_fat_percentage <= 100)),
    muscle_mass_percentage  NUMERIC(4,2) CHECK (muscle_mass_percentage IS NULL OR (muscle_mass_percentage >= 1 AND muscle_mass_percentage <= 100)),
    bone_mass_percentage    NUMERIC(4,2) CHECK (bone_mass_percentage IS NULL OR (bone_mass_percentage >= 1 AND bone_mass_percentage <= 100)),
    water_percentage        NUMERIC(4,2) CHECK (water_percentage IS NULL OR (water_percentage >= 1 AND water_percentage <= 100)),
    visceral_fat_rating     INT CHECK (visceral_fat_rating IS NULL OR (visceral_fat_rating >= 1 AND visceral_fat_rating <= 59)),
    metabolic_age           INT CHECK (metabolic_age IS NULL OR (metabolic_age >= 10 AND metabolic_age <= 100)),
    source                  VARCHAR(50) NOT NULL DEFAULT 'manual' CHECK (source IN ('okok','manual')),
    created_at              TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_body_composition_user ON user_body_composition(user_id, recorded_at DESC);

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
