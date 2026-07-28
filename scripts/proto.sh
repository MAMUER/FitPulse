#!/usr/bin/env bash
set -euo pipefail

echo "Generating proto files..."

# Find protoc - try multiple methods for cross-platform compatibility
PROTOC_CMD=""

# Method 1: standard command -v (works on Linux/macOS/Git Bash with proper PATH)
if command -v protoc >/dev/null 2>&1; then
    PROTOC_CMD="protoc"
# Method 2: Windows 'where' command (works in Git Bash on Windows)
elif command -v where >/dev/null 2>&1; then
    PROTOC_PATH=$(where protoc 2>/dev/null | head -n1)
    if [ -n "$PROTOC_PATH" ]; then
        PROTOC_CMD="$PROTOC_PATH"
    fi
# Method 3: which (Linux/macOS fallback)
elif command -v which >/dev/null 2>&1; then
    PROTOC_PATH=$(which protoc 2>/dev/null)
    if [ -n "$PROTOC_PATH" ]; then
        PROTOC_CMD="$PROTOC_PATH"
    fi
fi

if [ -z "$PROTOC_CMD" ]; then
    echo "Error: protoc not found in PATH. Install protoc and ensure it's in PATH for bash."
    echo "On Windows: ensure protoc is in system PATH and run 'make' from Git Bash terminal."
    exit 1
fi

echo "Using protoc: $PROTOC_CMD"

"$PROTOC_CMD" --proto_path=api/proto \
  --go_out=api/gen/user --go_opt=paths=source_relative \
  --go-grpc_out=api/gen/user --go-grpc_opt=paths=source_relative \
  api/proto/user.proto

"$PROTOC_CMD" --proto_path=api/proto \
  --go_out=api/gen/biometric --go_opt=paths=source_relative \
  --go-grpc_out=api/gen/biometric --go-grpc_opt=paths=source_relative \
  api/proto/biometric.proto

"$PROTOC_CMD" --proto_path=api/proto \
  --go_out=api/gen/training --go_opt=paths=source_relative \
  --go-grpc_out=api/gen/training --go-grpc_opt=paths=source_relative \
  api/proto/training.proto

"$PROTOC_CMD" --proto_path=api/proto \
  --go_out=api/gen/ml --go_opt=paths=source_relative \
  --go-grpc_out=api/gen/ml --go-grpc_opt=paths=source_relative \
  api/proto/ml.proto

echo "Proto generation complete"