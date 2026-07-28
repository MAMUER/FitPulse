#!/usr/bin/env bash
set -euo pipefail

echo "Generating proto files..."

if ! command -v protoc >/dev/null 2>&1; then
  echo "protoc not found, skipping proto generation. Install protoc + protoc-gen-go + protoc-gen-go-grpc to enable."
  exit 0
fi

protoc --proto_path=api/proto \
  --go_out=api/gen/user --go_opt=paths=source_relative \
  --go-grpc_out=api/gen/user --go-grpc_opt=paths=source_relative \
  api/proto/user.proto

protoc --proto_path=api/proto \
  --go_out=api/gen/biometric --go_opt=paths=source_relative \
  --go-grpc_out=api/gen/biometric --go-grpc_opt=paths=source_relative \
  api/proto/biometric.proto

protoc --proto_path=api/proto \
  --go_out=api/gen/training --go_opt=paths=source_relative \
  --go-grpc_out=api/gen/training --go-grpc_opt=paths=source_relative \
  api/proto/training.proto

protoc --proto_path=api/proto \
  --go_out=api/gen/ml --go_opt=paths=source_relative \
  --go-grpc_out=api/gen/ml --go-grpc_opt=paths=source_relative \
  api/proto/ml.proto

echo "Proto generation complete"
