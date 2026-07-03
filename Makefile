imports:
	@echo "Updating Go imports..."
	goimports -w ./cmd ./internal ./pkg ./api
	@echo "Imports updated."

.PHONY: proto tidy fmt vet lint test test-cover check imports
BIN_DIR := bin
GO_VERSION := 1.26.4

tidy:
	@echo "Tidying Go modules..."
	go mod tidy
	@echo "Tidy complete."

fmt:
	@echo "Formatting Go code..."
	go fmt ./...
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
	go test -v -coverprofile=coverage.out ./...
	@echo "Checking coverage threshold (>= 80%)..."
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < 80" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage check failed: $$COVERAGE% (below 80% threshold)"; \
		exit 1; \
	else \
		echo "✅ Coverage: $$COVERAGE%"; \
	fi
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

check: tidy fmt vet lint imports test-cover
	@echo "========================================"
	@echo "  LOCAL CHECKS PASSED!"
	@echo "========================================"

proto:
	@echo "Generating proto files..."
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
	@echo "  make test-cover - Run tests with coverage report (80% threshold)"
	@echo "  make check      - Run tidy, fmt, vet, lint, test"
	@echo "  make proto      - Generate proto files"
	@echo "  make imports    - Update Go imports"
