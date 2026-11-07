#!/bin/bash
set -euo pipefail

run_smoke_tests() {
	go test -v -tags=smoke -timeout 10m ./internal/testcontainers/...
}

run_fuzz_tests() {
	go test -fuzz=FuzzString -fuzztime=120s ./internal/sanitize/
	go test -fuzz=FuzzAESGCMEncryptor -fuzztime=120s ./internal/crypto/
}

run_uat_staging() {
	python scripts/api-test.py --base-url "$BASE_URL" || echo "⚠️ API tests failed" >&2
}

run_uat_production() {
	python scripts/api-test.py --base-url "$TARGET"
}

main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all)
		run_smoke_tests
		run_fuzz_tests
		run_uat_staging
		run_uat_production
		;;
	run_smoke_tests) run_smoke_tests ;;
	run_fuzz_tests) run_fuzz_tests ;;
	run_uat_staging) run_uat_staging ;;
	run_uat_production) run_uat_production ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
