#!/usr/bin/env bash
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-./backups}"
BACKUP_KEY="${BACKUP_KEY:-}"
PROMETHEUS_TEXTFILE_DIR="${PROMETHEUS_TEXTFILE_DIR:-./prometheus-textfile-collector}"
BACKUP_TYPE="${BACKUP_TYPE:-full}"

mkdir -p "$BACKUP_DIR"

if [[ -z "${PGDATABASE:-}" ]]; then
    echo "ERROR: PGDATABASE environment variable must be set"
    exit 1
fi

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
FILENAME="${BACKUP_DIR}/backup-${PGDATABASE}-${TIMESTAMP}.dump"
ENCRYPTED="${FILENAME}.enc"

export PGPASSWORD="${PGPASSWORD:-}"

set +e
pg_dump --format=custom --file="$FILENAME" \
    --host="${PGHOST:-localhost}" \
    --port="${PGPORT:-5432}" \
    --username="${PGUSER:-postgres}" \
    "${PGDATABASE}"
PG_DUMP_STATUS=$?
set -e

if [[ $PG_DUMP_STATUS -ne 0 ]]; then
    echo "ERROR: pg_dump failed with exit code $PG_DUMP_STATUS"
    exit $PG_DUMP_STATUS
fi

openssl enc -aes-256-cbc -salt -pbkdf2 -pass pass:"$BACKUP_KEY" -in "$FILENAME" -out "$ENCRYPTED"
rm -f "$FILENAME"

echo "Encrypted backup created: $ENCRYPTED"

if [[ -n "$PROMETHEUS_TEXTFILE_DIR" ]]; then
    mkdir -p "$PROMETHEUS_TEXTFILE_DIR"
    TEXTFILE="$PROMETHEUS_TEXTFILE_DIR/backup_success.prom.$$"
    echo "# TYPE backup_success counter" >"$TEXTFILE"
    echo "backup_success{type=\"$BACKUP_TYPE\",job=\"backup-db\"} 1" >>"$TEXTFILE"
    mv "$TEXTFILE" "${PROMETHEUS_TEXTFILE_DIR}/backup_success.prom"
fi
