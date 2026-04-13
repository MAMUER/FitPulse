@echo off
REM Fitness Platform - Stop All Services
REM Usage: stop-local.bat            — остановить, сохранить данные
REM        stop-local.bat --full     — остановить и удалить volumes (сброс БД)

set SCRIPT_DIR=%~dp0
set "PATH=C:\Program Files\Docker\Docker\resources\bin;%PATH%"

echo ========================================
echo   FITNESS PLATFORM - STOP SERVICES
echo ========================================
echo.

if "%~1"=="--full" (
    echo [FULL MODE] Stopping containers AND deleting volumes...
    echo This will DELETE all database data!
    echo.
    powershell.exe -ExecutionPolicy Bypass -NoProfile -File "%SCRIPT_DIR%stop-local.ps1" --full
) else (
    powershell.exe -ExecutionPolicy Bypass -NoProfile -File "%SCRIPT_DIR%stop-local.ps1"
)

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo ERROR: Script failed with exit code %ERRORLEVEL%
    pause
    exit /b %ERRORLEVEL%
)

pause
