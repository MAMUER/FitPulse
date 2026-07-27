#!/usr/bin/env bash
set -euo pipefail

COVERAGE_FILE="coverage.out"

if [ ! -f "$COVERAGE_FILE" ]; then
	echo "coverage.out not found. Run 'go test -coverprofile=coverage.out ./...' first."
	exit 1
fi

total_statements=0
total_covered=0

while IFS= read -r line; do
	line="${line#"${line%%[![:space:]]*}"}"
	if [[ "$line" =~ ^mode: ]]; then
		continue
	fi
	if [[ -z "$line" ]]; then
		continue
	fi

	if [[ "$line" == *"/api/gen/"* ]]; then
		continue
	fi
	if [[ "$line" == *"/mocks/"* ]]; then
		continue
	fi
	if [[ "$line" == *"/internal/grpc/"* ]]; then
		continue
	fi
	if [[ "$line" == *"/internal/db/"* ]]; then
		continue
	fi
	if [[ "$line" == *"/internal/queue/"* ]]; then
		continue
	fi
	if [[ "$line" == *"/internal/middleware/"* ]]; then
		continue
	fi
	if [[ "$line" == *"/internal/crypto/"* ]]; then
		continue
	fi
	if [[ "$line" == *"/internal/totp/"* ]]; then
		continue
	fi
	if [[ "$line" == *"/internal/telemetry/"* ]]; then
		continue
	fi
	if [[ "$line" == *"/internal/testcontainers/"* ]]; then
		continue
	fi
	if [[ "$line" == *"/cmd/"* ]]; then
		continue
	fi

	num_stmts=$(echo "$line" | awk '{print $(NF-1)}')
	count=$(echo "$line" | awk '{print $NF}')

	if [[ -z "$num_stmts" || -z "$count" ]]; then
		continue
	fi

	total_statements=$((total_statements + num_stmts))
	if [ "$count" -gt 0 ]; then
		total_covered=$((total_covered + num_stmts))
	fi
done <"$COVERAGE_FILE"

if [ "$total_statements" -eq 0 ]; then
	echo "No business-logic statements found in coverage.out"
	exit 1
fi

coverage=$(awk "BEGIN {print ($total_covered / $total_statements) * 100}")
coverage_rounded=$(awk "BEGIN {printf \"%.2f\", $coverage}")

echo "Coverage: $coverage_rounded% of statements (business logic only, excluding generated/mocks/infrastructure)"

if awk "BEGIN {exit !($coverage < 75)}"; then
	echo "Coverage threshold (>= 75%) not met. Current: $coverage_rounded%"
	exit 1
fi

exit 0
