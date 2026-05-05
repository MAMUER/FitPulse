param(
    [Parameter(Mandatory = $true)]
    [string]$BackupFile
)

if (-not $env:BACKUP_KEY) {
    Write-Error "ERROR: BACKUP_KEY environment variable must be set"
    exit 1
}

if (-not (Test-Path $BackupFile)) {
    Write-Error "ERROR: backup file not found: $BackupFile"
    exit 1
}

$decrypted = [System.IO.Path]::GetTempFileName() + ".dump"
try {
    openssl enc -d -aes-256-cbc -salt -pbkdf2 -pass pass:$env:BACKUP_KEY -in $BackupFile -out $decrypted
    $env:PGPASSWORD = $env:PGPASSWORD
    $pgHost = if ($env:PGHOST) { $env:PGHOST } else { 'localhost' }
    $port = if ($env:PGPORT) { $env:PGPORT } else { '5432' }
    $user = if ($env:PGUSER) { $env:PGUSER } else { 'postgres' }
    $dbname = if ($env:PGDATABASE) { $env:PGDATABASE } else { 'postgres' }
    pg_restore --clean --no-owner --host="$pgHost" --port="$port" --username="$user" --dbname="$dbname" $decrypted
    Write-Output "Restore completed from: $BackupFile"
}
finally {
    if (Test-Path $decrypted) {
        Remove-Item -Force $decrypted
    }
}
