#!/bin/bash
set -euo pipefail

if [ $# -lt 1 ]; then
	echo "Usage: $0 <VPS_HOST> [OUTPUT_FILE]" >&2
	exit 1
fi

VPS_HOST="$1"
OUTPUT_FILE="${2:-/dev/stdout}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE="${SCRIPT_DIR}/../base/k3s-config-template.yaml"

if [ ! -f "$TEMPLATE" ]; then
	echo "❌ Template not found: $TEMPLATE" >&2
	exit 1
fi

if command -v envsubst &>/dev/null; then
	export VPS_HOST
	envsubst < "$TEMPLATE" > "$OUTPUT_FILE"
else
	sed "s|\${VPS_HOST}|${VPS_HOST}|g" "$TEMPLATE" > "$OUTPUT_FILE"
fi

echo "✅ Config generated: ${OUTPUT_FILE:-stdout}" >&2
