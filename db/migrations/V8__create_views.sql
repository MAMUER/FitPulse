-- V8__create_views.sql
-- Views for backward compatibility and derived data

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