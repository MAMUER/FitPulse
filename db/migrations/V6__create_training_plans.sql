-- V6__create_training_plans.sql
-- Training plans and related data

-- Training plans
CREATE TABLE IF NOT EXISTS training_plans (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                VARCHAR(255),
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

-- Training plan weeks (1NF — normalized from JSONB)
CREATE TABLE IF NOT EXISTS training_plan_weeks (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    training_plan_id        UUID NOT NULL REFERENCES training_plans(id) ON DELETE CASCADE,
    week_number             INT NOT NULL CHECK (week_number > 0),
    total_training_days     INT DEFAULT 0,
    total_duration_minutes  INT DEFAULT 0,
    average_intensity       DECIMAL(3,2),
    UNIQUE (training_plan_id, week_number)
);

-- Training plan days (1NF — normalized from JSONB)
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

-- Individual exercises (1NF — normalized from JSONB)
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