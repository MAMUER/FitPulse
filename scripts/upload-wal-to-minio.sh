#!/usr/bin/env bash
set -euo pipefail

# Upload WAL file to MinIO/S3-compatible storage
# Usage: upload-wal-to-minio.sh <wal_file> <wal_filename>

WAL_FILE="${1:-}"
WAL_FILENAME="${2:-}"

if [[ -z "$WAL_FILE" || -z "$WAL_FILENAME" ]]; then
	echo "ERROR: Usage: $0 <wal_file> <wal_filename>"
	exit 1
fi

if [[ ! -f "$WAL_FILE" ]]; then
	echo "ERROR: WAL file not found: $WAL_FILE"
	exit 1
fi

S3_ENDPOINT="${S3_ENDPOINT:-http://minio.minio.svc.cluster.local:9000}"
S3_BUCKET="${S3_BUCKET:-postgres-wal-archive}"
S3_PATH="${S3_PATH:-wal}"
AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-${MINIO_ROOT_USER:-}}"
AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-${MINIO_ROOT_PASSWORD:-}}"

if [[ -z "$AWS_ACCESS_KEY_ID" || -z "$AWS_SECRET_ACCESS_KEY" ]]; then
	echo "ERROR: AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set"
	exit 1
fi

export AWS_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY

S3_KEY="${S3_PATH}/${WAL_FILENAME}"

echo "Uploading WAL file to S3: ${S3_BUCKET}/${S3_KEY}"

if aws --endpoint-url "$S3_ENDPOINT" s3 cp "$WAL_FILE" "s3://${S3_BUCKET}/${S3_KEY}" --only-show-errors; then
	echo "WAL file uploaded successfully: ${S3_KEY}"
	exit 0
else
	echo "ERROR: Failed to upload WAL file to S3"
	exit 1
fi
