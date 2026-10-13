#!/bin/bash
set -euo pipefail

lint_frontend() {
	npm config set registry https://registry.npmjs.org
	NPM_CONFIG_MIN_RELEASE_AGE=0 npm ci --ignore-scripts --legacy-peer-deps
	npm run lint
}

lint_go() {
	for i in 1 2 3; do
		if go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0; then # NOSONAR
			break
		fi
		echo "golangci-lint install attempt $i failed; retrying..."
		sleep 5
	done
	go mod tidy
	go vet ./...
	go fmt ./...
	golangci-lint run ./...
}

super_linter() {
	echo "Super Linter is handled via GitHub Action (super-linter/super-linter)"
	echo "No local script needed — this function is a placeholder for composite dispatch"
}

main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all)
		lint_frontend
		lint_go
		super_linter
		;;
	lint_frontend) lint_frontend ;;
	lint_go) lint_go ;;
	super_linter) super_linter ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
