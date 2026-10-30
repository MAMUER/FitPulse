#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-}"
shift || true

if [[ -z "$MODE" || ("$MODE" != "ssh" && "$MODE" != "scp") ]]; then
	echo "Usage: $0 {ssh|scp} [args...]"
	echo "  ssh:  $0 ssh user@host [command]  (if command omitted, reads from stdin)"
	echo "  scp:  $0 scp [source...] user@host:/remote/path"
	exit 1
fi

MAX_ATTEMPTS="${SSH_RETRY_MAX_ATTEMPTS:-5}"
DELAY="${SSH_RETRY_DELAY_SECONDS:-15}"
TIMEOUT="${SSH_RETRY_TIMEOUT:-600}"
SSH_STRICT_HOST_KEY_CHECKING="${SSH_STRICT_HOST_KEY_CHECKING:-yes}"

if [[ "$SSH_STRICT_HOST_KEY_CHECKING" == "yes" || "$SSH_STRICT_HOST_KEY_CHECKING" == "true" ]]; then
	COMMON_OPTS=(-o BatchMode=yes -o ConnectTimeout=30 -o ServerAliveInterval=60 -o StrictHostKeyChecking=yes)
else
	COMMON_OPTS=(-o BatchMode=yes -o ConnectTimeout=30 -o ServerAliveInterval=60 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null)
fi

if [[ -n "${BASTION_HOST:-}" && -n "${BASTION_USER:-}" ]]; then
	echo "-> Using bastion: ${BASTION_USER}@${BASTION_HOST}"
	COMMON_OPTS+=("-o" "ProxyJump=${BASTION_USER}@${BASTION_HOST}")
elif [[ -n "${SSH_JUMP_HOST:-}" ]]; then
	echo "-> Using jump host: ${SSH_JUMP_HOST}"
	COMMON_OPTS+=("-o" "ProxyJump=${SSH_JUMP_HOST}")
fi

if [[ -n "${VPS_PORT:-}" ]]; then
	if [[ "$MODE" == "scp" ]]; then
		COMMON_OPTS+=("-P" "${VPS_PORT}")
	else
		COMMON_OPTS+=("-p" "${VPS_PORT}")
	fi
fi

if [[ -n "${SSH_OPTS:-}" ]]; then
	read -r -a EXTRA_SSH_OPTS <<<"$SSH_OPTS"
	COMMON_OPTS+=("${EXTRA_SSH_OPTS[@]}")
fi

# Extract target before loop (so retries work)
if [[ "$MODE" == "ssh" ]]; then
	TARGET="${1:-}"
	shift || true
	COMMAND="${*:-}"
fi

attempt=1
while true; do
	echo "--------------------------------------------------"
	echo "[$(date '+%H:%M:%S')] Attempt $attempt/$MAX_ATTEMPTS - Mode: $MODE"

	if [[ "$MODE" == "ssh" ]]; then
		if [[ -n "${COMMAND:-}" ]]; then
			echo "-> Target : $TARGET"
			echo "-> Command: ${COMMAND:0:200}..."
			if printf '%s\n' "$COMMAND" | timeout "$TIMEOUT" ssh "${COMMON_OPTS[@]}" "$TARGET" bash -s; then
				echo "Success on attempt $attempt"
				exit 0
			fi
		else
			echo "-> Target : $TARGET"
			echo "-> Reading command from stdin..."
			if timeout "$TIMEOUT" ssh "${COMMON_OPTS[@]}" "$TARGET" bash -s; then
				echo "Success on attempt $attempt"
				exit 0
			fi
		fi
	else
		# Для scp последний аргумент — это назначение, все предыдущие — источники
		DEST="${!#}"
		NUM_ARGS=$#
		SOURCES=("${@:1:$((NUM_ARGS - 1))}")

		echo "-> Source : ${SOURCES[*]}"
		echo "-> Dest   : $DEST"

		if timeout "$TIMEOUT" scp "${COMMON_OPTS[@]}" "${SOURCES[@]}" "$DEST"; then
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
