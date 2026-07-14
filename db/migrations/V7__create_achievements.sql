-- V7__create_achievements.sql
-- Achievements system

-- Achievements
CREATE TABLE IF NOT EXISTS achievements (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    criteria    JSONB,        -- Config data, acceptable for 1NF
    icon_url    VARCHAR(255),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- User achievements
CREATE TABLE IF NOT EXISTS user_achievements (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    achievement_id  UUID NOT NULL REFERENCES achievements(id) ON DELETE CASCADE,
    earned_at       TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, achievement_id)
);

-- Seed achievements
INSERT INTO achievements (name, description, criteria, icon_url) VALUES
    ('Первый шаг', 'Первая завершенная тренировка', '{"type": "workout_count", "threshold": 1}', '/icons/first_workout.png'),
    ('Десятка', '10 завершенных тренировок', '{"type": "workout_count", "threshold": 10}', '/icons/ten_workouts.png'),
    ('Полтинник', '50 завершенных тренировок', '{"type": "workout_count", "threshold": 50}', '/icons/fifty_workouts.png'),
    ('Сто дней', '100 дней активности', '{"type": "active_days", "threshold": 100}', '/icons/hundred_days.png'),
    ('Мастер спорта', '1000 завершенных тренировок', '{"type": "workout_count", "threshold": 1000}', '/icons/master_sport.png')
ON CONFLICT DO NOTHING;