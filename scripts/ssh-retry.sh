#!/bin/bash
#
# scripts/ssh-retry.sh
# Reusable SSH/SCP wrapper with retries, timeouts, and good logging.
#
set -euo pipefail

MODE="${1:-}"
shift || true

if [[ -z "$MODE" || ( "$MODE" != "ssh" && "$MODE" != "scp" ) ]]; then
  echo "Usage: $0 {ssh|scp} [args...]"
  echo "  ssh:  $0 ssh user@host 'remote command here'"
  echo "  scp:  $0 scp /local/path user@host:/remote/path"
  exit 1
fi

MAX_ATTEMPTS="${SSH_RETRY_MAX_ATTEMPTS:-5}"
DELAY="${SSH_RETRY_DELAY_SECONDS:-15}"
TIMEOUT="${SSH_RETRY_TIMEOUT:-600}"

# Common safe options
COMMON_OPTS="-o BatchMode=yes -o ConnectTimeout=30 -o ServerAliveInterval=60"

# Bastion / Jump host support
if [[ -n "${BASTION_HOST:-}" && -n "${BASTION_USER:-}" ]]; then
  echo "-> Using bastion: ${BASTION_USER}@${BASTION_HOST}"
  COMMON_OPTS="$COMMON_OPTS -o ProxyJump=${BASTION_USER}@${BASTION_HOST}"
elif [[ -n "${SSH_JUMP_HOST:-}" ]]; then
  echo "-> Using jump host: ${SSH_JUMP_HOST}"
  COMMON_OPTS="$COMMON_OPTS -o ProxyJump=${SSH_JUMP_HOST}"
fi

# Merge with user-provided options
if [[ -n "${SSH_OPTS:-}" ]]; then
  COMMON_OPTS="$COMMON_OPTS $SSH_OPTS"
fi

# Parse arguments
TARGET=""
COMMAND=""
SRC=""
DEST=""
if [[ "$MODE" == "ssh" ]]; then
  TARGET="${1:-}"
  shift || true
  COMMAND="$*"
else
  SRC="${1:-}"
  DEST="${2:-}"
fi

attempt=1
while true; do
  echo "--------------------------------------------------"
  echo "[$(date '+%H:%M:%S')] Attempt $attempt/$MAX_ATTEMPTS - Mode: $MODE"

  if [[ "$MODE" == "ssh" ]]; then
    echo "-> Target : $TARGET"
    echo "-> Command: ${COMMAND:0:200}..."

    # ИСПРАВЛЕНИЕ: передаём скрипт через stdin в bash -s
    # Это работает с любыми командами: многострочными, с кавычками, пайпами и т.д.
    if printf '%s\n' "$COMMAND" | timeout "$TIMEOUT" ssh $COMMON_OPTS "$TARGET" bash -s; then
      echo "Success on attempt $attempt"
      exit 0
    fi
  else
    echo "-> Source : $SRC"
    echo "-> Dest   : $DEST"

    if timeout "$TIMEOUT" scp $COMMON_OPTS "$SRC" "$DEST"; then
      echo "Success on attempt $attempt"
      exit 0
    fi
  fi

  if [[ $attempt -ge $MAX_ATTEMPTS ]]; then
    echo "Failed after $MAX_ATTEMPTS attempts"
    exit 1
  fi

  echo "Retrying in ${DELAY}s..."
  sleep "$DELAY"
  attempt=$((attempt + 1))
done