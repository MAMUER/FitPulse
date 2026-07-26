-- V1__full_schema.sql
-- Consolidated idempotent schema for FitPulse.
-- This is the single authoritative migration for all deployments (new and existing).

-- ===================== Extensions =====================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pgsodium;

-- ===================== Users & Auth =====================
CREATE TABLE IF NOT EXISTS users (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email_encrypted         BYTEA,
    email_nonce             BYTEA,
    email_hash              VARCHAR(64) UNIQUE,
    password_hash           VARCHAR(255),
    full_name_encrypted     BYTEA,
    full_name_nonce         BYTEA,
    full_name_hash          VARCHAR(64),
    nickname_encrypted      BYTEA,
    nickname_nonce          BYTEA,
    nickname_hash           VARCHAR(64),
    profile_photo_url       VARCHAR(500),
    role                    VARCHAR(50) NOT NULL DEFAULT 'client'
                                CHECK (role IN ('client', 'admin')),
    email_confirmed         BOOLEAN NOT NULL DEFAULT FALSE,
    provider                VARCHAR(50) NOT NULL DEFAULT 'local',
    external_id             VARCHAR(255),
    totp_secret_encrypted   BYTEA,
    totp_enabled            BOOLEAN NOT NULL DEFAULT FALSE,
    totp_backup_codes_hash  TEXT[],
    totp_backup_codes_remaining INT NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ DEFAULT NOW(),
    updated_at              TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email_hash ON users(email_hash);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_totp_enabled ON users(totp_enabled) WHERE totp_enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_full_name_hash ON users(full_name_hash);
CREATE INDEX IF NOT EXISTS idx_users_nickname_hash ON users(nickname_hash);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_provider_external ON users(provider, external_id) WHERE external_id IS NOT NULL;

-- ===================== Email Verifications =====================
CREATE TABLE IF NOT EXISTS email_verifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_encrypted BYTEA,
    email_nonce     BYTEA,
    email_hash       VARCHAR(64),
    token           VARCHAR(255) UNIQUE NOT NULL,
    token_encrypted  BYTEA,
    token_nonce     BYTEA,
    expires_at      TIMESTAMPTZ NOT NULL,
    used            BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_verifications_token ON email_verifications(token);
CREATE INDEX IF NOT EXISTS idx_email_verifications_user ON email_verifications(user_id);

-- ===================== Refresh Tokens =====================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token       VARCHAR(255) UNIQUE NOT NULL,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens(expires_at);

-- Fix: rename legacy "used" column to "revoked" if it exists from old init-db.sql
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'refresh_tokens' AND column_name = 'used') THEN
        ALTER TABLE refresh_tokens RENAME COLUMN used TO revoked;
    END IF;
END $$;

-- ===================== Invite Codes =====================
CREATE TABLE IF NOT EXISTS invite_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(100) UNIQUE NOT NULL,
    role        VARCHAR(50) NOT NULL DEFAULT 'admin',
    specialty   VARCHAR(100),
    max_uses    INT NOT NULL DEFAULT 1,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at  TIMESTAMPTZ,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS invite_code_uses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invite_code_id  UUID NOT NULL REFERENCES invite_codes(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    used_at         TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invite_code_uses_code ON invite_code_uses(invite_code_id);
CREATE INDEX IF NOT EXISTS idx_invite_code_uses_user ON invite_code_uses(user_id);

-- ===================== Profiles =====================
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    age             INT CHECK (age IS NULL OR (age >= 0 AND age <= 150)),
    gender          VARCHAR(10) CHECK (gender IS NULL OR gender IN ('male', 'female', 'other')),
    height_cm       INT CHECK (height_cm IS NULL OR (height_cm >= 50 AND height_cm <= 300)),
    weight_kg       DECIMAL(5,2) CHECK (weight_kg IS NULL OR (weight_kg >= 1 AND weight_kg <= 500)),
    fitness_level   VARCHAR(50) CHECK (fitness_level IS NULL OR fitness_level IN ('beginner', 'intermediate', 'advanced')),
    nutrition       TEXT,
    sleep_hours     REAL CHECK (sleep_hours IS NULL OR (sleep_hours >= 0 AND sleep_hours <= 24)),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_goals (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    goal        VARCHAR(100) NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, goal)
);

CREATE TABLE IF NOT EXISTS user_contraindications (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contraindication VARCHAR(255) NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, contraindication)
);

-- ===================== Devices =====================
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

-- ===================== Biometrics =====================
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

CREATE TABLE IF NOT EXISTS device_ingest_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL,
    quality     VARCHAR(20) DEFAULT 'good',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ingest_log_device_time ON device_ingest_log(device_id, timestamp);

-- ===================== Training =====================
CREATE TABLE IF NOT EXISTS training_plans (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                VARCHAR(255),
    classification_class VARCHAR(255),
    training_goal       VARCHAR(50) CHECK (training_goal IS NULL OR training_goal IN (
        'weight_loss', 'muscle_gain', 'endurance', 'strength', 'flexibility', 'general_fitness',
        'recovery', 'endurance_e1e2', 'threshold_e3', 'strength_hiit'
    )),
    training_location   VARCHAR(50) CHECK (training_location IS NULL OR training_location IN ('home', 'gym', 'pool', 'outdoor')),
    available_time      VARCHAR(20) CHECK (available_time IS NULL OR available_time IN ('morning', 'afternoon', 'evening')),
    duration_weeks      INT CHECK (duration_weeks IS NULL OR (duration_weeks > 0 AND duration_weeks <= 52)),
    generated_at        TIMESTAMPTZ DEFAULT NOW(),
    start_date          DATE,
    end_date            DATE,
    status              VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'completed', 'cancelled', 'paused')),
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_training_plans_user ON training_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_training_plans_status ON training_plans(user_id, status);
CREATE INDEX IF NOT EXISTS idx_training_plans_classification_class ON training_plans(classification_class);

CREATE TABLE IF NOT EXISTS training_plan_weeks (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    training_plan_id        UUID NOT NULL REFERENCES training_plans(id) ON DELETE CASCADE,
    week_number             INT NOT NULL CHECK (week_number > 0),
    total_training_days     INT DEFAULT 0,
    total_duration_minutes  INT DEFAULT 0,
    average_intensity       DECIMAL(3,2),
    UNIQUE (training_plan_id, week_number)
);

CREATE TABLE IF NOT EXISTS training_plan_days (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    week_id                 UUID NOT NULL REFERENCES training_plan_weeks(id) ON DELETE CASCADE,
    day_of_week             INT NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
    training_date           DATE,
    training_type           VARCHAR(50),
    is_rest_day             BOOLEAN NOT NULL DEFAULT FALSE,
    total_duration_minutes  INT,
    intensity_level         DECIMAL(3,2),
    notes                   TEXT
);

CREATE TABLE IF NOT EXISTS training_exercises (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    day_id          UUID NOT NULL REFERENCES training_plan_days(id) ON DELETE CASCADE,
    exercise_name   VARCHAR(255) NOT NULL,
    duration_minutes INT,
    intensity       DECIMAL(3,2),
    sets            INT,
    reps            INT,
    rest_seconds    INT DEFAULT 60,
    description     TEXT,
    video_url       VARCHAR(500),
    sort_order      INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_exercises_day ON training_exercises(day_id);

CREATE TABLE IF NOT EXISTS workout_completions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    training_plan_id    UUID REFERENCES training_plans(id) ON DELETE CASCADE,
    workout_id          VARCHAR(100) NOT NULL,
    scheduled_date      DATE DEFAULT CURRENT_DATE,
    completed           BOOLEAN DEFAULT FALSE,
    completed_at        TIMESTAMPTZ,
    feedback            TEXT,
    rating              INT CHECK (rating IS NULL OR (rating >= 1 AND rating <= 5)),
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workout_completions_user ON workout_completions(user_id);
CREATE INDEX IF NOT EXISTS idx_workout_completions_plan ON workout_completions(training_plan_id);

-- ===================== Achievements =====================
CREATE TABLE IF NOT EXISTS achievements (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    criteria    JSONB,
    icon_url    VARCHAR(255),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_achievements (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    achievement_id  UUID NOT NULL REFERENCES achievements(id) ON DELETE CASCADE,
    earned_at       TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, achievement_id)
);

INSERT INTO achievements (name, description, criteria, icon_url) VALUES
    ('Первый шаг', 'Первая завершенная тренировка', '{"type": "workout_count", "threshold": 1}', '/icons/first_workout.png'),
    ('Десятка', '10 завершенных тренировок', '{"type": "workout_count", "threshold": 10}', '/icons/ten_workouts.png'),
    ('Полтинник', '50 завершенных тренировок', '{"type": "workout_count", "threshold": 50}', '/icons/fifty_workouts.png'),
    ('Сто дней', '100 дней активности', '{"type": "active_days", "threshold": 100}', '/icons/hundred_days.png'),
    ('Мастер спорта', '1000 завершенных тренировок', '{"type": "workout_count", "threshold": 1000}', '/icons/master_sport.png')
ON CONFLICT DO NOTHING;

-- ===================== Health Features =====================
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

-- ===================== Device Providers / OAuth =====================
CREATE TABLE IF NOT EXISTS oauth_states (
    state       VARCHAR(255) PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider    VARCHAR(50) NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_states_expires ON oauth_states(expires_at);

CREATE TABLE IF NOT EXISTS device_provider_accounts (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider                VARCHAR(50) NOT NULL,
    provider_user_id        VARCHAR(255) NOT NULL,
    access_token            TEXT NOT NULL,
    refresh_token           TEXT,
    token_expires_at        TIMESTAMPTZ,
    scopes                  TEXT[],
    webhook_subscription_id VARCHAR(255),
    last_sync_at            TIMESTAMPTZ,
    is_active               BOOLEAN DEFAULT TRUE,
    created_at              TIMESTAMPTZ DEFAULT NOW(),
    updated_at              TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_provider_accounts_user ON device_provider_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_provider_accounts_provider ON device_provider_accounts(provider);

CREATE TABLE IF NOT EXISTS device_sync_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_account_id UUID REFERENCES device_provider_accounts(id) ON DELETE CASCADE,
    sync_type           VARCHAR(50) NOT NULL,
    records_count       INT DEFAULT 0,
    started_at          TIMESTAMPTZ DEFAULT NOW(),
    completed_at        TIMESTAMPTZ,
    status              VARCHAR(20) DEFAULT 'pending',
    error_message       TEXT,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sync_log_provider_account ON device_sync_log(provider_account_id);

-- ===================== Biometric Dedup =====================
ALTER TABLE biometric_data
    ADD CONSTRAINT IF NOT EXISTS uq_biometric_user_metric_time_device
    UNIQUE (user_id, metric_type, timestamp, device_type);

-- ===================== Views =====================
CREATE OR REPLACE VIEW invite_code_stats AS
SELECT
    ic.id,
    ic.code,
    ic.role,
    ic.specialty,
    ic.max_uses,
    COUNT(icu.id) AS used_count,
    ic.is_active,
    ic.expires_at,
    ic.created_at
FROM invite_codes ic
LEFT JOIN invite_code_uses icu ON icu.invite_code_id = ic.id
GROUP BY ic.id, ic.code, ic.role, ic.specialty, ic.max_uses, ic.is_active, ic.expires_at, ic.created_at;

CREATE OR REPLACE VIEW user_profiles_with_goals AS
SELECT
    up.user_id,
    up.age,
    up.gender,
    up.height_cm,
    up.weight_kg,
    up.fitness_level,
    ARRAY_AGG(ug.goal) FILTER (WHERE ug.goal IS NOT NULL) AS goals,
    up.nutrition,
    up.sleep_hours,
    up.created_at,
    up.updated_at
FROM user_profiles up
LEFT JOIN user_goals ug ON ug.user_id = up.user_id
GROUP BY up.user_id, up.age, up.gender, up.height_cm, up.weight_kg, up.fitness_level,
         up.nutrition, up.sleep_hours, up.created_at, up.updated_at;

-- ===================== Functions =====================
CREATE OR REPLACE FUNCTION create_invite_code(
    p_role VARCHAR(50),
    p_specialty VARCHAR(100),
    p_max_uses INT,
    p_valid_days INT
) RETURNS VARCHAR AS $$
DECLARE
    v_code VARCHAR(100);
BEGIN
    v_code := UPPER(p_role) || '-' || TO_CHAR(NOW(), 'YYYY') || '-' ||
              UPPER(REPLACE(
                  REPLACE(
                      COALESCE(p_specialty, 'GENERAL'),
                      ' ', '-'
                  ),
                  '_', '-'
              )) || '-' ||
              LPAD((SELECT COALESCE(MAX(
                  NULLIF(SUBSTRING(code FROM '-[^-]*$'), '')::int, 0)
              ) + 1 FROM invite_codes WHERE code LIKE UPPER(p_role) || '-%'), 3, '0');

    INSERT INTO invite_codes (code, role, specialty, max_uses, is_active, expires_at)
    VALUES (
        v_code,
        p_role,
        p_specialty,
        p_max_uses,
        TRUE,
        CASE WHEN p_valid_days > 0 THEN NOW() + (p_valid_days || ' days')::INTERVAL ELSE NULL END
    );

    RETURN v_code;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION use_invite_code(
    p_code VARCHAR(100)
) RETURNS TABLE(
    is_valid BOOLEAN,
    role VARCHAR(50),
    specialty VARCHAR(100),
    error_msg TEXT
) AS $$
DECLARE
    v_record RECORD;
    v_used_count INT;
BEGIN
    SELECT * INTO v_record FROM invite_codes WHERE code = p_code AND is_active = TRUE;

    IF NOT FOUND THEN
        is_valid := FALSE; role := NULL; specialty := NULL; error_msg := 'Invite code not found or inactive';
        RETURN NEXT; RETURN;
    END IF;

    IF v_record.expires_at IS NOT NULL AND v_record.expires_at < NOW() THEN
        is_valid := FALSE; role := NULL; specialty := NULL; error_msg := 'Invite code has expired';
        RETURN NEXT; RETURN;
    END IF;

    SELECT COUNT(*) INTO v_used_count FROM invite_code_uses WHERE invite_code_id = v_record.id;
    IF v_used_count >= v_record.max_uses THEN
        is_valid := FALSE; role := NULL; specialty := NULL; error_msg := 'Invite code has reached its usage limit';
        RETURN NEXT; RETURN;
    END IF;

    is_valid := TRUE;
    role := v_record.role;
    specialty := v_record.specialty;
    error_msg := NULL;
    RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION log_invite_code_use(
    p_code VARCHAR(100),
    p_user_id UUID
) RETURNS VOID AS $$
DECLARE
    v_invite_id UUID;
BEGIN
    SELECT id INTO v_invite_id FROM invite_codes WHERE code = p_code;
    IF v_invite_id IS NULL THEN
        RAISE EXCEPTION 'Invite code not found: %', p_code;
    END IF;

    INSERT INTO invite_code_uses (invite_code_id, user_id) VALUES (v_invite_id, p_user_id);
END;
$$ LANGUAGE plpgsql;
