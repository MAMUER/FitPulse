.PHONY: proto tidy fmt vet lint test check imports frontend-install frontend-lint frontend-test frontend-build coverage build clean pip-compile swag tf-init tf-plan tf-apply tf-fmt
BIN_DIR := bin
GO_VERSION := 1.27.0

imports:
	@echo "Updating Go imports with gci..."
	@go install github.com/daixiang0/gci@v0.14.0
	@gci write -s standard -s default -s 'prefix(github.com/MAMUER/project)' --skip-generated --skip-vendor cmd internal || echo "gci failed, continuing without import reorganization"
	@echo "Imports step finished."

tidy:
	@echo "Tidying Go modules..."
	@go mod tidy
	@echo "Tidy complete."

pip-compile:
	@echo "Compiling Python requirements..."
	@python -m pip install --quiet --upgrade pip-tools
	@cd cmd/ml_generator && python -m piptools compile --strip-extras --output-file=requirements.lock.txt requirements.txt
	@echo "Pip-compile complete."

fmt:
	@echo "Formatting Go code..."
	@go fmt ./...
	@echo "Format complete."

vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "Vet complete."

lint:
	@echo "Running golangci-lint..."
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.0 run --max-issues-per-linter=0 ./cmd/... ./internal/...
	@echo "Lint complete."

test:
	@echo "Running unit tests..."
	@go test -short -v -timeout 5m ./...
	@echo "Tests complete."

coverage:
	@echo "Generating Go coverage..."
	@mkdir -p $(BIN_DIR)
	@go test -short -covermode=atomic -coverprofile="$(BIN_DIR)/coverage.out" ./...
	@echo "Coverage report generated at $(BIN_DIR)/coverage.out"

build:
	@echo "Building Go binaries into $(BIN_DIR)/..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/gateway.exe ./cmd/gateway
	@go build -o $(BIN_DIR)/user-service.exe ./cmd/user-service
	@go build -o $(BIN_DIR)/training-service.exe ./cmd/training-service
	@go build -o $(BIN_DIR)/biometric-service.exe ./cmd/biometric-service
	@go build -o $(BIN_DIR)/classifier.exe ./cmd/classifier
	@go build -o $(BIN_DIR)/device-aggregator.exe ./cmd/device-aggregator
	@go build -o $(BIN_DIR)/data-processor.exe ./cmd/data-processor
	@go build -o $(BIN_DIR)/admin-cli.exe ./cmd/admin-cli
	@echo "Build complete. Binaries are in $(BIN_DIR)/"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@find . -maxdepth 1 -type f -name '*.exe' -delete
	@find . -maxdepth 1 -type f -name '*.test' -delete
	@echo "Clean complete."

check: tidy fmt vet lint frontend-install frontend-build
	@echo "========================================"
	@echo "  ALL CHECKS PASSED!"
	@echo "========================================"

proto:
	@echo "Generating proto files..."
	@bash scripts/proto.sh

swag:
	@echo "Generating Swagger docs..."
	@go install github.com/swaggo/swag/cmd/swag@v1.16.6
	@swag init -g cmd/gateway/main.go -o api/rest --parseDependency --parseInternal

frontend-install:
	@echo "Installing frontend dependencies..."
	@npm --prefix web ci --legacy-peer-deps
	@echo "Frontend dependencies installed."

frontend-lint:
	@echo "Running frontend lint..."
	@npm --prefix web run lint
	@echo "Frontend lint complete."

frontend-test:
	@echo "Running frontend tests..."
	@npm --prefix web run test
	@echo "Frontend tests complete."

frontend-build:
	@echo "Building frontend..."
	@npm --prefix web run build
	@echo "Frontend build complete."

tf-init:
	cd terraform && terraform init

tf-fmt:
	cd terraform && terraform fmt -check -recursive

tf-plan:
	cd terraform && terraform plan -var-file="production.tfvars"

tf-apply:
	cd terraform && terraform apply -var-file="production.tfvars"


help:
	@echo "Available commands:"
	@echo "  make tidy            - Tidy Go modules"
	@echo "  make fmt             - Format Go code"
	@echo "  make vet             - Run go vet"
	@echo "  make lint            - Run golangci-lint"
	@echo "  make test            - Run unit tests"
	@echo "  make check           - Run tidy, fmt, vet, lint, coverage, frontend-install, frontend-build, pip-compile"
	@echo "  make build           - Build all Go binaries into bin/"
	@echo "  make clean           - Remove bin/ and stray .exe/.test files"
	@echo "  make proto           - Generate proto files"
	@echo "  make frontend-install - Install frontend dependencies with npm"
	@echo "  make imports         - Update Go imports with gci"
	@echo "  make pip-compile     - Compile Python requirements with pip-compile"
	@echo "  make coverage         - Generate Go coverage report in bin/coverage.out"
	@echo "  make js-check        - Check JavaScript syntax with Node.js"
	@echo "  make frontend-lint   - Lint frontend code with Biome"
	@echo "  make frontend-test   - Run frontend tests with Vitest"
	@echo "  make frontend-build  - Build frontend with Vite"
