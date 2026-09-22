# Restore PostgreSQL backup to a clone environment (Windows PowerShell)
# Usage: .\restore-to-clone.ps1 -BackupS3 <s3_path> [-RecoveryTargetTime <ISO8601>]
#
# Example:
#   .\restore-to-clone.ps1 -BackupS3 "s3://postgres-backups/backup-fitness-20260101_020000.dump.enc"
#   .\restore-to-clone.ps1 -BackupS3 "s3://postgres-backups/backup-fitness-20260101_020000.dump.enc" -RecoveryTargetTime "2026-01-01T03:00:00Z"

param(
    [Parameter(Mandatory=$true)]
    [string]$BackupS3,
    
    [Parameter(Mandatory=$false)]
    [string]$RecoveryTargetTime = ""
)

$ErrorActionPreference = "Stop"

$S3Endpoint = $env:S3_ENDPOINT -replace 'minio.minio.svc.cluster.local:9000', 'localhost:9000'
if (-not $S3Endpoint) {
    $S3Endpoint = "http://localhost:9000"
}
$S3Bucket = $env:S3_BUCKET -replace 'postgres-backups', 'postgres-backups'
if (-not $S3Bucket) {
    $S3Bucket = "postgres-backups"
}
$S3WalPath = $env:S3_WAL_PATH -replace 'wal', 'wal'
if (-not $S3WalPath) {
    $S3WalPath = "wal"
}
$BackupKey = $env:BACKUP_KEY
$AwsAccessKeyId = $env:AWS_ACCESS_KEY_ID
$AwsSecretAccessKey = $env:AWS_SECRET_ACCESS_KEY

if (-not $BackupKey) {
    Write-Error "BACKUP_KEY environment variable must be set for decryption"
    exit 1
}

if (-not $AwsAccessKeyId -or -not $AwsSecretAccessKey) {
    Write-Error "AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set"
    exit 1
}

$env:AWS_ACCESS_KEY_ID = $AwsAccessKeyId
$env:AWS_SECRET_ACCESS_KEY = $AwsSecretAccessKey

$TempDir = Join-Path $env:TEMP ("restore-clone-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    Write-Host "Creating temporary directory: $TempDir"
    
    # Parse backup filename from S3 path
    $BackupFilename = Split-Path -Path $BackupS3 -Leaf
    $LocalBackup = Join-Path $TempDir $BackupFilename
    
    if ($BackupS3 -like "s3://*") {
        # Download from S3
        Write-Host "Downloading backup from S3: $BackupS3"
        aws --endpoint-url $S3Endpoint s3 cp $BackupS3 $LocalBackup --only-show-errors
    } else {
        # Local file
        if (-not (Test-Path $BackupS3)) {
            Write-Error "Backup file not found: $BackupS3"
            exit 1
        }
        Copy-Item $BackupS3 $LocalBackup
    }
    
    # Decrypt backup
    Write-Host "Decrypting backup..."
    $DecryptedFile = Join-Path $TempDir "backup.dump"
    
    openssl enc -d -aes-256-cbc -salt -pbkdf2 -pass pass:$BackupKey -in $LocalBackup -out $DecryptedFile
    
    if (-not (Test-Path $DecryptedFile)) {
        Write-Error "Decryption failed"
        exit 1
    }
    
    Write-Host "Backup decrypted successfully"
    
    # Download WAL files for recovery
    $WalRecoveryDir = Join-Path $TempDir "wal_recovery"
    New-Item -ItemType Directory -Path $WalRecoveryDir -Force | Out-Null
    
    if ($RecoveryTargetTime) {
        Write-Host "Downloading WAL files for recovery target: $RecoveryTargetTime"
        aws --endpoint-url $S3Endpoint s3 sync "s3://${S3Bucket}/${S3WalPath}/" $WalRecoveryDir --exclude "*" --include "*.wal.gz" --include "*.wal" | Out-Null
    }
    
    # Create recovery configuration
    $RecoveryConf = Join-Path $TempDir "postgresql.conf"
    $RecoveryContent = @"
restore_command = 'cp ${WalRecoveryDir}/%f %p'
recovery_target_time = '${RecoveryTargetTime}'
"@
    Set-Content -Path $RecoveryConf -Value $RecoveryContent
    
    Write-Host ""
    Write-Host "=== Restore-to-clone preparation completed ==="
    Write-Host "Backup file: $DecryptedFile"
    Write-Host "WAL recovery dir: $WalRecoveryDir"
    Write-Host "Recovery config: $RecoveryConf"
    Write-Host ""
    Write-Host "To restore to a new PostgreSQL instance:"
    Write-Host "  1. Initialize new cluster:"
    Write-Host "     initdb -D /tmp/clone_pgdata"
    Write-Host ""
    Write-Host "  2. Copy recovery config:"
    Write-Host "     cp $RecoveryConf /tmp/clone_pgdata/postgresql.conf"
    Write-Host ""
    Write-Host "  3. Start PostgreSQL in recovery mode:"
    Write-Host "     pg_ctl -D /tmp/clone_pgdata -o `"-c restore_command='cp ${WalRecoveryDir}/%f %p'`" start"
    Write-Host ""
    Write-Host "  4. After recovery completes, promote to primary:"
    Write-Host "     pg_ctl -D /tmp/clone_pgdata promote"
    Write-Host ""
    Write-Host "Or use pg_restore for point-in-time recovery:"
    Write-Host "  pg_restore --clean --no-owner -d fitness '$DecryptedFile'"
}
finally {
    Write-Host "Cleaning up temporary directory: $TempDir"
    Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}
