imports:
	@echo "Updating Go imports with gci..."
	@go run github.com/daixiang0/gci@v0.14.0 write \
		-s standard -s default -s "prefix(github.com/MAMUER/project)" \
		--skip-generated --skip-vendor \
		cmd internal
	@echo "Imports updated."

.PHONY: proto tidy fmt vet lint test test-cover check imports frontend-install frontend-lint frontend-test frontend-build
BIN_DIR := bin
GO_VERSION := 1.26.4

tidy:
	@echo "Tidying Go modules..."
	go mod tidy
	@echo "Tidy complete."

fmt:
	@echo "Formatting Go code..."
	@go fmt ./...
	@echo "Format complete."

vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "Vet complete."

lint:
	@echo "Running golangci-lint..."
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run --max-issues-per-linter=0 ./cmd/... ./internal/...
	@echo "Lint complete."

test:
	@echo "Running unit tests..."
	go test -v -timeout 5m ./...
	@echo "Tests complete."

test-cover:
	@echo "Running tests with coverage..."
	@go test -count=1 -v -coverprofile=coverage.out ./internal/...
	@echo "Checking coverage threshold (>= 75%)..."
	@bash scripts/coverage-check.sh
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

frontend-install:
	@echo "Installing frontend dependencies..."
	cd web && npm install
	@echo "Frontend dependencies installed."

frontend-lint:
	@echo "Running frontend lint..."
	cd web && npm run lint
	@echo "Frontend lint complete."

frontend-test:
	@echo "Running frontend tests..."
	cd web && npm run test
	@echo "Frontend tests complete."

frontend-build:
	@echo "Building frontend..."
	cd web && npm run build
	@echo "Frontend build complete."

check: tidy fmt vet imports lint test-cover proto frontend-install frontend-lint frontend-test frontend-build
	@echo "========================================"
	@echo "  ALL CHECKS PASSED!"
	@echo "========================================"

proto:
	@echo "Generating proto files..."
	@if command -v protoc >/dev/null 2>&1; then \
		echo "Generating user proto..." && \
		protoc --proto_path=api/proto --go_out=api/gen/user --go_opt=paths=source_relative --go-grpc_out=api/gen/user --go-grpc_opt=paths=source_relative api/proto/user.proto && \
		echo "Generating biometric proto..." && \
		protoc --proto_path=api/proto --go_out=api/gen/biometric --go_opt=paths=source_relative --go-grpc_out=api/gen/biometric --go-grpc_opt=paths=source_relative api/proto/biometric.proto && \
		echo "Generating training proto..." && \
		protoc --proto_path=api/proto --go_out=api/gen/training --go_opt=paths=source_relative --go-grpc_out=api/gen/training --go-grpc_opt=paths=source_relative api/proto/training.proto && \
		echo "Generating ml proto..." && \
		protoc --proto_path=api/proto --go_out=api/gen/ml --go_opt=paths=source_relative --go-grpc_out=api/gen/ml --go-grpc_opt=paths=source_relative api/proto/ml.proto && \
		echo "Proto generation complete"; \
	else \
		echo "⚠️  protoc not found, skipping proto generation. Install protoc + protoc-gen-go + protoc-gen-go-grpc to enable."; \
	fi

help:
	@echo "Available commands:"
	@echo "  make tidy            - Tidy Go modules"
	@echo "  make fmt             - Format Go code"
	@echo "  make vet             - Run go vet"
	@echo "  make lint            - Run golangci-lint"
	@echo "  make test            - Run unit tests"
	@echo "  make test-cover      - Run tests with coverage report (75% threshold, business logic only)"
	@echo "  make check           - Run tidy, fmt, vet, lint, test, proto, js-check, frontend-install, frontend-lint, frontend-test, frontend-build"
	@echo "  make proto           - Generate proto files"
	@echo "  make frontend-install - Install frontend dependencies with npm"
	@echo "  make imports         - Update Go imports with gci"
	@echo "  make js-check        - Check JavaScript syntax with Node.js"
	@echo "  make frontend-lint   - Lint frontend code with Biome"
	@echo "  make frontend-test   - Run frontend tests with Vitest"
	@echo "  make frontend-build  - Build frontend with Vite"