-- =============================================================================
-- FitPulse — Complete Database Schema (3NF / BCNF)
-- =============================================================================
-- This file creates the entire normalized database schema from scratch.
-- No separate migration files needed — this is the single source of truth.
--
-- Normalization highlights:
--   1NF:  No arrays (TEXT[]) or nested JSONB for business data
--   2NF:  No partial dependencies (doctor data separated from user data)
--   3NF:  No transitive dependencies (polymorphic FK resolved, derived data via VIEWs)
--   BCNF: All determinants are superkeys
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- 1. CORE: Users & Authentication
-- =============================================================================

CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    full_name       VARCHAR(255),
    role            VARCHAR(50) NOT NULL DEFAULT 'client'
                        CHECK (role IN ('client', 'admin', 'doctor')),
    email_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Email verification tokens
CREATE TABLE IF NOT EXISTS email_verifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email       VARCHAR(255) NOT NULL,
    token       VARCHAR(255) UNIQUE NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_verifications_token ON email_verifications(token);
CREATE INDEX IF NOT EXISTS idx_email_verifications_user ON email_verifications(user_id);

-- Invite codes (for doctor/admin registration)
CREATE TABLE IF NOT EXISTS invite_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(100) UNIQUE NOT NULL,
    role        VARCHAR(50) NOT NULL DEFAULT 'doctor',
    specialty   VARCHAR(100),
    max_uses    INT NOT NULL DEFAULT 1,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at  TIMESTAMPTZ,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Invite code usage log (3NF — replaces used_count counter)
CREATE TABLE IF NOT EXISTS invite_code_uses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invite_code_id  UUID NOT NULL REFERENCES invite_codes(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    used_at         TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invite_code_uses_code ON invite_code_uses(invite_code_id);
CREATE INDEX IF NOT EXISTS idx_invite_code_uses_user ON invite_code_uses(user_id);

-- =============================================================================
-- 2. CORE: User Profiles (1NF — no arrays)
-- =============================================================================

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

-- User fitness goals (1NF — normalized from goals TEXT[])
CREATE TABLE IF NOT EXISTS user_goals (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    goal        VARCHAR(100) NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, goal)
);

-- User contraindications (1NF — normalized from contraindications TEXT[])
CREATE TABLE IF NOT EXISTS user_contraindications (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contraindication VARCHAR(255) NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, contraindication)
);

-- =============================================================================
-- 3. BIOMETRIC DATA
-- =============================================================================

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

-- =============================================================================
-- 4. DEVICES (registered by device-connector)
-- =============================================================================

CREATE TABLE IF NOT EXISTS devices (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_type VARCHAR(50) NOT NULL,
    token       VARCHAR(255) UNIQUE NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_devices_user ON devices(user_id);

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

-- =============================================================================
-- 5. TRAINING PLANS (1NF — normalized from plan_data JSONB)
-- =============================================================================

CREATE TABLE IF NOT EXISTS training_plans (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                VARCHAR(255),
    training_goal       VARCHAR(50) CHECK (training_goal IS NULL OR training_goal IN (
        'weight_loss', 'muscle_gain', 'endurance', 'strength', 'flexibility', 'general_fitness'
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

-- Training plan weeks (1NF — from JSONB)
CREATE TABLE IF NOT EXISTS training_plan_weeks (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    training_plan_id        UUID NOT NULL REFERENCES training_plans(id) ON DELETE CASCADE,
    week_number             INT NOT NULL CHECK (week_number > 0),
    total_training_days     INT DEFAULT 0,
    total_duration_minutes  INT DEFAULT 0,
    average_intensity       DECIMAL(3,2),
    UNIQUE (training_plan_id, week_number)
);

-- Training plan days (1NF — from JSONB)
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

-- Individual exercises (1NF — from JSONB)
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

-- Workout completions
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

-- =============================================================================
-- 6. ACHIEVEMENTS
-- =============================================================================

CREATE TABLE IF NOT EXISTS achievements (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    criteria    JSONB,        -- Config data, acceptable for 1NF
    icon_url    VARCHAR(255),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_achievements (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    achievement_id  UUID NOT NULL REFERENCES achievements(id) ON DELETE CASCADE,
    earned_at       TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, achievement_id)
);

-- Seed achievements
INSERT INTO achievements (name, description, criteria) VALUES
    ('Первый шаг', 'Первая завершенная тренировка', '{"type": "workout_count", "threshold": 1}'),
    ('Десятка', '10 завершенных тренировок', '{"type": "workout_count", "threshold": 10}'),
    ('Полтинник', '50 завершенных тренировок', '{"type": "workout_count", "threshold": 50}'),
    ('Сто дней', '100 дней активности', '{"type": "active_days", "threshold": 100}'),
    ('Мастер спорта', '1000 завершенных тренировок', '{"type": "workout_count", "threshold": 1000}')
ON CONFLICT DO NOTHING;

-- =============================================================================
-- 7. DOCTOR SERVICE (2NF / 3NF / BCNF)
-- =============================================================================

-- Doctor profiles (3NF — doctor is a user, linked via user_id)
CREATE TABLE IF NOT EXISTS doctors (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    specialty       VARCHAR(100),
    license_number  VARCHAR(100) UNIQUE,
    phone           VARCHAR(20),
    bio             TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_doctors_specialty ON doctors(specialty);
CREATE INDEX IF NOT EXISTS idx_doctors_active ON doctors(id) WHERE is_active = TRUE;

-- Doctor reviews (3NF — source of truth for rating, not stored in doctors table)
CREATE TABLE IF NOT EXISTS doctor_reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id UUID NOT NULL REFERENCES consultations(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    doctor_id       UUID NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    rating_value    DECIMAL(3,2) NOT NULL CHECK (rating_value >= 0 AND rating_value <= 5),
    comment         TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(consultation_id)
);

CREATE INDEX IF NOT EXISTS idx_doctor_reviews_doctor ON doctor_reviews(doctor_id);
CREATE INDEX IF NOT EXISTS idx_doctor_reviews_user ON doctor_reviews(user_id);

-- Doctor subscriptions (BCNF — no UNIQUE(user_id, doctor_id) to allow history)
CREATE TABLE IF NOT EXISTS doctor_subscriptions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    doctor_id   UUID NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    plan_type   VARCHAR(50) NOT NULL DEFAULT 'monthly'
                    CHECK (plan_type IN ('monthly', 'quarterly', 'yearly')),
    starts_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    price       DECIMAL(10,2),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_doctor ON doctor_subscriptions(user_id, doctor_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_active ON doctor_subscriptions(user_id, doctor_id)
    WHERE is_active = TRUE AND expires_at > NOW();

-- Consultations
CREATE TABLE IF NOT EXISTS consultations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    doctor_id   UUID NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    status      VARCHAR(20) NOT NULL DEFAULT 'scheduled'
                    CHECK (status IN ('scheduled', 'in_progress', 'completed', 'cancelled')),
    scheduled_at TIMESTAMPTZ NOT NULL,
    started_at  TIMESTAMPTZ,
    ended_at    TIMESTAMPTZ,
    notes       TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_consultations_user ON consultations(user_id);
CREATE INDEX IF NOT EXISTS idx_consultations_doctor ON consultations(doctor_id);
CREATE INDEX IF NOT EXISTS idx_consultations_status ON consultations(user_id, doctor_id, status);

-- Consultation messages (3NF — explicit FK instead of polymorphic sender_id/sender_type)
CREATE TABLE IF NOT EXISTS consultation_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    doctor_id       UUID NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    sender_user_id  UUID REFERENCES users(id) ON DELETE SET NULL,
    sender_doctor_id UUID REFERENCES doctors(id) ON DELETE SET NULL,
    message         TEXT NOT NULL,
    message_type    VARCHAR(20) NOT NULL DEFAULT 'text'
                        CHECK (message_type IN ('text', 'image', 'file', 'voice')),
    is_read         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    CHECK (
        (sender_user_id IS NOT NULL AND sender_doctor_id IS NULL) OR
        (sender_user_id IS NULL AND sender_doctor_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_messages_user_doctor ON consultation_messages(user_id, doctor_id, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_sender_user ON consultation_messages(sender_user_id) WHERE sender_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_sender_doctor ON consultation_messages(sender_doctor_id) WHERE sender_doctor_id IS NOT NULL;

-- Doctor prescriptions (3NF — with consultation_id FK and CHECK constraints)
CREATE TABLE IF NOT EXISTS doctor_prescriptions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id     UUID REFERENCES consultations(id) ON DELETE SET NULL,
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    doctor_id           UUID NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    prescription_type   VARCHAR(50) NOT NULL
                            CHECK (prescription_type IN ('recommendation', 'diet_change', 'training_change', 'medication')),
    title               VARCHAR(255) NOT NULL,
    description         TEXT,
    priority            VARCHAR(20) NOT NULL DEFAULT 'normal'
                            CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    status              VARCHAR(20) NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active', 'completed', 'cancelled')),
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_prescriptions_user ON doctor_prescriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_prescriptions_doctor ON doctor_prescriptions(doctor_id);

-- Doctor training modifications (3NF — user_id removed, available via training_plans JOIN)
CREATE TABLE IF NOT EXISTS doctor_training_modifications (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    doctor_id           UUID NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    training_plan_id    UUID NOT NULL REFERENCES training_plans(id) ON DELETE CASCADE,
    modification_type   VARCHAR(50) NOT NULL,
    old_value           JSONB,
    new_value           JSONB,
    reason              TEXT,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_modifications_plan ON doctor_training_modifications(training_plan_id);
CREATE INDEX IF NOT EXISTS idx_modifications_doctor ON doctor_training_modifications(doctor_id);

-- =============================================================================
-- 8. VIEWS (backward compatibility for derived/aggregated data)
-- =============================================================================

-- Doctor statistics (replaces doctors.rating and doctors.consultation_count)
CREATE OR REPLACE VIEW doctor_stats AS
SELECT
    d.id AS doctor_id,
    d.specialty,
    d.is_active,
    COUNT(DISTINCT c.id) FILTER (WHERE c.status IN ('completed', 'in_progress')) AS consultation_count,
    COALESCE(ROUND(AVG(dr.rating_value)::numeric, 2), 0.00) AS avg_rating
FROM doctors d
LEFT JOIN consultations c ON c.doctor_id = d.id
LEFT JOIN doctor_reviews dr ON dr.doctor_id = d.id
GROUP BY d.id, d.specialty, d.is_active;

-- Invite code statistics (replaces invite_codes.used_count)
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

-- User profiles with goals array (backward compatibility for goals TEXT[])
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

-- =============================================================================
-- 9. FUNCTIONS: Invite code management
-- =============================================================================

-- Create a new invite code
-- Parameters: p_role, p_specialty, p_max_uses, p_valid_days
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

-- Validate and consume an invite code (returns is_valid, role, specialty, error_msg)
-- Order matches Go code: Scan(&isValid, &role, &specialty, &errMsg)
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
    -- Lookup code
    SELECT * INTO v_record FROM invite_codes WHERE code = p_code AND is_active = TRUE;

    IF NOT FOUND THEN
        is_valid := FALSE; role := NULL; specialty := NULL; error_msg := 'Invite code not found or inactive';
        RETURN NEXT; RETURN;
    END IF;

    -- Check expiration
    IF v_record.expires_at IS NOT NULL AND v_record.expires_at < NOW() THEN
        is_valid := FALSE; role := NULL; specialty := NULL; error_msg := 'Invite code has expired';
        RETURN NEXT; RETURN;
    END IF;

    -- Check usage count
    SELECT COUNT(*) INTO v_used_count FROM invite_code_uses WHERE invite_code_id = v_record.id;
    IF v_used_count >= v_record.max_uses THEN
        is_valid := FALSE; role := NULL; specialty := NULL; error_msg := 'Invite code has reached its usage limit';
        RETURN NEXT; RETURN;
    END IF;

    -- Valid — return role/specialty (caller must INSERT into invite_code_uses after user creation)
    is_valid := TRUE;
    role := v_record.role;
    specialty := v_record.specialty;
    error_msg := NULL;
    RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

-- Log invite code usage
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

-- =============================================================================
-- Schema complete
-- =============================================================================
