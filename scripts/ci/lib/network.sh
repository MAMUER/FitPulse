#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/utils.sh"

ensure_service_account() {
	echo "Checking service account app-service-account..."
	kubectl create serviceaccount app-service-account \
		-n fitness-platform-production \
		--dry-run=client -o yaml | kubectl apply -f -
	kubectl create role app-service-account-role \
		-n fitness-platform-production \
		--verb=get,list \
		--resource=configmaps,secrets \
		--dry-run=client -o yaml | kubectl apply -f - || true
	kubectl create rolebinding app-service-account-binding \
		--role=app-service-account-role \
		--serviceaccount=fitness-platform-production:app-service-account \
		-n fitness-platform-production \
		--dry-run=client -o yaml | kubectl apply -f - || true
	echo "ServiceAccount ready"
}

create_image_pull_secret() {
	echo "Creating imagePullSecret for ghcr.io..."
	kubectl create namespace fitness-platform-production --dry-run=client -o yaml | kubectl apply --validate=false -f -
	kubectl create secret docker-registry ghcr-pull-secret \
		--docker-server=ghcr.io \
		--docker-username="$GITHUB_ACTOR" \
		--docker-password="${GHCR_TOKEN}" \
		-n fitness-platform-production \
		--dry-run=client -o yaml | kubectl apply --validate=false -f -
	echo "imagePullSecret created/updated"
}

update_image_tags() {
	echo "Updating image tags to :latest"
	cd configs/k8s/overlays/production
	for svc in user-service biometric-service training-service gateway classifier ml-generator open-wearables-backend open-wearables-frontend device-aggregator data-processor; do
		kustomize edit set image "ghcr.io/mamuer/project/$svc=ghcr.io/mamuer/project/$svc:latest" || true
	done
}

cleanup_disk() {
	echo "-> Skipping disk cleanup in CI (handled by VPS cron)"
}
