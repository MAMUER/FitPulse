#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-}"
shift || true

if [[ -z "$MODE" || ("$MODE" != "ssh" && "$MODE" != "scp") ]]; then
	echo "Usage: $0 {ssh|scp} [args...]" >&2
	echo "  ssh:  $0 ssh user@host [command]  (if command omitted, reads from stdin)" >&2
	echo "  scp:  $0 scp [source...] user@host:/remote/path" >&2
	exit 1
fi

# All diagnostics go to stderr: stdout is reserved for remote command output,
# which callers capture with $(...). Writing logs to stdout corrupted values
# such as the detected k3s version and long-lived secrets.
log() {
	printf '%s\n' "$*" >&2
}

# Uploads (scp) are the fragile case: a degraded link makes them stall instead of
# failing, so they get a much shorter per-attempt budget and fewer attempts.
# A 600 s scp timeout x 5 attempts meant a single image could burn ~51 minutes
# before the job finally failed. Env vars still win when explicitly set.
if [[ "$MODE" == "scp" ]]; then
	MAX_ATTEMPTS="${SSH_RETRY_SCP_MAX_ATTEMPTS:-${SSH_RETRY_MAX_ATTEMPTS:-3}}"
	TIMEOUT="${SSH_RETRY_SCP_TIMEOUT:-${SSH_RETRY_TIMEOUT:-180}}"
else
	MAX_ATTEMPTS="${SSH_RETRY_MAX_ATTEMPTS:-5}"
	TIMEOUT="${SSH_RETRY_TIMEOUT:-600}"
fi
DELAY="${SSH_RETRY_DELAY_SECONDS:-15}"
SSH_STRICT_HOST_KEY_CHECKING="${SSH_STRICT_HOST_KEY_CHECKING:-yes}"

if [[ "$SSH_STRICT_HOST_KEY_CHECKING" == "yes" || "$SSH_STRICT_HOST_KEY_CHECKING" == "true" ]]; then
	COMMON_OPTS=(-o BatchMode=yes -o ConnectTimeout=30 -o ServerAliveInterval=60 -o StrictHostKeyChecking=yes)
else
	COMMON_OPTS=(-o BatchMode=yes -o ConnectTimeout=30 -o ServerAliveInterval=60 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null)
fi

if [[ -n "${BASTION_HOST:-}" && -n "${BASTION_USER:-}" ]]; then
	log "-> Using bastion: ${BASTION_USER}@${BASTION_HOST}"
	COMMON_OPTS+=("-o" "ProxyJump=${BASTION_USER}@${BASTION_HOST}")
elif [[ -n "${SSH_JUMP_HOST:-}" ]]; then
	log "-> Using jump host: ${SSH_JUMP_HOST}"
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

# When a command is supplied, run it directly instead of feeding it to
# `bash -s`: this keeps the caller's stdin (e.g. a piped secret) available to
# the remote command. Buffer that stdin so it can be replayed on retries.
STDIN_FILE=""
if [[ "$MODE" == "ssh" && -n "${COMMAND:-}" && ! -t 0 ]]; then
	STDIN_FILE=$(mktemp)
	cat >"$STDIN_FILE"
	trap 'rm -f "$STDIN_FILE"' EXIT
fi

attempt=1
while true; do
	log "--------------------------------------------------"
	log "[$(date '+%H:%M:%S')] Attempt $attempt/$MAX_ATTEMPTS - Mode: $MODE"

	if [[ "$MODE" == "ssh" ]]; then
		if [[ -n "${COMMAND:-}" ]]; then
			log "-> Target : $TARGET"
			log "-> Command: ${COMMAND:0:200}..."
			if [[ -n "$STDIN_FILE" ]]; then
				if timeout "$TIMEOUT" ssh "${COMMON_OPTS[@]}" "$TARGET" "$COMMAND" <"$STDIN_FILE"; then
					log "Success on attempt $attempt"
					exit 0
				fi
			elif timeout "$TIMEOUT" ssh "${COMMON_OPTS[@]}" "$TARGET" "$COMMAND"; then
				log "Success on attempt $attempt"
				exit 0
			fi
		else
			log "-> Target : $TARGET"
			log "-> Reading command from stdin..."
			if timeout "$TIMEOUT" ssh "${COMMON_OPTS[@]}" "$TARGET" bash -s; then
				log "Success on attempt $attempt"
				exit 0
			fi
		fi
	else
		# Для scp последний аргумент — это назначение, все предыдущие — источники
		DEST="${!#}"
		NUM_ARGS=$#
		SOURCES=("${@:1:$((NUM_ARGS - 1))}")

		log "-> Source : ${SOURCES[*]}"
		log "-> Dest   : $DEST"

		if timeout "$TIMEOUT" scp "${COMMON_OPTS[@]}" "${SOURCES[@]}" "$DEST"; then
			log "Success on attempt $attempt"
			exit 0
		fi
	fi

	if [[ $attempt -ge $MAX_ATTEMPTS ]]; then
		log "Failed after $MAX_ATTEMPTS attempts"
		exit 1
	fi

	log "Retrying in ${DELAY}s..."
	sleep "$DELAY"
	attempt=$((attempt + 1))
done
