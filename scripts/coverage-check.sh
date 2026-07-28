#!/usr/bin/env bash
set -euo pipefail

coverage_file="coverage.out"

if [ ! -f "$coverage_file" ]; then
  echo "ERROR: coverage.out not found. Run 'go test -coverprofile=coverage.out ./...' first."
  exit 1
fi

total_statements=0
total_covered=0

while IFS= read -r line; do
  [ -z "$line" ] && continue
  [[ "$line" == mode:* ]] && continue
  [[ "$line" == *"/api/gen/"* ]] && continue
  [[ "$line" == *"/mocks/"* ]] && continue
  [[ "$line" == *"/internal/grpc/"* ]] && continue
  [[ "$line" == *"/internal/db/"* ]] && continue
  [[ "$line" == *"/internal/queue/"* ]] && continue
  [[ "$line" == *"/internal/middleware/"* ]] && continue
  [[ "$line" == *"/internal/crypto/"* ]] && continue
  [[ "$line" == *"/internal/totp/"* ]] && continue
  [[ "$line" == *"/internal/telemetry/"* ]] && continue
  [[ "$line" == *"/internal/testcontainers/"* ]] && continue

  num_stmts=$(echo "$line" | awk '{print $(NF-1)}')
  count=$(echo "$line" | awk '{print $NF}')

  if [ -n "$num_stmts" ] && [ -n "$count" ]; then
    total_statements=$((total_statements + num_stmts))
    if [ "$count" -gt 0 ]; then
      total_covered=$((total_covered + num_stmts))
    fi
  fi
done < "$coverage_file"

if [ "$total_statements" -eq 0 ]; then
  echo "ERROR: No business-logic statements found in coverage.out"
  exit 1
fi

coverage=$(awk "BEGIN {printf \"%.2f\", ($total_covered / $total_statements) * 100}")
echo "Coverage: ${coverage}% of statements (business logic only, excluding generated/mocks/infrastructure)"

if awk "BEGIN {exit !($coverage < 75)}"; then
  echo "ERROR: Coverage threshold (>= 75%) not met. Current: ${coverage}%"
  exit 1
fi

exit 0
