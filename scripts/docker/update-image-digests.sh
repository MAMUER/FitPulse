#!/bin/bash
set -euo pipefail

# update-image-digests.sh
# Updates base images in all Dockerfiles to use digest pinning for reproducibility
# Usage: ./scripts/docker/update-image-digests.sh

echo "=== Updating Docker image digests ==="

# Define images and their digests (update these regularly)
declare -A IMAGE_DIGESTS
IMAGE_DIGESTS["node:26-alpine"]="node:26-alpine@sha256:REPLACE_NODE_DIGEST"
IMAGE_DIGESTS["golang:1.26-alpine"]="golang:1.26-alpine@sha256:REPLACE_GOLANG_DIGEST"
IMAGE_DIGESTS["alpine:3.24"]="alpine:3.24@sha256:REPLACE_ALPINE_DIGEST"
IMAGE_DIGESTS["python:3.14-slim"]="python:3.14-slim@sha256:REPLACE_PYTHON_DIGEST"

# Function to get digest for an image
get_digest() {
  local image=$1
  echo "Fetching digest for $image..."
  local digest
  digest=$(docker manifest inspect "$image" | jq -r '.manifests[].digest' | head -1)
  if [ -z "$digest" ] || [ "$digest" == "null" ]; then
    echo "WARNING: Could not fetch digest for $image"
    return 1
  fi
  echo "$digest"
}

# Update Dockerfiles
find cmd -name "Dockerfile" -type f | while read -r dockerfile; do
  echo "Processing $dockerfile..."
  
  for image in "${!IMAGE_DIGESTS[@]}"; do
    if grep -q "^FROM $image" "$dockerfile"; then
      digest=$(get_digest "$image")
      if [ -n "$digest" ]; then
        sed -i "s|^FROM $image|FROM ${image%@*}@${digest#*sha256:}|" "$dockerfile"
        echo "  Updated $image to use digest"
      fi
    fi
  done
done

echo "=== Digest update complete ==="
echo "Please review changes and commit if satisfied."
