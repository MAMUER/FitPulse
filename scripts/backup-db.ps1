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
$checksum = "$encrypted.sha256"

$env:PGPASSWORD = $env:PGPASSWORD
$pgHost = if ($env:PGHOST) { $env:PGHOST } else { 'localhost' }
$port = if ($env:PGPORT) { $env:PGPORT } else { '5432' }
$user = if ($env:PGUSER) { $env:PGUSER } else { 'postgres' }

$pgDumpProcess = Start-Process -FilePath "pg_dump" -ArgumentList "--format=custom","--file=$filename","--host=$pgHost","--port=$port","--username=$user",$env:PGDATABASE -NoNewWindow -Wait -PassThru
if ($pgDumpProcess.ExitCode -ne 0) {
    Write-Error "ERROR: pg_dump failed with exit code $($pgDumpProcess.ExitCode)"
    exit $pgDumpProcess.ExitCode
}

openssl enc -aes-256-cbc -salt -pbkdf2 -pass pass:$env:BACKUP_KEY -in "$filename" -out "$encrypted"
$hash = (Get-FileHash -Path "$encrypted" -Algorithm SHA256).Hash
$hash | Out-File -FilePath "$checksum" -Encoding utf8
Remove-Item -Force "$filename"
Write-Output "Encrypted backup created: $encrypted"
Write-Output "Checksum saved: $checksum"
