#!/bin/bash
#
# scripts/ssh-retry.sh
# Reusable SSH/SCP wrapper with retries, timeouts, and good logging.
#
# Usage:
#   ./scripts/ssh-retry.sh ssh user@host "remote command"
#   ./scripts/ssh-retry.sh scp /local/file user@host:/remote/path
#
# Environment variables (optional):
#   SSH_RETRY_MAX_ATTEMPTS   (default: 5)
#   SSH_RETRY_DELAY_SECONDS  (default: 10)
#   SSH_RETRY_TIMEOUT        (default: 120)
#   SSH_OPTS                 (additional ssh options, e.g. "-o StrictHostKeyChecking=accept-new")
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
DELAY="${SSH_RETRY_DELAY_SECONDS:-10}"
TIMEOUT="${SSH_RETRY_TIMEOUT:-120}"

# Common safe options
COMMON_OPTS="-o BatchMode=yes -o ConnectTimeout=30 -o ServerAliveInterval=60"

# Bastion / Jump host support via ProxyJump (recommended)
  if [[ -n "${BASTION_HOST:-}" && -n "${BASTION_USER:-}" ]]; then
    echo "→ Using bastion: ${BASTION_USER}@${BASTION_HOST}"
    COMMON_OPTS="$COMMON_OPTS -o ProxyJump=${BASTION_USER}@${BASTION_HOST}"
  elif [[ -n "${SSH_JUMP_HOST:-}" ]]; then
    echo "→ Using jump host: ${SSH_JUMP_HOST}"
    COMMON_OPTS="$COMMON_OPTS -o ProxyJump=${SSH_JUMP_HOST}"
  fi

# Merge with user-provided options if any
if [[ -n "${SSH_OPTS:-}" ]]; then
  COMMON_OPTS="$COMMON_OPTS $SSH_OPTS"
fi

attempt=1
while true; do
  echo "────────────────────────────────────────────────────────"
  echo "[$(date '+%H:%M:%S')] Attempt $attempt/$MAX_ATTEMPTS — Mode: $MODE"

  if [[ "$MODE" == "ssh" ]]; then
    TARGET="$1"
    shift
    COMMAND="$*"

    echo "→ Target : $TARGET"
    echo "→ Command: $COMMAND"

    if timeout "$TIMEOUT" ssh $COMMON_OPTS "$TARGET" "$COMMAND"; then
      echo "✅ Success on attempt $attempt"
      exit 0
    fi
  else
    # SCP mode
    SRC="$1"
    DEST="$2"

    echo "→ Source : $SRC"
    echo "→ Dest   : $DEST"

    if timeout "$TIMEOUT" scp $COMMON_OPTS "$SRC" "$DEST"; then
      echo "✅ Success on attempt $attempt"
      exit 0
    fi
  fi

  if [[ $attempt -ge $MAX_ATTEMPTS ]]; then
    echo "❌ Failed after $MAX_ATTEMPTS attempts"
    exit 1
  fi

  echo "⏳ Retrying in ${DELAY}s..."
  sleep "$DELAY"
  attempt=$((attempt + 1))
done
