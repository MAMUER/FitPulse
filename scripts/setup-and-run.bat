@echo off
cd /d "%~dp0.."

echo ========================================
echo  FITPULSE -- FULL SETUP AND RUN
echo ========================================
echo.

:: [1/4] Seed admin user via direct SQL (Python script is unreliable)
echo [1/4] Seeding admin user...
set "ADMIN_HASH=$2a$10$m4Czlz7sDoa34XNwdf8Jz.kduIXBQ.hPurUCjj1elF6eGBEwq4DkC"
docker compose --env-file .env -f deployments/docker-compose.yml exec -T postgres psql -U fitness_admin -d fitness -c "INSERT INTO users (id, email, password_hash, full_name, role, email_confirmed, created_at, updated_at) VALUES (gen_random_uuid(), 'admin@fitpulse.local', '%ADMIN_HASH%', 'System Administrator', 'admin', TRUE, NOW(), NOW()) ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash, updated_at = NOW();"
if %errorlevel% neq 0 (
    echo ERROR: Failed to seed admin user
    pause
    exit /b 1
)
echo.

:: [2/4] Clean up existing devices
echo [2/4] Cleaning up existing devices...
docker compose --env-file .env -f deployments/docker-compose.yml exec -T postgres psql -U fitness_admin -d fitness -c "DELETE FROM devices WHERE user_id IN (SELECT id FROM users WHERE email='admin@fitpulse.local');" >nul 2>&1
echo.

:: [3/4] Run device emulator via `go run` (no local build needed)
echo [3/4] Running device emulator...

:: Get admin user UUID
for /f "delims=" %%u in ('powershell -Command "docker exec fitness-postgres psql -U fitness_admin -d fitness -t -c 'SELECT id FROM users WHERE email=''admin@fitpulse.local'''"') do set "USER_ID=%%u"
set USER_ID=%USER_ID: =%

echo User ID: %USER_ID%
echo.

go run ./cmd/device-emulator --user-id="%USER_ID%" --device-type="samsung_galaxy_watch" --connector-url="http://localhost:8082" --sync-interval=15s --auto-register
if %errorlevel% neq 0 (
    echo.
    echo ERROR: Emulator exited with code %errorlevel%
    pause
    exit /b 1
)
echo.

echo [4/4] All done!
echo Admin user: admin@fitpulse.local / Admin@FitPulse2026
echo Access: https://localhost:8443
echo.
pause