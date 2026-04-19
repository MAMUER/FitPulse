param(
    [string]$DeviceType = "apple_watch",
    [int]$Interval = 30
)

Get-Content .env | ForEach-Object {
    if ($_ -match '^\s*([^#=]+)=(.*)$') {
        [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim().Trim('"', "'"), "Process")
    }
}

# Get admin user UUID
$userId = docker exec fitness-postgres psql -U fitness_admin -d fitness -t -c "SELECT id FROM users WHERE email='admin@fitpulse.local';"
$userId = $userId.Trim()

# Get existing device for this user
$deviceId = docker exec fitness-postgres psql -U fitness_admin -d fitness -t -c "SELECT id FROM devices WHERE user_id='$userId' LIMIT 1;"
$deviceId = $deviceId.Trim()

if ([string]::IsNullOrEmpty($deviceId)) {
    # No device found, register new one
    $deviceId = [guid]::NewGuid().ToString()
    $deviceToken = [guid]::NewGuid().ToString()
    docker exec fitness-postgres psql -U fitness_admin -d fitness -c "INSERT INTO devices (id, user_id, device_type, token, created_at) VALUES ('$deviceId', '$userId', '$DeviceType', '$deviceToken', NOW() AT TIME ZONE 'UTC';"
} else {
    # Get token for existing device
    $deviceToken = docker exec fitness-postgres psql -U fitness_admin -d fitness -t -c "SELECT token FROM devices WHERE id='$deviceId';"
    $deviceToken = $deviceToken.Trim()
}

$env:DEVICE_ID = $deviceId
$env:DEVICE_TOKEN = $deviceToken

.\bin\device-emulator.exe --user-id="$userId" --device-type="$DeviceType" --connector-url="http://localhost:8082" --sync-interval="$Interval`s" --auto-register=$false