@echo off
cd /d "%~dp0.."

echo ========================================
echo  DEVICE EMULATOR RUNNER
echo ========================================

:: Load environment variables
for /f "usebackq tokens=1* delims==" %%a in ("%~dp0..\.env") do (
    if not "%%b"=="" set "%%a=%%b"
)

:: Get admin user UUID
for /f "delims=" %%u in ('docker exec fitness-postgres psql -U fitness_admin -d fitness -t -c "SELECT id FROM users WHERE email='admin@fitpulse.local';"') do set USER_ID=%%u
set USER_ID=%USER_ID: =%

:: Get existing device
for /f "delims=" %%d in ('docker exec fitness-postgres psql -U fitness_admin -d fitness -t -c "SELECT id FROM devices WHERE user_id='%USER_ID%' LIMIT 1;"') do set DEVICE_ID=%%d
set DEVICE_ID=%DEVICE_ID: =%

if "%DEVICE_ID%"=="" (
    echo Creating new device...
    powershell -Command "$id = [guid]::NewGuid().ToString(); $token = [guid]::NewGuid().ToString(); docker exec fitness-postgres psql -U fitness_admin -d fitness -c \"INSERT INTO devices (id, user_id, device_type, token, created_at) VALUES ('$id', '%USER_ID%', 'samsung_galaxy_watch', '$token', NOW() AT TIME ZONE 'UTC');\" | Out-Null; echo $id $token > %TEMP%\device.tmp"
    for /f "tokens=1,2" %%a in (%TEMP%\device.tmp) do (
        set DEVICE_ID=%%a
        set DEVICE_TOKEN=%%b
    )
) else (
    echo Using existing device
    for /f "delims=" %%t in ('docker exec fitness-postgres psql -U fitness_admin -d fitness -t -c "SELECT token FROM devices WHERE id='%DEVICE_ID%';"') do set DEVICE_TOKEN=%%t
    set DEVICE_TOKEN=%DEVICE_TOKEN: =%
)

echo User:       %USER_ID%
echo Device ID:  %DEVICE_ID%
echo.

:: Start emulator
set DEVICE_ID=%DEVICE_ID%
set DEVICE_TOKEN=%DEVICE_TOKEN%

"%~dp0..\bin\device-emulator.exe" --user-id="%USER_ID%" --device-type="samsung_galaxy_watch" --connector-url="http://localhost:8082" --sync-interval=15s --auto-register=false