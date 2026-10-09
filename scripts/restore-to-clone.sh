#!/usr/bin/env bash
set -euo pipefail

# Restore PostgreSQL backup to a clone environment
# Usage: restore-to-clone.sh <backup_file> [recovery_target_time]
#
# Example:
#   restore-to-clone.sh s3://postgres-backups/backup-fitness-20260101_020000.dump.enc
#   restore-to-clone.sh s3://postgres-backups/backup-fitness-20260101_020000.dump.enc "2026-01-01T03:00:00Z"

BACKUP_S3="${1:-}"
RECOVERY_TARGET="${2:-}"

if [[ -z "$BACKUP_S3" ]]; then
	echo "ERROR: Usage: $0 <backup_s3_path> [recovery_target_time]"
	echo "Example: $0 s3://postgres-backups/backup-fitness-20260101_020000.dump.enc"
	echo "Example: $0 s3://postgres-backups/backup-fitness-20260101_020000.dump.enc '2026-01-01T03:00:00Z'"
	exit 1
fi

S3_ENDPOINT="${S3_ENDPOINT:-http://minio.minio.svc.cluster.local:9000}"
S3_BUCKET="${S3_BUCKET:-postgres-backups}"
S3_WAL_PATH="${S3_WAL_PATH:-wal}"
BACKUP_KEY="${BACKUP_KEY:-}"
AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-${MINIO_ROOT_USER:-}}"
AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-${MINIO_ROOT_PASSWORD:-}}"

if [[ -z "$BACKUP_KEY" ]]; then
	echo "ERROR: BACKUP_KEY environment variable must be set for decryption"
	exit 1
fi

if [[ -z "$AWS_ACCESS_KEY_ID" || -z "$AWS_SECRET_ACCESS_KEY" ]]; then
	echo "ERROR: AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set"
	exit 1
fi

export AWS_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY

TEMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TEMP_DIR"' EXIT

echo "Creating temporary directory: $TEMP_DIR"

# Parse backup filename from S3 path
BACKUP_FILENAME="$(basename "$BACKUP_S3")"
if [[ "$BACKUP_S3" == s3://* ]]; then
	# Download from S3
	echo "Downloading backup from S3: $BACKUP_S3"
	aws --endpoint-url "$S3_ENDPOINT" s3 cp "$BACKUP_S3" "${TEMP_DIR}/${BACKUP_FILENAME}"
else
	# Local file
	if [[ ! -f "$BACKUP_S3" ]]; then
		echo "ERROR: Backup file not found: $BACKUP_S3"
		exit 1
	fi
	cp "$BACKUP_S3" "${TEMP_DIR}/${BACKUP_FILENAME}"
fi

# Decrypt backup
echo "Decrypting backup..."
DECRYPTED_FILE="${TEMP_DIR}/backup.dump"
openssl enc -d -aes-256-cbc -salt -pbkdf2 -pass pass:"$BACKUP_KEY" -in "${TEMP_DIR}/${BACKUP_FILENAME}" -out "$DECRYPTED_FILE"

if [[ ! -f "$DECRYPTED_FILE" ]]; then
	echo "ERROR: Decryption failed"
	exit 1
fi

echo "Backup decrypted successfully"

# Download WAL files for recovery
WAL_RECOVERY_DIR="${TEMP_DIR}/wal_recovery"
mkdir -p "$WAL_RECOVERY_DIR"

if [[ -n "$RECOVERY_TARGET" ]]; then
	echo "Downloading WAL files for recovery target: $RECOVERY_TARGET"
	# Download all WAL files from S3
	aws --endpoint-url "$S3_ENDPOINT" s3 sync "s3://${S3_BUCKET}/${S3_WAL_PATH}/" "$WAL_RECOVERY_DIR" --exclude "*" --include "*.wal.gz" --include "*.wal" || true
fi

# Create recovery configuration
RECOVERY_CONF="${TEMP_DIR}/postgresql.conf"
cat >"$RECOVERY_CONF" <<EOF
restore_command = 'cp ${WAL_RECOVERY_DIR}/%f %p'
recovery_target_time = '${RECOVERY_TARGET}'
EOF

echo ""
echo "=== Restore-to-clone preparation completed ==="
echo "Backup file: $DECRYPTED_FILE"
echo "WAL recovery dir: $WAL_RECOVERY_DIR"
echo "Recovery config: $RECOVERY_CONF"
echo ""
echo "To restore to a new PostgreSQL instance:"
echo "  1. Initialize new cluster:"
echo "     initdb -D /tmp/clone_pgdata"
echo ""
echo "  2. Copy recovery config:"
echo "     cp $RECOVERY_CONF /tmp/clone_pgdata/postgresql.conf"
echo ""
echo "  3. Start PostgreSQL in recovery mode:"
echo "     pg_ctl -D /tmp/clone_pgdata -o \"-c restore_command='cp ${WAL_RECOVERY_DIR}/%f %p'\" start"
echo ""
echo "  4. After recovery completes, promote to primary:"
echo "     pg_ctl -D /tmp/clone_pgdata promote"
echo ""
echo "Or use pg_restore for point-in-time recovery:"
echo "  pg_restore --clean --no-owner -d fitness '$DECRYPTED_FILE'"
