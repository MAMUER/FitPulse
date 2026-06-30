-- V1__create_extensions.sql
-- Create necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- V2__create_users_and_auth.sql
-- Core users and authentication tables
-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    nickname VARCHAR(100) UNIQUE,
    profile_photo_url VARCHAR(500),
    role VARCHAR(50) NOT NULL DEFAULT 'client' CHECK (role IN ('client', 'admin')),
    email_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Email verification tokens
CREATE TABLE IF NOT EXISTS email_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_verifications_token ON email_verifications(token);

CREATE INDEX IF NOT EXISTS idx_email_verifications_user ON email_verifications(user_id);

-- Invite codes (for admin registration)
CREATE TABLE IF NOT EXISTS invite_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(100) UNIQUE NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'admin',
    specialty VARCHAR(100),
    max_uses INT NOT NULL DEFAULT 1,
    created_by UUID REFERENCES users(id) ON DELETE
    SET
        NULL,
        expires_at TIMESTAMPTZ,
        is_active BOOLEAN NOT NULL DEFAULT TRUE,
        created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Invite code usage log (3NF — replaces used_count counter)
CREATE TABLE IF NOT EXISTS invite_code_uses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invite_code_id UUID NOT NULL REFERENCES invite_codes(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE
    SET
        NULL,
        used_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invite_code_uses_code ON invite_code_uses(invite_code_id);

CREATE INDEX IF NOT EXISTS idx_invite_code_uses_user ON invite_code_uses(user_id);

-- V3__create_user_profiles.sql
-- User profiles and related data
-- User profiles
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    age INT CHECK (
        age IS NULL
        OR (
            age >= 0
            AND age <= 150
        )
    ),
    gender VARCHAR(10) CHECK (
        gender IS NULL
        OR gender IN ('male', 'female', 'other')
    ),
    height_cm INT CHECK (
        height_cm IS NULL
        OR (
            height_cm >= 50
            AND height_cm <= 300
        )
    ),
    weight_kg DECIMAL(5, 2) CHECK (
        weight_kg IS NULL
        OR (
            weight_kg >= 1
            AND weight_kg <= 500
        )
    ),
    fitness_level VARCHAR(50) CHECK (
        fitness_level IS NULL
        OR fitness_level IN ('beginner', 'intermediate', 'advanced')
    ),
    nutrition TEXT,
    sleep_hours REAL CHECK (
        sleep_hours IS NULL
        OR (
            sleep_hours >= 0
            AND sleep_hours <= 24
        )
    ),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User fitness goals (1NF — normalized from goals TEXT[])
CREATE TABLE IF NOT EXISTS user_goals (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    goal VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, goal)
);

-- User contraindications (1NF — normalized from contraindications TEXT[])
CREATE TABLE IF NOT EXISTS user_contraindications (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contraindication VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, contraindication)
);

-- V4__create_devices.sql
-- Devices registered by device-connector
-- Devices
CREATE TABLE IF NOT EXISTS devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_type VARCHAR(50) NOT NULL,
    device_name VARCHAR(100),
    token VARCHAR(255) UNIQUE NOT NULL,
    is_connected BOOLEAN NOT NULL DEFAULT TRUE,
    last_sync TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_devices_user ON devices(user_id);

-- V5__create_biometric_data.sql
-- Biometric data and device ingestion
-- Biometric data
CREATE TABLE IF NOT EXISTS biometric_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL,
    value DOUBLE PRECISION NOT NULL CHECK (value >= 0),
    timestamp TIMESTAMPTZ NOT NULL,
    device_type VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_biometric_user_metric_time ON biometric_data(user_id, metric_type, timestamp);

CREATE INDEX IF NOT EXISTS idx_biometric_timestamp ON biometric_data(timestamp);

-- Device ingestion log (deduplication)
CREATE TABLE IF NOT EXISTS device_ingest_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    quality VARCHAR(20) DEFAULT 'good',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ingest_log_device_time ON device_ingest_log(device_id, timestamp);

-- V6__create_training_plans.sql
-- Training plans and related data
-- Training plans
CREATE TABLE IF NOT EXISTS training_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255),
    training_goal VARCHAR(50) CHECK (
        training_goal IS NULL
        OR training_goal IN (
            'weight_loss',
            'muscle_gain',
            'endurance',
            'strength',
            'flexibility',
            'general_fitness',
            'recovery',
            'endurance_e1e2',
            'threshold_e3',
            'strength_hiit'
        )
    ),
    training_location VARCHAR(50) CHECK (
        training_location IS NULL
        OR training_location IN ('home', 'gym', 'pool', 'outdoor')
    ),
    available_time VARCHAR(20) CHECK (
        available_time IS NULL
        OR available_time IN ('morning', 'afternoon', 'evening')
    ),
    duration_weeks INT CHECK (
        duration_weeks IS NULL
        OR (
            duration_weeks > 0
            AND duration_weeks <= 52
        )
    ),
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    start_date DATE,
    end_date DATE,
    status VARCHAR(50) DEFAULT 'active' CHECK (
        status IN ('active', 'completed', 'cancelled', 'paused')
    ),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_training_plans_user ON training_plans(user_id);

CREATE INDEX IF NOT EXISTS idx_training_plans_status ON training_plans(user_id, status);

-- Training plan weeks (1NF — normalized from JSONB)
CREATE TABLE IF NOT EXISTS training_plan_weeks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    training_plan_id UUID NOT NULL REFERENCES training_plans(id) ON DELETE CASCADE,
    week_number INT NOT NULL CHECK (week_number > 0),
    total_training_days INT DEFAULT 0,
    total_duration_minutes INT DEFAULT 0,
    average_intensity DECIMAL(3, 2),
    UNIQUE (training_plan_id, week_number)
);

-- Training plan days (1NF — normalized from JSONB)
CREATE TABLE IF NOT EXISTS training_plan_days (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    week_id UUID NOT NULL REFERENCES training_plan_weeks(id) ON DELETE CASCADE,
    day_of_week INT NOT NULL CHECK (
        day_of_week >= 0
        AND day_of_week <= 6
    ),
    training_date DATE,
    training_type VARCHAR(50),
    is_rest_day BOOLEAN NOT NULL DEFAULT FALSE,
    total_duration_minutes INT,
    intensity_level DECIMAL(3, 2),
    notes TEXT
);

-- Individual exercises (1NF — normalized from JSONB)
CREATE TABLE IF NOT EXISTS training_exercises (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    day_id UUID NOT NULL REFERENCES training_plan_days(id) ON DELETE CASCADE,
    exercise_name VARCHAR(255) NOT NULL,
    duration_minutes INT,
    intensity DECIMAL(3, 2),
    sets INT,
    reps INT,
    rest_seconds INT DEFAULT 60,
    description TEXT,
    video_url VARCHAR(500),
    sort_order INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_exercises_day ON training_exercises(day_id);

-- Workout completions
CREATE TABLE IF NOT EXISTS workout_completions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    training_plan_id UUID REFERENCES training_plans(id) ON DELETE CASCADE,
    workout_id VARCHAR(100) NOT NULL,
    scheduled_date DATE DEFAULT CURRENT_DATE,
    completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    feedback TEXT,
    rating INT CHECK (
        rating IS NULL
        OR (
            rating >= 1
            AND rating <= 5
        )
    ),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workout_completions_user ON workout_completions(user_id);

CREATE INDEX IF NOT EXISTS idx_workout_completions_plan ON workout_completions(training_plan_id);

-- V7__create_achievements.sql
-- Achievements system
-- Achievements
CREATE TABLE IF NOT EXISTS achievements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    criteria JSONB,
    icon_url VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User achievements
CREATE TABLE IF NOT EXISTS user_achievements (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    achievement_id UUID NOT NULL REFERENCES achievements(id) ON DELETE CASCADE,
    earned_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, achievement_id)
);

-- Seed achievements
INSERT INTO
    achievements (name, description, criteria)
VALUES
    (
        'Первый шаг',
        'Первая завершенная тренировка',
        '{"type": "workout_count", "threshold": 1}'
    ),
    (
        'Десятка',
        '10 завершенных тренировок',
        '{"type": "workout_count", "threshold": 10}'
    ),
    (
        'Полтинник',
        '50 завершенных тренировок',
        '{"type": "workout_count", "threshold": 50}'
    ),
    (
        'Сто дней',
        '100 дней активности',
        '{"type": "active_days", "threshold": 100}'
    ),
    (
        'Мастер спорта',
        '1000 завершенных тренировок',
        '{"type": "workout_count", "threshold": 1000}'
    ) ON CONFLICT DO NOTHING;

-- V8__create_views.sql
-- Views for backward compatibility and derived data
-- Invite code statistics (replaces invite_codes.used_count)
CREATE
OR REPLACE VIEW invite_code_stats AS
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
FROM
    invite_codes ic
    LEFT JOIN invite_code_uses icu ON icu.invite_code_id = ic.id
GROUP BY
    ic.id,
    ic.code,
    ic.role,
    ic.specialty,
    ic.max_uses,
    ic.expires_at,
    ic.created_at;

-- User profiles with goals array (backward compatibility for goals TEXT[])
CREATE
OR REPLACE VIEW user_profiles_with_goals AS
SELECT
    up.user_id,
    up.age,
    up.gender,
    up.height_cm,
    up.weight_kg,
    up.fitness_level,
    ARRAY_AGG(ug.goal) FILTER (
        WHERE
            ug.goal IS NOT NULL
    ) AS goals,
    up.nutrition,
    up.sleep_hours,
    up.created_at,
    up.updated_at
FROM
    user_profiles up
    LEFT JOIN user_goals ug ON ug.user_id = up.user_id
GROUP BY
    up.user_id,
    up.age,
    up.gender,
    up.height_cm,
    up.weight_kg,
    up.fitness_level,
    up.nutrition,
    up.sleep_hours,
    up.created_at,
    up.updated_at;

-- V9__create_functions.sql
-- Functions for invite code management
-- Create a new invite code
CREATE
OR REPLACE FUNCTION create_invite_code(
    p_role VARCHAR(50),
    p_specialty VARCHAR(100),
    p_max_uses INT,
    p_valid_days INT
) RETURNS VARCHAR AS $$ DECLARE v_code VARCHAR(100);

BEGIN v_code := UPPER(p_role) || '-' || TO_CHAR(NOW(), 'YYYY') || '-' || UPPER(
    REPLACE(
        REPLACE(
            COALESCE(p_specialty, 'GENERAL'),
            ' ',
            '-'
        ),
        '_',
        '-'
    )
) || '-' || LPAD(
    (
        SELECT
            COALESCE(
                MAX(
                    NULLIF(
                        SUBSTRING(
                            code
                            FROM
                                '-[^-]*$'
                        ),
                        ''
                    ) :: int,
                    0
                )
            ) + 1
        FROM
            invite_codes
        WHERE
            code LIKE UPPER(p_role) || '-%'
    ),
    3,
    '0'
);

INSERT INTO
    invite_codes (
        code,
        role,
        specialty,
        max_uses,
        is_active,
        expires_at
    )
VALUES
    (
        v_code,
        p_role,
        p_specialty,
        p_max_uses,
        TRUE,
        CASE
            WHEN p_valid_days > 0 THEN NOW() + (p_valid_days || ' days') :: INTERVAL
            ELSE NULL
        END
    );

RETURN v_code;

END;

$$ LANGUAGE plpgsql;

-- Validate and consume an invite code
CREATE
OR REPLACE FUNCTION use_invite_code(p_code VARCHAR(100)) RETURNS TABLE(
    is_valid BOOLEAN,
    role VARCHAR(50),
    specialty VARCHAR(100),
    error_msg TEXT
) AS $$ DECLARE v_record RECORD;

v_used_count INT;

BEGIN
SELECT
    * INTO v_record
FROM
    invite_codes
WHERE
    code = p_code
    AND is_active = TRUE;

IF NOT FOUND THEN is_valid := FALSE;

role := NULL;

specialty := NULL;

error_msg := 'Invite code not found or inactive';

RETURN NEXT;

RETURN;

END IF;

IF v_record.expires_at IS NOT NULL
AND v_record.expires_at < NOW() THEN is_valid := FALSE;

role := NULL;

specialty := NULL;

error_msg := 'Invite code has expired';

RETURN NEXT;

RETURN;

END IF;

SELECT
    COUNT(*) INTO v_used_count
FROM
    invite_code_uses
WHERE
    invite_code_id = v_record.id;

IF v_used_count >= v_record.max_uses THEN is_valid := FALSE;

role := NULL;

specialty := NULL;

error_msg := 'Invite code has reached its usage limit';

RETURN NEXT;

RETURN;

END IF;

is_valid := TRUE;

role := v_record.role;

specialty := v_record.specialty;

error_msg := NULL;

RETURN NEXT;

END;

$$ LANGUAGE plpgsql;

-- Log invite code usage
CREATE
OR REPLACE FUNCTION log_invite_code_use(p_code VARCHAR(100), p_user_id UUID) RETURNS VOID AS $$ DECLARE v_invite_id UUID;

BEGIN
SELECT
    id INTO v_invite_id
FROM
    invite_codes
WHERE
    code = p_code;

IF v_invite_id IS NULL THEN RAISE EXCEPTION 'Invite code not found: %',
p_code;

END IF;

INSERT INTO
    invite_code_uses (invite_code_id, user_id)
VALUES
    (v_invite_id, p_user_id);

END;

$$ LANGUAGE plpgsql;

-- V10__add_classification_class_column.sql
-- Add classification_class column to training_plans (was missing from V6)
ALTER TABLE
    training_plans
ADD
    COLUMN IF NOT EXISTS classification_class VARCHAR(255);

COMMENT ON COLUMN training_plans.classification_class IS 'ML-классификация типа тренировки (например endurance_e1e2)';

-- V11__device_providers.sql
-- Таблица для OAuth state (защита от CSRF)
CREATE TABLE IF NOT EXISTS oauth_states (
    state VARCHAR(255) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_states_expires ON oauth_states(expires_at);

-- Таблица для хранения OAuth токенов провайдеров
CREATE TABLE IF NOT EXISTS device_provider_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_expires_at TIMESTAMP WITH TIME ZONE,
    scopes TEXT [],
    webhook_subscription_id VARCHAR(255),
    last_sync_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_provider_accounts_user ON device_provider_accounts(user_id);

CREATE INDEX IF NOT EXISTS idx_provider_accounts_provider ON device_provider_accounts(provider);

-- Лог синхронизаций
CREATE TABLE IF NOT EXISTS device_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_account_id UUID REFERENCES device_provider_accounts(id) ON DELETE CASCADE,
    sync_type VARCHAR(50) NOT NULL,
    records_count INT DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sync_log_provider_account ON device_sync_log(provider_account_id);

-- JWT opaque refresh tokens (access token TTL 15m, refresh TTL 7d)
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

-- V12__oauth_providers.sql
-- OAuth providers support (Google OAuth)
-- Make password optional for OAuth users
ALTER TABLE
    users
ALTER COLUMN
    password_hash DROP NOT NULL;

-- OAuth provider fields
ALTER TABLE
    users
ADD
    COLUMN IF NOT EXISTS provider VARCHAR(50) NOT NULL DEFAULT 'local',
ADD
    COLUMN IF NOT EXISTS external_id VARCHAR(255);

-- Unique index for OAuth lookups
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_provider_external ON users(provider, external_id)
WHERE
    external_id IS NOT NULL;

-- V13__add_totp_2fa.sql
-- Adds TOTP two-factor authentication support
ALTER TABLE
    users
ADD
    COLUMN IF NOT EXISTS totp_secret_encrypted BYTEA;

ALTER TABLE
    users
ADD
    COLUMN IF NOT EXISTS totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE
    users
ADD
    COLUMN IF NOT EXISTS totp_backup_codes_hash TEXT [];

ALTER TABLE
    users
ADD
    COLUMN IF NOT EXISTS totp_backup_codes_remaining INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_users_totp_enabled ON users(totp_enabled)
WHERE
    totp_enabled = TRUE;

COMMENT ON COLUMN users.totp_secret_encrypted IS 'TOTP secret encrypted with AES-256-GCM using TOTP_ENCRYPTION_KEY';

COMMENT ON COLUMN users.totp_backup_codes_hash IS 'Array of SHA-256 hashes of backup codes (10 codes)';