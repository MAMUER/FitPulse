@echo off
REM ============================================================================
REM Seed Admin User Script — FitPulse
REM ============================================================================
REM Creates the initial admin user (email_confirmed = TRUE) and invite codes.
REM Runs via Docker exec — no psql required on host.
REM
REM Usage:
REM   scripts\seed-admin.bat
REM   scripts\seed-admin.bat --email admin@example.com --password MyAdmin123!
REM ============================================================================

setlocal enabledelayedexpansion

REM === Configuration ===
set "DB_USER=fitness_admin"
set "DB_NAME=fitness"
set "ADMIN_EMAIL=admin@fitpulse.local"
set "ADMIN_PASSWORD=Admin@FitPulse2026"
set "ADMIN_NAME=System Administrator"
set "COMPOSE_FILE=deployments/docker-compose.yml"

REM Allow override via environment variables
if not "%SEED_ADMIN_EMAIL%"=="" set "ADMIN_EMAIL=%SEED_ADMIN_EMAIL%"
if not "%SEED_ADMIN_PASSWORD%"=="" set "ADMIN_PASSWORD=%SEED_ADMIN_PASSWORD%"

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
    echo Creates initial admin user (email_confirmed = TRUE) and invite codes.
    echo Default: admin@fitpulse.local / Admin@FitPulse2026
    echo.
    echo Environment variables override:
    echo   SEED_ADMIN_EMAIL, SEED_ADMIN_PASSWORD
    exit /b 0
)
shift
goto :parse_args
:done_args

echo.
echo ========================================
echo   FITPULSE -- ADMIN SEED SCRIPT
echo ========================================
echo.
echo   Admin:    %ADMIN_EMAIL%
echo   DB:       %DB_NAME%
echo.

REM === Check Docker ===
docker ps >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Docker is not running!
    exit /b 1
)

REM === Check PostgreSQL container ===
docker compose -f "%COMPOSE_FILE%" ps postgres --format "json" >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] PostgreSQL container not found. Run: scripts\run-local.ps1
    exit /b 1
)

REM === Generate bcrypt hash via Go ===
echo [1/3] Generating bcrypt hash...
set "TEMP_DIR=%TEMP%\fitpulse_seed_%RANDOM%"
mkdir "%TEMP_DIR%" 2>nul

> "%TEMP_DIR%\hash.go" echo package main
>> "%TEMP_DIR%\hash.go" echo import ("fmt"; "os"; "golang.org/x/crypto/bcrypt")
>> "%TEMP_DIR%\hash.go" echo func main() {
>> "%TEMP_DIR%\hash.go" echo     h, _ := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost)
>> "%TEMP_DIR%\hash.go" echo     fmt.Print(string(h))
>> "%TEMP_DIR%\hash.go" echo }

cd /d "%~dp0\.."
go run "%TEMP_DIR%\hash.go" "%ADMIN_PASSWORD%" > "%TEMP_DIR%\hash.txt" 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] Failed to generate bcrypt hash. Make sure Go is installed.
    rmdir /s /q "%TEMP_DIR%" 2>nul
    exit /b 1
)
set /p BCRYPT_HASH=<"%TEMP_DIR%\hash.txt"
rmdir /s /q "%TEMP_DIR%" 2>nul

REM === Build and execute SQL via docker exec ===
echo [2/3] Creating admin user in database...

REM Escape single quotes for SQL
set "ESCAPED_EMAIL=%ADMIN_EMAIL:'=''%"
set "ESCAPED_NAME=%ADMIN_NAME:'=''%"

REM Build SQL (use double quotes for bcrypt hash to avoid $ interpretation)
set "SQL=INSERT INTO users (email, password_hash, full_name, role, email_confirmed) SELECT '%ESCAPED_EMAIL%', '%BCRYPT_HASH%', '%ESCAPED_NAME%', 'admin', TRUE WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = '%ESCAPED_EMAIL%');"

set "SQL2=INSERT INTO user_profiles (user_id) SELECT id FROM users WHERE email = '%ESCAPED_EMAIL%' AND NOT EXISTS (SELECT 1 FROM user_profiles WHERE user_id = (SELECT id FROM users WHERE email = '%ESCAPED_EMAIL%'));"

set "SQL3=INSERT INTO invite_codes (code, role, specialty, max_uses, is_active) SELECT 'ADMIN-BOOTSTRAP-' || substr(md5(random()::text), 1, 8), 'admin', NULL, 10, TRUE WHERE NOT EXISTS (SELECT 1 FROM invite_codes WHERE code LIKE 'ADMIN-BOOTSTRAP-%%');"

set "SQL4=INSERT INTO invite_codes (code, role, specialty, max_uses, is_active) SELECT 'DOCTOR-BOOTSTRAP-' || substr(md5(random()::text), 1, 8), 'doctor', NULL, 50, TRUE WHERE NOT EXISTS (SELECT 1 FROM invite_codes WHERE code LIKE 'DOCTOR-BOOTSTRAP-%%');"

set "SQL5=SELECT email, full_name, role, email_confirmed, created_at FROM users WHERE email = '%ESCAPED_EMAIL%';"

set "SQL6=SELECT code, role, max_uses, is_active, created_at FROM invite_codes WHERE code LIKE '%%BOOTSTRAP%%' ORDER BY role, created_at DESC;"

docker compose -f "%COMPOSE_FILE%" exec -T postgres psql -U "%DB_USER%" -d "%DB_NAME%" -c "%SQL%" -c "%SQL2%" -c "%SQL3%" -c "%SQL4%" -c "%SQL5%" -c "%SQL6%" 2>&1

if %errorlevel% neq 0 (
    echo.
    echo [ERROR] Seed script failed.
    exit /b 1
)

echo.
echo ========================================
echo   ADMIN USER READY
echo ========================================
echo.
echo   Email:    %ADMIN_EMAIL%
echo   Password: %ADMIN_PASSWORD%
echo.
echo   Login via API:
echo   curl -k -X POST https://localhost:8443/api/v1/login ^
echo     -H "Content-Type: application/json" ^
echo     -d "{\"email\":\"%ADMIN_EMAIL%\",\"password\":\"%ADMIN_PASSWORD%\"}"
echo.
echo   NOTE: Change the default password after first login!
echo ========================================
