@echo off
REM ============================================================================
REM Seed Admin User Script — FitPulse
REM ============================================================================
REM Creates the initial admin user and generates invite codes.
REM Run AFTER `make migrate` (database schema must exist).
REM
REM Usage:
REM   scripts\seed-admin.bat
REM   scripts\seed-admin.bat --email admin@example.com --password MyAdmin123!
REM ============================================================================

setlocal enabledelayedexpansion

REM === Configuration ===
set "DB_HOST=%DB_HOST:=localhost%"
set "DB_PORT=%DB_PORT:=localhost%"
if "%DB_PORT%"=="" set "DB_PORT=5432"
set "DB_USER=%DB_USER: postgres%"
if "%DB_USER%"=="" set "DB_USER=postgres"
set "DB_PASSWORD=%DB_PASSWORD: postgres%"
if "%DB_PASSWORD%"=="" set "DB_PASSWORD=postgres"
set "DB_NAME=%DB_NAME: fitness%"
if "%DB_NAME%"=="" set "DB_NAME=fitness"

set "ADMIN_EMAIL=admin@fitpulse.local"
set "ADMIN_PASSWORD=Admin@FitPulse2026"
set "ADMIN_NAME=System Administrator"

REM Parse command line arguments
:parse_args
if "%~1"=="" goto :done_args
if "%~1"=="--email" (
    set "ADMIN_EMAIL=%~2"
    shift & shift
    goto :parse_args
)
if "%~1"=="--password" (
    set "ADMIN_PASSWORD=%~2"
    shift & shift
    goto :parse_args
)
if "%~1"=="--help" (
    echo Usage: %0 [--email EMAIL] [--password PASSWORD]
    echo.
    echo Creates initial admin user and invite codes.
    echo Default: admin@fitpulse.local / Admin@FitPulse2026
    exit /b 0
)
shift
goto :parse_args
:done_args

echo.
echo ========================================
echo   FITPULSE — ADMIN SEED SCRIPT
echo ========================================
echo.
echo   DB:       %DB_HOST%:%DB_PORT%/%DB_NAME%
echo   User:     %DB_USER%
echo   Admin:    %ADMIN_EMAIL%
echo.
echo   Press Ctrl+C to cancel, or
pause

REM === Check psql ===
where psql >nul 2>&1
if %errorlevel% neq 0 (
    echo.
    echo [ERROR] psql not found. Install PostgreSQL client or use Docker:
    echo   docker compose -f deployments/docker-compose.yml exec postgres psql -U postgres -d fitness
    exit /b 1
)

REM === SQL Script ===
set "TEMP_SQL=%TEMP%\fitpulse_seed_admin_%RANDOM%.sql"

REM Generate bcrypt hash using Go one-liner
set "TEMP_HASH=%TEMP%\fitpulse_hash_%RANDOM%.txt"
(
    echo package main
    echo import ("fmt"; "os"; "golang.org/x/crypto/bcrypt")
    echo func main() {
    echo     h, _ := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost)
    echo     fmt.Print(string(h))
    echo }
) > "%TEMP_HASH%\hash.go"

echo [0/3] Generating bcrypt hash...
cd /d "%~dp0\.."
go run "%TEMP_HASH%\hash.go" "%ADMIN_PASSWORD%" > "%TEMP_HASH%\hash.txt" 2>nul
if %errorlevel% neq 0 (
    echo.
    echo [WARNING] Could not generate bcrypt hash via Go.
    echo [INFO] Please create admin user manually via API registration.
    del "%TEMP_HASH%\hash.go" 2>nul
    del "%TEMP_HASH%\hash.txt" 2>nul
    rmdir "%TEMP_HASH%" 2>nul
    exit /b 1
)
set /p BCRYPT_HASH=<"%TEMP_HASH%\hash.txt"
del "%TEMP_HASH%\hash.go" 2>nul
del "%TEMP_HASH%\hash.txt" 2>nul
rmdir "%TEMP_HASH%" 2>nul

(
    echo BEGIN;
    echo.
    echo -- 1. Create admin user (bypass invite requirement for bootstrap)
    echo DO $$
    echo DECLARE
    echo     v_user_id UUID;
    echo BEGIN
    echo     -- Check if admin already exists
    echo     IF EXISTS (SELECT 1 FROM users WHERE email = '%ADMIN_EMAIL%') THEN
    echo         RAISE NOTICE 'Admin user already exists: %%', '%ADMIN_EMAIL%';
    echo     ELSE
    echo         -- Create admin user with confirmed email
    echo         INSERT INTO users (email, password_hash, full_name, role, email_confirmed)
    echo         VALUES (
    echo             '%ADMIN_EMAIL%',
    echo             '%BCRYPT_HASH%',
    echo             '%ADMIN_NAME%',
    echo             'admin',
    echo             TRUE
    echo         )
    echo         RETURNING id INTO v_user_id;
    echo.
    echo         RAISE NOTICE 'Admin user created with ID: %%', v_user_id;
    echo.
    echo         -- Create user profile
    echo         INSERT INTO user_profiles (user_id) VALUES (v_user_id);
    echo     END IF;
    echo.
    echo     -- 2. Generate invite codes for future use
    echo     IF NOT EXISTS (SELECT 1 FROM invite_codes WHERE role = 'admin' AND code LIKE 'ADMIN-BOOTSTRAP-%%') THEN
    echo         INSERT INTO invite_codes (code, role, specialty, max_uses, is_active)
    echo         VALUES (
    echo             'ADMIN-BOOTSTRAP-' || substr(md5(random()::text), 1, 8),
    echo             'admin',
    echo             NULL,
    echo             10,
    echo             TRUE
    echo         );
    echo         RAISE NOTICE 'Admin invite codes generated';
    echo     END IF;
    echo.
    echo     IF NOT EXISTS (SELECT 1 FROM invite_codes WHERE role = 'doctor' AND code LIKE 'DOCTOR-BOOTSTRAP-%%') THEN
    echo         INSERT INTO invite_codes (code, role, specialty, max_uses, is_active)
    echo         VALUES (
    echo             'DOCTOR-BOOTSTRAP-' || substr(md5(random()::text), 1, 8),
    echo             'doctor',
    echo             NULL,
    echo             50,
    echo             TRUE
    echo         );
    echo         RAISE NOTICE 'Doctor invite codes generated';
    echo     END IF;
    echo END $$;
    echo.
    echo -- 3. Show results
    echo SELECT email, full_name, role, email_confirmed, created_at
    echo FROM users WHERE email = '%ADMIN_EMAIL%';
    echo.
    echo SELECT code, role, max_uses, is_active, created_at
    echo FROM invite_codes WHERE code LIKE '%%BOOTSTRAP%%'
    echo ORDER BY role, created_at DESC;
    echo.
    echo COMMIT;
) > "%TEMP_SQL%"

REM === Execute ===
echo.
echo [1/2] Running seed script...
echo.

set "PGPASSWORD=%DB_PASSWORD%"
psql -h "%DB_HOST%" -p "%DB_PORT%" -U "%DB_USER%" -d "%DB_NAME%" -f "%TEMP_SQL%" 2>&1

if %errorlevel% neq 0 (
    echo.
    echo [ERROR] Seed script failed.
    del "%TEMP_SQL%" 2>nul
    exit /b 1
)

REM === Cleanup ===
del "%TEMP_SQL%" 2>nul

echo.
echo ========================================
echo   ADMIN USER READY
echo ========================================
echo.
echo   Email:    %ADMIN_EMAIL%
echo   Password: %ADMIN_PASSWORD%
echo.
echo   Login via API:
echo   curl -X POST http://localhost:8080/api/v1/login ^
echo     -H "Content-Type: application/json" ^
echo     -d "{\"email\":\"%ADMIN_EMAIL%\",\"password\":\"%ADMIN_PASSWORD%\"}"
echo.
echo   IMPORTANT: Change the default password after first login!
echo.
echo ========================================
