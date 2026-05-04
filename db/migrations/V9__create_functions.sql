-- V9__create_functions.sql
-- Functions for invite code management

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