@echo off
cd /d "%~dp0.."

echo ========================================
echo  FITPULSE -- FULL SETUP AND RUN
echo ========================================
echo.

echo [1/4] Seeding admin user...
python scripts\seed-admin.py
if %errorlevel% neq 0 (
    echo ERROR: Failed to seed admin user
    pause
    exit /b 1
)
echo.

echo [2/4] Cleaning up existing devices...
docker compose --env-file .env -f deployments/docker-compose.yml exec -T postgres psql -U fitness_admin -d fitness -c "DELETE FROM devices WHERE user_id IN (SELECT id FROM users WHERE email='admin@fitpulse.local');"
echo.

echo [3/4] Building binaries...
call make build
if %errorlevel% neq 0 (
    echo ERROR: Failed to build binaries
    pause
    exit /b 1
)
echo.

echo [4/4] Running device emulator...
call scripts\run-emulator.bat
if %errorlevel% neq 0 (
    echo ERROR: Failed to run emulator
    pause
    exit /b 1
)
echo.

echo [5/5] All done!
echo Admin user: admin@fitpulse.local / Admin@FitPulse2026
echo Access: https://localhost:8443
echo.