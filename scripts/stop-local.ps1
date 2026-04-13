# Fitness Platform - Stop All Services
# Usage: .\stop-local.ps1           — stop, keep data
#        .\stop-local.ps1 --full    — stop + delete volumes (reset DB)

param(
    [switch]$Full
)

# Fix: PowerShell 7 may not have Docker in PATH
$dockerPath = "C:\Program Files\Docker\Docker\resources\bin"
if (-not ($env:PATH -split ";" | Where-Object { $_ -eq $dockerPath })) {
    $env:PATH = "$dockerPath;$env:PATH"
}

$ErrorActionPreference = "Continue"

# Load .env so docker-compose can interpolate variables
$envFile = Join-Path $PSScriptRoot "..\.env"
if (Test-Path $envFile) {
    Get-Content $envFile | Where-Object { $_ -match '^\s*([^#][^=]+)=(.*)$' } | ForEach-Object {
        $key = $Matches[1].Trim()
        $value = $Matches[2].Trim()
        if ($key -and $value) {
            [Environment]::SetEnvironmentVariable($key, $value, "Process")
        }
    }
}

Write-Host "========================================"
Write-Host "   STOPPING ALL SERVICES"
Write-Host "========================================"
Write-Host ""

# Stop processes by saved PIDs
$pidFile = Join-Path $PSScriptRoot ".pids.json"
if (Test-Path $pidFile) {
    Write-Host "[1/3] Stopping Go and Python services..."
    $processes = Get-Content $pidFile | ConvertFrom-Json
    foreach ($proc in $processes.PSObject.Properties) {
        try {
            Stop-Process -Id $proc.Value -Force -ErrorAction SilentlyContinue
            Write-Host "  [OK] Stopped: $($proc.Name) (PID: $($proc.Value))" -ForegroundColor Green
        } catch {
            Write-Host "  [WARN] Not found: $($proc.Name)" -ForegroundColor Yellow
        }
    }
    Remove-Item $pidFile -Force -ErrorAction SilentlyContinue
} else {
    Write-Host "[1/3] PID file not found. Searching for processes..."
    Get-Process -Name "go" -ErrorAction SilentlyContinue | ForEach-Object {
        try {
            Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
            Write-Host "  [OK] Stopped go.exe PID $($_.Id)" -ForegroundColor Green
        } catch {}
    }
}

# Stop Docker containers
Write-Host ""
Write-Host "[2/3] Stopping Docker containers..."
$composeFile = Join-Path $PSScriptRoot "..\deployments\docker-compose.yml"
if ($Full) {
    Write-Host "  [FULL MODE] Deleting volumes (database will be reset)..." -ForegroundColor Red
    docker compose --env-file $envFile -f $composeFile down -v 2>&1
} else {
    docker compose --env-file $envFile -f $composeFile down 2>&1
}
Write-Host "  OK - Containers stopped" -ForegroundColor Green

# Cleanup
Write-Host ""
Write-Host "[3/3] Cleaning up..."
Remove-Item $pidFile -Force -ErrorAction SilentlyContinue
Write-Host "  OK - Cleanup completed" -ForegroundColor Green

Write-Host ""
Write-Host "========================================"
Write-Host "   ALL SERVICES STOPPED!"
Write-Host "========================================"
Write-Host ""
