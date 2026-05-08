# Загрузка переменных из .env
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: proto build run test test-integration test-cover docker-up docker-down clean dev fmt vet lint certs

# ... (existing targets)

# Generate self-signed TLS certificates for local development
certs:
	@echo "Generating self-signed TLS certificates..."
	@powershell -Command "if (!(Test-Path 'deploy/tls/certs')) { New-Item -ItemType Directory -Path 'deploy/tls/certs' -Force }"
	openssl req -x509 -nodes -days 365 \
		-newkey rsa:2048 \
		-keyout deploy/tls/certs/server.key \
		-out deploy/tls/certs/server.crt \
		-subj "/C=RU/ST=Moscow/L=Moscow/O=FitnessPlatform/CN=localhost" \
		-addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
	@echo "Certificates generated in deploy/tls/certs/"
	@echo "NOTE: These are self-signed — browsers will show a warning."

BIN_DIR := bin

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

# Запуск линтера
lint:
	@echo "Running golangci-lint..."
	golangci-lint run --max-issues-per-linter=0
	@echo "Lint complete."

# Проверка YAML файлов на синтаксис
yaml-check:
	@echo "Checking YAML files..."
	go run tools/validate_yaml.go
	@echo "YAML check complete."

# Запуск integration тестов
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./...
	@echo "Integration tests complete."

# Запуск всех проверок
check: tidy fmt vet yaml-check lint test test-integration build test-cover
	@echo "========================================"
	@echo "  ALL CHECKS PASSED!"
	@echo "========================================"

# Сборка всех Go-сервисов
build:
	@echo "Building Go services..."
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/user-service ./cmd/user-service
	go build -o bin/biometric-service ./cmd/biometric-service
	go build -o bin/training-service ./cmd/training-service
	go build -o bin/data-processor ./cmd/data-processor
	go build -o bin/device-connector ./cmd/device-connector
	go build -o bin/device-emulator ./cmd/device-emulator
	@echo "Skipping Python-based ML services for Go build target: cmd/ml-classifier, cmd/ml-generator"
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

run: build
	.\bin\gateway.exe

docker-up:
	docker-compose -f deployments/docker-compose.yml up -d

docker-down:
	docker-compose -f deployments/docker-compose.yml down

# Создание combined init-db.sql из миграций
combine-migrations:
	python -c "from pathlib import Path; migrations_dir=Path('db/migrations'); init_file=Path('scripts/init-db.sql'); [init_file.write_text(''.join(f'-- {f.name}\n{f.read_text()}\n\n' for f in sorted(migrations_dir.glob('V*.sql'))))]; print('Combined init-db.sql created')"

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

dev: docker-up
	@echo "Services started. Run 'make run' to start gateway."

help:
	@echo "Available commands:"
	@echo "  make fmt        - Format Go code"
	@echo "  make vet        - Run go vet"
	@echo "  make test       - Run unit tests"
	@echo "  make test-cover - Run tests with coverage report"
	@echo "  make check      - Run fmt, vet, lint, test, build"
	@echo "  make proto      - Generate proto files"
	@echo "  make build      - Build all services"
	@echo "  make run        - Run gateway"
	@echo "  make docker-up  - Start Docker services"
	@echo "  make docker-down - Stop Docker services"
	@echo "  make migrate    - Run database migrations (Python, cross-platform)"
	@echo "  make api-test   - Run API test suite (Python, cross-platform)"
	@echo "  make load-test  - Run load tests (requires k6)"
	@echo "  make clean      - Clean generated files"
	@echo "  make dev        - Start Docker services and run gateway"