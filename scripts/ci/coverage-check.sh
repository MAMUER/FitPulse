#!/usr/bin/env bash
set -euo pipefail

# Unified Go coverage accounting — the single source of truth for both
# ci-helpers.sh:run_coverage (SonarCloud job, report-only) and the Tests job
# (threshold enforcement), plus `make coverage-check`.
#
# Usage: coverage-check.sh [coverage_file]] [min_coverage]]
#   min_coverage=0 or empty  -> report only, always exits 0
#   default                  -> COVERAGE_THRESHOLD or 80

coverage_file="${1:-coverage.out}"
min_coverage="${2:-${COVERAGE_THRESHOLD:-80}}"

# Only generated protobuf stubs are excluded from the total: they are pure
# boilerplate with 0% coverage and they hide the real coverage of everything
# else. Business logic is NOT excluded, so newly added tests always move the number.
EXCLUDE='(^|/)api/gen/'

if [[ ! -f "$coverage_file" ]]; then
	echo "ERROR: $coverage_file not found. Run 'go test -coverprofile=$coverage_file ./...' first."
	exit 1
fi

read -r total covered < <(awk -v exclude="$EXCLUDE" '
	/^mode:/ { next }
	{
		file = $1
		sub(/:.*/, "", file)
		if (file ~ exclude) next
		total += $(NF-1)
		if ($NF > 0) covered += $(NF-1)
	}
	END { print total + 0, covered + 0 }
' "$coverage_file")

if [[ "$total" -eq 0 ]]; then
	echo "ERROR: no statements found in $coverage_file"
	exit 1
fi

coverage=$(awk -v c="$covered" -v t="$total" 'BEGIN { printf "%.2f", (c / t) * 100 }')
echo "Coverage: ${coverage}% of statements ($covered/$total, excluding generated api/gen)"

if [[ "$min_coverage" = "0" ]] || [[ -z "$min_coverage" ]]; then
	echo "Coverage threshold check skipped (report-only mode)"
	exit 0
fi

if awk -v c="$coverage" -v threshold="$min_coverage" 'BEGIN { exit !(c < threshold) }'; then
	echo "ERROR: coverage ${coverage}% is below the ${min_coverage}% threshold"
	exit 1
fi

echo "Coverage threshold met: ${coverage}% >= ${min_coverage}%"
