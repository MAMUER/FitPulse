#!/usr/bin/env bash
set -euo pipefail

# DVC Setup Script for FitPulse
# Initializes DVC and creates remote storage

echo "=== FitPulse DVC Setup ==="

# Check if DVC is installed
if ! command -v dvc &> /dev/null; then
    echo "DVC is not installed. Install with: pip install dvc"
    exit 1
fi

# Initialize DVC if not already initialized
if [ ! -d ".dvc" ]; then
    echo "Initializing DVC..."
    dvc init
    git add .dvc .dvcignore
    git commit -m "chore: initialize DVC"
fi

# Configure local remote (for development)
# For production, use: dvc remote add -d production s3://fitpulse-ml-models
echo "Configuring local remote..."
dvc remote default local_remote || true
dvc remote modify local_remote url ./dvc_remote || true

# Create remote directory
mkdir -p ./dvc_remote

# Create placeholder directories for local development
# Actual datasets are stored locally and NOT committed to git
mkdir -p datasets/raw
mkdir -p datasets/processed
mkdir -p models
mkdir -p metrics

# Ensure raw datasets are ignored by git
if ! grep -q "^datasets/raw/$" .gitignore 2>/dev/null; then
    echo "" >> .gitignore
    echo "# DVC / ML" >> .gitignore
    echo "datasets/raw/" >> .gitignore
fi

# Git add pipeline files and gitignore
git add dvc.yaml .dvc .gitignore || true
git commit -m "feat: initialize DVC pipeline" || true

echo ""
echo "=== DVC Setup Complete ==="
echo ""
echo "Next steps:"
echo "1. Place raw datasets into datasets/raw/ (this directory is gitignored)"
echo "2. Run pipeline: dvc repro"
echo "3. Push to remote: dvc push"
echo "4. Pull from remote: dvc pull"
echo ""
echo "For production, configure S3-compatible remote:"
echo "  dvc remote add -d production s3://fitpulse-ml-models"
echo "  dvc remote modify production endpointurl http://minio:9000"
echo "  dvc remote modify production access_key_id <key>"
echo "  dvc remote modify production secret_access_key <secret>"
echo ""
echo "IMPORTANT: Do NOT commit DVC remote credentials to git."
