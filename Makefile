imports:
	@echo "Updating Go imports with gci..."
	@go run github.com/daixiang0/gci@latest write \
		-s standard -s default -s "prefix(github.com/MAMUER/project)" \
		--skip-generated --skip-vendor \
		cmd internal pkg
	@echo "Imports updated."

.PHONY: proto tidy fmt vet lint test test-cover check imports js-check
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
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run --max-issues-per-linter=0
	@echo "Lint complete."

test:
	@echo "Running unit tests..."
	go test -v -timeout 5m ./...
	@echo "Tests complete."

test-cover:
 	@echo "Running tests with coverage..."
 	@go test -count=1 -v -coverprofile=coverage.out ./internal/...
 	@echo "Checking coverage threshold (>= 75%)..."
 	@powershell -NoProfile -ExecutionPolicy Bypass -File scripts/coverage-check.ps1
 	go tool cover -html=coverage.out -o coverage.html
 	@echo "Coverage report: coverage.html"

js-check:
	@echo "Checking JavaScript syntax..."
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$env:Path = [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User'); Get-ChildItem web/static/js/*.js | ForEach-Object { node --check $$_.FullName }"
	@echo "JS check complete."

check: tidy fmt vet imports lint test-cover js-check
	@echo "========================================"
	@echo "  LOCAL CHECKS PASSED!"
	@echo "========================================"

proto:
	@echo "Generating proto files..."
	@echo "Требуются: protoc + protoc-gen-go + protoc-gen-go-grpc в PATH (см. docs/ARCHITECTURE.md §8, CONTRIBUTING.md §Протоколы)"
	powershell -Command "if (!(Test-Path 'api/gen/user')) { New-Item -ItemType Directory -Path 'api/gen/user' -Force }"
	powershell -Command "if (!(Test-Path 'api/gen/biometric')) { New-Item -ItemType Directory -Path 'api/gen/biometric' -Force }"
	powershell -Command "if (!(Test-Path 'api/gen/training')) { New-Item -ItemType Directory -Path 'api/gen/training' -Force }"
	powershell -Command "if (!(Test-Path 'api/gen/ml')) { New-Item -ItemType Directory -Path 'api/gen/ml' -Force }"
	@echo "Generating user proto..."
	protoc --proto_path=api/proto --go_out=api/gen/user --go_opt=paths=source_relative --go-grpc_out=api/gen/user --go-grpc_opt=paths=source_relative api/proto/user.proto
	@echo "Generating biometric proto..."
	protoc --proto_path=api/proto --go_out=api/gen/biometric --go_opt=paths=source_relative --go-grpc_out=api/gen/biometric --go-grpc_opt=paths=source_relative api/proto/biometric.proto
	@echo "Generating training proto..."
	protoc --proto_path=api/proto --go_out=api/gen/training --go_opt=paths=source_relative --go-grpc_out=api/gen/training --go-grpc_opt=paths=source_relative api/proto/training.proto
	@echo "Generating ml proto..."
	protoc --proto_path=api/proto --go_out=api/gen/ml --go_opt=paths=source_relative --go-grpc_out=api/gen/ml --go-grpc_opt=paths=source_relative api/proto/ml.proto
	@echo "Proto generation complete"

help:
	@echo "Available commands:"
	@echo "  make tidy       - Tidy Go modules"
	@echo "  make fmt        - Format Go code"
	@echo "  make vet        - Run go vet"
	@echo "  make lint       - Run golangci-lint"
	@echo "  make test       - Run unit tests"
	@echo "  make test-cover - Run tests with coverage report (75% threshold, business logic only)"
	@echo "  make check      - Run tidy, fmt, vet, lint, test, js-check"
	@echo "  make proto      - Generate proto files"
	@echo "  make imports    - Update Go imports with gci"
	@echo "  make js-check   - Check JavaScript syntax with Node.js"