#!/usr/bin/env bash
set -euo pipefail

BACKUP_KEY="${BACKUP_KEY:-}"
BACKUP_FILE="${1:-}"

if [[ -z "$BACKUP_KEY" ]]; then
    echo "ERROR: BACKUP_KEY environment variable must be set"
    exit 1
fi

if [[ -z "$BACKUP_FILE" ]]; then
    echo "Usage: $0 <encrypted-backup-file>"
    exit 1
fi

if [[ ! -f "$BACKUP_FILE" ]]; then
    echo "ERROR: backup file not found: $BACKUP_FILE"
    exit 1
fi

DECRYPTED_FILE="$(mktemp --suffix=.dump)"
trap 'rm -f "$DECRYPTED_FILE"' EXIT

openssl enc -d -aes-256-cbc -salt -pbkdf2 -pass pass:"$BACKUP_KEY" -in "$BACKUP_FILE" -out "$DECRYPTED_FILE"

export PGPASSWORD="${PGPASSWORD:-}"

pg_restore --clean --no-owner --host="${PGHOST:-localhost}" --port="${PGPORT:-5432}" --username="${PGUSER:-postgres}" --dbname="${PGDATABASE:-postgres}" "$DECRYPTED_FILE"

echo "Restore completed from: $BACKUP_FILE"
