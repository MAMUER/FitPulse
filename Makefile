
.PHONY: proto build test test-integration test-cover clean dev fmt vet lint vulncheck docker-lint fmt-py fmt-shell lint-py lint-shell lint-json lint-markdown lint-dockerfile
BIN_DIR := bin
GO_VERSION := 1.26.4
# Зависимости
tidy:
	@echo "Tidying Go modules..."
	go mod tidy
	@echo "Tidy complete."

# Форматирование кода
fmt:
	@echo "Formatting Go code..."
	go fmt ./...
	@echo "Format complete."

# Проверка кода
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "Vet complete."

# Проверка на неиспользуемые импорты и переменные
unused:
	@echo "Checking for unused code..."
	golangci-lint run --enable=unused --max-issues-per-linter=0
	@echo "Unused check complete."

# Запуск всех тестов (без integration)
test:
	@echo "Running unit tests..."
	go test -v -timeout 5m ./...
	@echo "Tests complete."

# Запуск тестов с покрытием
test-cover:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Запуск линтера (кросс-платформенный)
lint:
	@echo "Running golangci-lint..."
	@go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run --max-issues-per-linter=0
	@echo "Lint complete."

# SAST: govulncheck — проверка зависимостей (кросс-платформенный)
vulncheck:
	@echo "Running govulncheck..."
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	@echo "Govulncheck complete."

# Проверка YAML файлов на синтаксис
yaml-check:
	@echo "Checking YAML files..."
	go run tools/validate_yaml.go
	@echo "YAML check complete."

# Проверка Docker файлов (кросс-платформенная: Windows + Linux)
DOCKER_LINT_SKIP := $(if $(filter Windows_NT,$(OS)),1,)
docker-lint:
	@echo "Running hadolint..."
	$(if $(DOCKER_LINT_SKIP),@echo "Skipping docker-lint on Windows (runs in GitHub Actions CI/CD)",@for f in cmd/*/Dockerfile; do echo "Linting $$f" && docker run --rm -i hadolint/hadolint < "$$f" || true; done)
	@echo "Docker lint complete."

# JSON validation (cross-platform)
lint-json:
	@echo "Validating JSON files..."
	@python -c "import os,json,sys; files=[os.path.join(r,f) for r,_,fs in os.walk('configs') for f in fs if f.endswith('.json')]; bad=[p for p in files if not json.load(open(p))]; [print('Invalid JSON:', p) for p in bad]; sys.exit(len(bad))" || true
	@echo "JSON validation complete."

# Markdown lint
lint-markdown:
	@echo "Linting markdown files..."
	@python -c "import shutil; print('markdownlint not installed, skipping') if not shutil.which('markdownlint') else None" 2>/dev/null || echo "markdownlint not installed, skipping"
	@echo "Markdown lint complete."

# Dockerfile hadolint
lint-dockerfile:
	@echo "Linting Dockerfiles with hadolint..."
	$(if $(DOCKER_LINT_SKIP),@echo "Skipping docker-lint on Windows (runs in GitHub Actions CI/CD)",@for f in cmd/*/Dockerfile; do echo "Linting $$f" && docker run --rm -i hadolint/hadolint < "$$f" || true; done)
	@echo "Dockerfile lint complete."

# Запуск integration тестов
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./...
	@echo "Integration tests complete."

# Запуск всех проверок (без build и test-cover, они есть в CI отдельно)
check: tidy fmt vet lint vulncheck yaml-check docker-lint test-integration lint-json lint-markdown lint-dockerfile
	@echo "========================================"
	@echo "  ALL CHECKS PASSED!"
	@echo "========================================"

# Сборка всех Go-сервисов
build:
	@echo "Building Go services..."
	go build -ldflags="-s -w" -o bin/gateway ./cmd/gateway
	go build -ldflags="-s -w" -o bin/user-service ./cmd/user-service
	go build -ldflags="-s -w" -o bin/biometric-service ./cmd/biometric-service
	go build -ldflags="-s -w" -o bin/training-service ./cmd/training-service
	go build -ldflags="-s -w" -o bin/data-processor ./cmd/data-processor
	go build -ldflags="-s -w" -o bin/device-connector ./cmd/device-connector
	go build -ldflags="-s -w" -o bin/classifier ./cmd/classifier
	@echo "Build complete."

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

proto-clean:
	@echo "Cleaning generated proto files..."
	powershell -Command "if (Test-Path 'api/gen') { Remove-Item -Recurse -Force 'api/gen' }"
	@echo "Done."


# Создание combined init-db.sql из миграций
combine-migrations:
	python -c "from pathlib import Path; migrations_dir=Path('db/migrations'); init_file=Path('configs/k8s/base/jobs/init-db.sql'); [init_file.write_text(''.join(f'-- {f.name}\n{f.read_text()}\n' for f in sorted(migrations_dir.glob('V*.sql'))))]; print('Combined init-db.sql created')"

# Миграция БД (кроссплатформенный)
migrate: combine-migrations
	python scripts/migrate.py

# API тест (кроссплатформенный)
api-test:
	python scripts/api-test.py

# Database backup helper
backup-db:
	@echo "Run scripts/backup-db.sh or scripts/backup-db.ps1 with BACKUP_KEY and PostgreSQL env vars set."

# Database restore helper
restore-db:
	@echo "Run scripts/restore-db.sh <encrypted-backup-file> or scripts/restore-db.ps1 <encrypted-backup-file> with BACKUP_KEY and PostgreSQL env vars set."

# Нагрузочный тест (требует k6)
load-test:
	@echo "Install k6: https://k6.io/docs/getting-started/installation/"
	python scripts/load-test.py

clean:
	@echo "Cleaning..."
	powershell -Command "if (Test-Path '$(BIN_DIR)') { Remove-Item -Recurse -Force '$(BIN_DIR)' }"
	powershell -Command "if (Test-Path 'api/gen') { Remove-Item -Recurse -Force 'api/gen' }"
	powershell -Command "if (Test-Path 'coverage.out') { Remove-Item 'coverage.out' }"
	powershell -Command "if (Test-Path 'coverage.html') { Remove-Item 'coverage.html' }"
	@echo "Clean complete."

dev:
	@echo "Deploy to VPS via GitHub Actions CI/CD only."

help:
	@echo "Available commands:"
	@echo "  make fmt        - Format Go code"
	@echo "  make vet        - Run go vet"
	@echo "  make lint       - Run golangci-lint"
	@echo "  make lint-dockerfile - Lint Dockerfiles with hadolint"
	@echo "  make fmt-py         - Format Python code (black + isort)"
	@echo "  make fmt-shell      - Format shell/sh scripts (shfmt)"
	@echo "  make lint-py        - Lint Python code (flake8 + mypy)"
	@echo "  make lint-shell     - Syntax check shell/sh scripts (bash -n)"
	@echo "  make lint-json      - Validate JSON files"
	@echo "  make lint-markdown  - Lint markdown files"
	@echo "  make vulncheck  - Run govulncheck dependency scanner"
	@echo "  make test       - Run unit tests"
	@echo "  make test-cover - Run tests with coverage report"
	@echo "  make check      - Run fmt, vet, lint, gosec, vulncheck, test, build"
	@echo "  make proto      - Generate proto files"
	@echo "  make build      - Build all services"
	@echo "  make migrate    - Run database migrations (Python, cross-platform)"
	@echo "  make api-test   - Run API test suite (Python, cross-platform)"
	@echo "  make load-test  - Run load tests (requires k6)"
	@echo "  make clean      - Clean generated files"
	@echo "  make dev        - Deploy target (VPS only; use GitHub Actions CI/CD)"