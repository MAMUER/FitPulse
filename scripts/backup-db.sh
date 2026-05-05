#!/usr/bin/env bash
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-./backups}"
BACKUP_KEY="${BACKUP_KEY:-}"

if [[ -z "$BACKUP_KEY" ]]; then
  echo "ERROR: BACKUP_KEY environment variable must be set"
  exit 1
fi

mkdir -p "$BACKUP_DIR"

if [[ -z "${PGDATABASE:-}" ]]; then
  echo "ERROR: PGDATABASE environment variable must be set"
  exit 1
fi

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
FILENAME="${BACKUP_DIR}/backup-${PGDATABASE}-${TIMESTAMP}.dump"
ENCRYPTED="${FILENAME}.enc"

export PGPASSWORD="${PGPASSWORD:-}"

pg_dump --format=custom --file="$FILENAME" \
  --host="${PGHOST:-localhost}" \
  --port="${PGPORT:-5432}" \
  --username="${PGUSER:-postgres}" \
  "${PGDATABASE}"

openssl enc -aes-256-cbc -salt -pbkdf2 -pass pass:"$BACKUP_KEY" -in "$FILENAME" -out "$ENCRYPTED"
rm -f "$FILENAME"

echo "Encrypted backup created: $ENCRYPTED"
