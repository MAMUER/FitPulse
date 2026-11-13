#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/utils.sh"

PROBLEM_IMAGES=(
	"registry.k8s.io/ingress-nginx/controller:v1.11.3"
	"docker.io/pgsty/silo:RELEASE.2026-11-12T00-00-00Z"
	"docker.io/library/postgres:18-alpine"
	"docker.io/library/rabbitmq:4-management-alpine"
	"docker.io/valkey/valkey:9-alpine"
	"docker.io/grafana/grafana:11.2.0"
	"docker.io/fluent/fluent-bit:3.0.0"
	"docker.elastic.co/elasticsearch/elasticsearch:8.12.0"
	"docker.elastic.co/kibana/kibana:8.12.0"
	"docker.elastic.co/logstash/logstash:8.12.0"
	"docker.io/otel/opentelemetry-collector:0.91.0"
	"docker.io/rancher/mirrored-pause:3.10.2"
	"docker.io/library/busybox:1.37.0"
	"docker.m.daocloud.io/library/postgres:18-alpine"
	"docker.m.daocloud.io/library/rabbitmq:4-management-alpine"
	"ghcr.io/mamuer/project/postgres-pgsodium:pg18"
)

import_fluent_bit() {
	local img="docker.io/fluent/fluent-bit:3.0.0"
	local tar_file="/tmp/fluent-bit.tar"
	if bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo k3s ctr -n k8s.io images ls -q | grep -q '^${img}$'"; then
		log_info "Image already present on VPS: $img"
		return 0
	fi
	log_info "-> Pulling $img on runner..."
	docker pull "$img"
	docker save "$img" -o "$tar_file"
	bash scripts/ssh-retry.sh scp "$tar_file" "${VPS_USER}@${VPS_HOST}:/tmp/"
	bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo k3s ctr -n k8s.io images import ${tar_file} && sudo rm -f ${tar_file}"
	log_info "Imported $img"
}

import_problem_images() {
	log_info "=== Importing problem images manually via runner ==="
	# Authenticate to Docker Hub to avoid anonymous pull rate limits.
	# DOCKER_USERNAME/DOCKER_PASSWORD are optional; if unset, fall back to anon.
	if [[ -n "${DOCKER_USERNAME:-}" && -n "${DOCKER_PASSWORD:-}" ]]; then
		echo "${DOCKER_PASSWORD}" | docker login docker.io -u "${DOCKER_USERNAME}" --password-stdin >/dev/null 2>&1 ||
			log_warn "Docker Hub login failed, continuing anonymously"
	fi
	for img in "${PROBLEM_IMAGES[@]}"; do
		safe_name=$(echo "$img" | tr '/:' '_')
		tar_file="/tmp/${safe_name}.tar"

		if bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo k3s ctr -n k8s.io images ls -q | grep -q '^${img}$'"; then
			log_info "Image already present on VPS: $img"
			continue
		fi

		log_info "-> [1/4] Pulling $img on runner..."
		if ! docker pull "$img"; then
			log_warn "Failed to pull $img (rate limit or auth) — skipping; VPS will pull directly if needed"
			continue
		fi

		log_info "-> [2/4] Saving image to $tar_file..."
		docker save "$img" -o "$tar_file"

		log_info "-> [3/4] Uploading archive to VPS..."
		bash scripts/ssh-retry.sh scp "$tar_file" "${VPS_USER}@${VPS_HOST}:/tmp/"

		log_info "-> [4/4] Importing image into k3s containerd on VPS..."
		bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo k3s ctr -n k8s.io images import /tmp/${safe_name}.tar && sudo rm -f /tmp/${safe_name}.tar"

		log_info "Imported $img"
	done
	log_info "=== Problem images import completed ==="
}

pre_pull_public_images() {
	# These images are not in the k3s airgap bundle and are not GHCR images, so
	# pull them directly on the VPS (it has working DNS/egress after fix_dns).
	local images=(
		"docker.elastic.co/elasticsearch/elasticsearch:8.12.0"
		"docker.elastic.co/kibana/kibana:8.12.0"
		"docker.elastic.co/logstash/logstash:8.12.0"
		"docker.io/fluent/fluent-bit:3.0.0"
		"docker.io/otel/opentelemetry-collector:0.91.0"
	)
	local img
	for img in "${images[@]}"; do
		if ./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo k3s ctr -n k8s.io images ls -q | grep -q '^${img}$'"; then
			log_info "Image already present on VPS: $img"
			continue
		fi
		log_info "-> Pulling $img directly on VPS..."
		if ./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo k3s ctr -n k8s.io images pull '${img}'"; then
			continue
		fi
		# Direct Docker Hub auth is reset on this VPS. The daocloud mirror is
		# reachable and serves the same images; pull from it and tag the
		# canonical name so the kubelet finds it by the manifest's image.
		local mirror="${img/docker.io\//docker.m.daocloud.io/}"
		if [[ "$mirror" != "$img" ]]; then
			log_warn "Direct pull failed; trying mirror $mirror"
			if ./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo k3s ctr -n k8s.io images pull '${mirror}' && sudo k3s ctr -n k8s.io images tag '${mirror}' '${img}'"; then
				continue
			fi
		fi
		log_warn "Failed to pull $img on VPS; the pod will retry via imagePullPolicy"
	done
	log_info "Public images pre-pull completed"
}

pre_pull_images() {
	local image_tag
	image_tag=$(git rev-parse --short HEAD)
	log_info "-> Pre-pulling application images on VPS..."
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
		for img in \\
			\\\"ghcr.io/mamuer/project/user-service:${image_tag}\\\" \\
			\\\"ghcr.io/mamuer/project/biometric-service:${image_tag}\\\" \\
			\\\"ghcr.io/mamuer/project/training-service:${image_tag}\\\" \\
			\\\"ghcr.io/mamuer/project/gateway:${image_tag}\\\" \\
			\\\"ghcr.io/mamuer/project/classifier:${image_tag}\\\" \\
			\\\"ghcr.io/mamuer/project/device-aggregator:${image_tag}\\\" \\
			\\\"ghcr.io/mamuer/project/ml-generator:${image_tag}\\\" \\
			\\\"ghcr.io/mamuer/project/open-wearables-backend:${image_tag}\\\" \\
			\\\"ghcr.io/mamuer/project/open-wearables-frontend:${image_tag}\\\" \\
			\\\"ghcr.io/mamuer/project/data-processor:${image_tag}\\\" \\
			\\\"ghcr.io/mamuer/project/postgres-pgsodium:pg18\\\" \\
			\\\"ghcr.io/mamuer/project/backup-tools:pg18\\\" \
			\\\"ghcr.io/mamuer/project/oncall-bot:v1\\\"; do

			echo \\\"Pulling \\\$img...\\\"
			sudo k3s ctr -n k8s.io images pull --user \"${GITHUB_ACTOR}:${GHCR_TOKEN}\" \"\\\$img\" >/dev/null 2>&1 || \\
			sudo k3s crictl pull --creds \"${GITHUB_ACTOR}:${GHCR_TOKEN}\" \"\\\$img\" >/dev/null 2>&1 || \\
			true
		done
	"
	log_info "Pre-pull step completed"
}
