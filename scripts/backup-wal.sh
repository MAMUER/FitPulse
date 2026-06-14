#!/usr/bin/env bash
set -euo pipefail

# WAL Backup Script using pg_basebackup and WAL archiving
# Requires: PostgreSQL with wal_level=replica, archive_mode=on

BACKUP_DIR="${BACKUP_DIR:-./backups/wal}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"

mkdir -p "$BACKUP_DIR"

echo "Starting WAL backup at $TIMESTAMP"

# Full backup with pg_basebackup (for PITR)
FULL_BACKUP_DIR="$BACKUP_DIR/full-$TIMESTAMP"
pg_basebackup -D "$FULL_BACKUP_DIR" -Ft -z -P -h "${PGHOST:-localhost}" -p "${PGPORT:-5432}" -U "${PGUSER:-postgres}"

echo "Full backup completed: $FULL_BACKUP_DIR"

# Archive WAL files (this would be called by archive_command in postgresql.conf)
# For demo, we'll simulate archiving recent WAL
WAL_DIR="$BACKUP_DIR/wal"
mkdir -p "$WAL_DIR"

# In production, this is handled by PostgreSQL's archive_command
# Example: archive_command = 'cp %p /path/to/archive/%f'

echo "WAL backup setup complete. Configure postgresql.conf:"
echo "  wal_level = replica"
echo "  archive_mode = on"
echo "  archive_command = 'cp %p $WAL_DIR/%f'"
echo "  restore_command = 'cp $WAL_DIR/%f %p'"
