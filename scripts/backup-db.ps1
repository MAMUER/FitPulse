param(
    [string]$BackupDir = "./backups"
)

if (-not $env:BACKUP_KEY) {
    Write-Error "ERROR: BACKUP_KEY environment variable must be set"
    exit 1
}

if (-not $env:PGDATABASE) {
    Write-Error "ERROR: PGDATABASE environment variable must be set"
    exit 1
}

New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null
$timestamp = (Get-Date).ToUniversalTime().ToString("yyyyMMddTHHmmssZ")
$filename = "$BackupDir/backup-$($env:PGDATABASE)-$timestamp.dump"
$encrypted = "$filename.enc"

$env:PGPASSWORD = $env:PGPASSWORD
$pgHost = if ($env:PGHOST) { $env:PGHOST } else { 'localhost' }
$port = if ($env:PGPORT) { $env:PGPORT } else { '5432' }
$user = if ($env:PGUSER) { $env:PGUSER } else { 'postgres' }

pg_dump --format=custom --file="$filename" --host="$pgHost" --port="$port" --username="$user" $env:PGDATABASE
openssl enc -aes-256-cbc -salt -pbkdf2 -pass pass:$env:BACKUP_KEY -in "$filename" -out "$encrypted"
Remove-Item -Force "$filename"
Write-Output "Encrypted backup created: $encrypted"
