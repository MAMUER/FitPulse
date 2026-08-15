imports:
	@echo "Updating Go imports with gci..."
	@go run github.com/daixiang0/gci@v0.14.0 write -s standard -s default -s "prefix(github.com/MAMUER/project)" --skip-generated --skip-vendor cmd internal
	@echo "Imports updated."

.PHONY: proto tidy fmt vet lint test check imports frontend-install frontend-lint frontend-test frontend-build coverage build clean
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
	@go test -v -timeout 5m ./...
	@echo "Tests complete."

coverage:
	@echo "Generating Go coverage..."
	@powershell -Command "New-Item -ItemType Directory -Force -Path coverage | Out-Null"
	@go test -coverprofile=coverage/coverage.out ./...
	@echo "Generating frontend coverage..."
	@cd web && npm run test
	@echo "Frontend coverage complete."

build:
	@echo "Building Go binaries into $(BIN_DIR)/..."
	@powershell -Command "New-Item -ItemType Directory -Force -Path $(BIN_DIR) | Out-Null"
	@go build -o $(BIN_DIR)/gateway.exe ./cmd/gateway
	@go build -o $(BIN_DIR)/user-service.exe ./cmd/user-service
	@go build -o $(BIN_DIR)/training-service.exe ./cmd/training-service
	@go build -o $(BIN_DIR)/biometric-service.exe ./cmd/biometric-service
	@go build -o $(BIN_DIR)/classifier.exe ./cmd/classifier
	@go build -o $(BIN_DIR)/device-aggregator.exe ./cmd/device-aggregator
	@go build -o $(BIN_DIR)/data-processor.exe ./cmd/data-processor
	@echo "Build complete. Binaries are in $(BIN_DIR)/"

clean:
	@echo "Cleaning build artifacts..."
	@powershell -Command "Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $(BIN_DIR)"
	@powershell -Command "Get-ChildItem -Path . -Filter *.exe -File | Remove-Item -Force"
	@powershell -Command "Get-ChildItem -Path . -Filter *.test -File | Remove-Item -Force"
	@echo "Clean complete."

check: tidy fmt vet imports lint coverage frontend-install frontend-lint frontend-build
	@echo "========================================"
	@echo "  ALL CHECKS PASSED!"
	@echo "========================================"

proto:
	@echo "Generating proto files..."
	@bash scripts/proto.sh

frontend-install:
	@echo "Installing frontend dependencies..."
	@cd web && npm install --legacy-peer-deps
	@echo "Frontend dependencies installed."

frontend-lint:
	@echo "Running frontend lint..."
	@cd web && npm run lint
	@echo "Frontend lint complete."

frontend-test:
	@echo "Running frontend tests..."
	@cd web && npm run test
	@echo "Frontend tests complete."

frontend-build:
	@echo "Building frontend..."
	@cd web && npm run build
	@echo "Frontend build complete."

help:
	@echo "Available commands:"
	@echo "  make tidy            - Tidy Go modules"
	@echo "  make fmt             - Format Go code"
	@echo "  make vet             - Run go vet"
	@echo "  make lint            - Run golangci-lint"
	@echo "  make test            - Run unit tests"
	@echo "  make check           - Run tidy, fmt, vet, lint, coverage, frontend-install, frontend-lint, frontend-build"
	@echo "  make build           - Build all Go binaries into bin/"
	@echo "  make clean           - Remove bin/ and stray .exe/.test files"
	@echo "  make proto           - Generate proto files"
	@echo "  make frontend-install - Install frontend dependencies with npm"
	@echo "  make imports         - Update Go imports with gci"
	@echo "  make coverage         - Generate Go and frontend coverage reports for SonarCloud"
	@echo "  make js-check        - Check JavaScript syntax with Node.js"
	@echo "  make frontend-lint   - Lint frontend code with Biome"
	@echo "  make frontend-test   - Run frontend tests with Vitest"
	@echo "  make frontend-build  - Build frontend with Vite"
