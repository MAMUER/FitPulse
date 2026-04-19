@echo off
cd /d "%~dp0.."

echo ========================================
echo  DEVICE EMULATOR RUNNER
echo ========================================

:: Get admin user UUID via PowerShell
for /f "delims=" %%u in ('powershell -Command "docker exec fitness-postgres psql -U fitness_admin -d fitness -t -c 'SELECT id FROM users WHERE email=''admin@fitpulse.local'''"') do set "USER_ID=%%u"
set USER_ID=%USER_ID: =%

echo User ID: %USER_ID%
echo.

:: Start emulator (auto-register will create device if not exists)
"%~dp0..\bin\device-emulator.exe" --user-id="%USER_ID%" --device-type="samsung_galaxy_watch" --connector-url="http://localhost:8082" --sync-interval=15s --auto-register