#!/usr/bin/env bash
set -euo pipefail

# Защита от повторного sourcing и бесконечной рекурсии (исправляет Segmentation fault)
if [[ -n "${__UTILS_SH_LOADED:-}" ]]; then
	return 0
fi
__UTILS_SH_LOADED=1

# === Логирование с таймстампами ===
log_info() { echo "[$(date +%H:%M:%S)] ℹ️  $*"; }
log_warn() { echo "[$(date +%H:%M:%S)] ⚠️  $*" >&2; }
log_error() { echo "[$(date +%H:%M:%S)] ❌ $*" >&2; }

# === Хелперы ===
resource_exists() {
	local resource="$1" ns="${2:-default}"
	kubectl get "$resource" -n "$ns" >/dev/null 2>&1
}

wait_for_condition() {
	local resource="$1" condition="$2" timeout="${3:-300s}" ns="${4:-default}"
	kubectl wait --for="condition=$condition" --timeout="$timeout" \
		"$resource" -n "$ns"
}

wait_for_api() {
	for i in {1..30}; do
		if kubectl get --raw /healthz &>/dev/null; then
			return 0
		fi
		log_info "Attempt $i/30: waiting for API server..."
		sleep 5
	done
	log_error "API server not available after 150s"
	return 1
}
PY
