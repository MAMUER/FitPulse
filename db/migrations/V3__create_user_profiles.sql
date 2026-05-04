-- V3__create_user_profiles.sql
-- User profiles and related data

-- User profiles
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