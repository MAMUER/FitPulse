@echo off
REM Fitness Platform - Local Startup Script
REM Auto-fixes Docker PATH for PowerShell 7

set SCRIPT_DIR=%~dp0
set PROJECT_DIR=%SCRIPT_DIR%..

REM Add Docker to PATH
set "PATH=C:\Program Files\Docker\Docker\resources\bin;%PATH%"

echo ========================================
echo   FITNESS PLATFORM - LOCAL STARTUP
echo ========================================
echo.

powershell.exe -ExecutionPolicy Bypass -NoProfile -File "%SCRIPT_DIR%run-local.ps1"

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo ERROR: Script failed with exit code %ERRORLEVEL%
    echo.
    echo If Docker is not running, start Docker Desktop first.
    pause
    exit /b %ERRORLEVEL%
)

pause
