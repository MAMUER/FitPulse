#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/utils.sh"

import_problem_images() {
	echo "=== Importing problem images manually via runner ==="
	for img in "${PROBLEM_IMAGES[@]}"; do
		safe_name=$(echo "$img" | tr '/:' '_')
		tar_file="/tmp/${safe_name}.tar"

		if bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo k3s ctr -n k8s.io images ls -q | grep -q '^${img}$'"; then
			echo "Image already present on VPS: $img"
			continue
		fi

		echo "-> [1/4] Pulling $img on runner..."
		docker pull "$img" || {
			echo "Failed to pull $img"
			exit 1
		}

		echo "-> [2/4] Saving image to $tar_file..."
		docker save "$img" -o "$tar_file"

		echo "-> [3/4] Uploading archive to VPS..."
		bash scripts/ssh-retry.sh scp "$tar_file" "${VPS_USER}@${VPS_HOST}:/tmp/"

		echo "-> [4/4] Importing image into k3s containerd on VPS..."
		bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo k3s ctr -n k8s.io images import /tmp/${safe_name}.tar && sudo rm -f /tmp/${safe_name}.tar"

		echo "Imported $img"
	done
	echo "=== Problem images import completed ==="
}



pre_pull_images() {
	echo "-> Pre-pulling application images on VPS..."
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
		for img in \
			\"ghcr.io/mamuer/project/user-service:latest\" \
			\"ghcr.io/mamuer/project/biometric-service:latest\" \
			\"ghcr.io/mamuer/project/training-service:latest\" \
			\"ghcr.io/mamuer/project/gateway:latest\" \
			\"ghcr.io/mamuer/project/classifier:latest\" \
			\"ghcr.io/mamuer/project/device-aggregator:latest\" \
			\"ghcr.io/mamuer/project/ml-generator:latest\" \
			\"ghcr.io/mamuer/project/open-wearables-backend:latest\" \
			\"ghcr.io/mamuer/project/open-wearables-frontend:latest\" \
			\"ghcr.io/mamuer/project/data-processor:latest\"; do

			echo \"Pulling \$img...\"
			sudo k3s ctr images pull --user \"${GITHUB_ACTOR}:${GHCR_TOKEN}\" \"\$img\" >/dev/null 2>&1 || \
			sudo k3s crictl pull --creds \"${GITHUB_ACTOR}:${GHCR_TOKEN}\" \"\$img\" >/dev/null 2>&1 || \
			true
		done
	"
	echo "✅ Pre-pull step completed"
}



