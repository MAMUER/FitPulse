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

pull_image_with_mirror_fallback() {
	local img="$1"
	local mirror="${img/docker.io\//docker.m.daocloud.io/}"

	# Prefer the canonical Docker Hub reference, but fall back to the Daocloud
	# mirror when Docker Hub is rate-limited or the anonymous token fetch fails.
	if docker pull "$img" >/dev/null 2>&1; then
		return 0
	fi

	if [[ "$mirror" != "$img" ]]; then
		log_warn "Direct pull failed for $img; trying mirror $mirror"
		if docker pull "$mirror" >/dev/null 2>&1; then
			docker tag "$mirror" "$img" >/dev/null 2>&1 || return 1
			return 0
		fi
	fi

	return 1
}

import_fluent_bit() {
	local img="docker.io/fluent/fluent-bit:3.0.0"
	local tar_file="/tmp/fluent-bit.tar"
	if bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo k3s ctr -n k8s.io images ls -q | grep -q '^${img}$'"; then
		log_info "Image already present on VPS: $img"
		return 0
	fi

	if [[ -n "${DOCKER_USERNAME:-}" && -n "${DOCKER_PASSWORD:-}" ]]; then
		echo "${DOCKER_PASSWORD}" | docker login docker.io -u "${DOCKER_USERNAME}" --password-stdin >/dev/null 2>&1 ||
			log_warn "Docker Hub login failed, continuing anonymously"
	fi

	log_info "-> Pulling $img on runner..."
	if ! pull_image_with_mirror_fallback "$img"; then
		log_warn "Failed to pull $img on runner; kubelet will retry the image when the pod starts"
		return 0
	fi

	docker save "$img" -o "$tar_file"
	bash scripts/ssh-retry.sh scp "$tar_file" "${VPS_USER}@${VPS_HOST}:/tmp/"
	bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo k3s ctr -n k8s.io images import ${tar_file} && sudo rm -f ${tar_file}"
	log_info "Imported $img"
}

import_duckdns_webhook_image() {
	local img="docker.io/ebrianne/cert-manager-webhook-duckdns:v1.2.3"
	local pinned_img="docker.io/ebrianne/cert-manager-webhook-duckdns@sha256:b5fdc5eaf273ba5f7c461d9baca3703b1d13e67d5d44f23139afbf1cc53e9a00"
	local tar_file="${RUNNER_TEMP:-/tmp}/duckdns-webhook-image.tar"
	local image_id pull_output
	local remote_images

	log_info "Checking whether $img is already present in k3s containerd..."
	remote_images=$(bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" \
		"sudo k3s ctr -n k8s.io images ls -q")
	# Here-string instead of `printf | grep`: grep exits on the first match and,
	# under `set -o pipefail`, printf then dies with SIGPIPE ("Broken pipe" noise).
	if grep -Fxq "$img" <<<"$remote_images"; then
		log_info "Image already present on VPS: $img"
		return 0
	fi

	log_info "Pulling pinned DuckDNS webhook image on the runner: $pinned_img (download progress hidden)"
	if ! pull_output=$(docker pull --platform linux/amd64 "$pinned_img" 2>&1); then
		echo "::error::Failed to pull DuckDNS webhook image; last 40 lines:" >&2
		printf '%s\n' "$pull_output" | tail -n 40 >&2
		return 1
	fi
	image_id=$(docker image inspect --format='{{.Id}}' "$pinned_img")
	log_info "Pulled image $image_id ($(docker image inspect --format='{{.Os}}/{{.Architecture}}' "$pinned_img"))"
	docker tag "$image_id" "$img"
	docker save "$img" -o "$tar_file"

	log_info "Uploading DuckDNS webhook image archive to VPS..."
	bash scripts/ssh-retry.sh scp "$tar_file" "${VPS_USER}@${VPS_HOST}:/tmp/duckdns-webhook-image.tar"
	log_info "Importing image into k3s containerd and verifying the tag..."
	bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
		sudo k3s ctr -n k8s.io images import /tmp/duckdns-webhook-image.tar &&
		sudo rm -f /tmp/duckdns-webhook-image.tar &&
		sudo k3s ctr -n k8s.io images ls -q | grep -Fx '${img}'
	"
	rm -f "$tar_file"
	log_info "Imported $img into k3s containerd"
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
		if ! pull_image_with_mirror_fallback "$img"; then
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

pull_remote_ctr_image() {
	local img="$1"
	local tag_as="${2:-}"
	local remote_script
	remote_script=$(
		printf 'IMAGE=%q\nTAG_AS=%q\n' "$img" "$tag_as"
		cat <<'REMOTE'
set -euo pipefail

if pull_output=$(sudo k3s ctr -n k8s.io images pull "$IMAGE" 2>&1); then
	echo "Pulled image: $IMAGE"
else
	status=$?
	echo "::error::Failed to pull $IMAGE; last 40 lines:" >&2
	printf '%s\n' "$pull_output" | tail -n 40 >&2
	exit "$status"
fi

if [[ -n "$TAG_AS" ]]; then
	sudo k3s ctr -n k8s.io images tag "$IMAGE" "$TAG_AS"
	echo "Tagged $IMAGE as $TAG_AS"
fi
REMOTE
	)
	if ! printf '%s\n' "$remote_script" |
		./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "bash -s"; then
		return 1
	fi
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
		log_info "-> Pulling $img directly on VPS (download progress hidden)..."
		if pull_remote_ctr_image "$img"; then
			continue
		fi
		# Direct Docker Hub auth is reset on this VPS. The daocloud mirror is
		# reachable and serves the same images; pull from it and tag the
		# canonical name so the kubelet finds it by the manifest's image.
		local mirror="${img/docker.io\//docker.m.daocloud.io/}"
		if [[ "$mirror" != "$img" ]]; then
			log_warn "Direct pull failed; trying mirror $mirror"
			if pull_remote_ctr_image "$mirror" "$img"; then
				continue
			fi
		fi
		log_warn "Failed to pull $img on VPS; the pod will retry via imagePullPolicy"
	done
	log_info "Public images pre-pull completed"
}

# Канонический список образов, которые публикует build-push.sh и запускает кластер.
# Один источник правды для pre_pull_images() (деплой) и скана образов в CI —
# иначе список разъедется и часть образов перестанет проверяться.
deploy_image_refs() {
	local image_tag="${1:-$(git rev-parse --short HEAD)}"
	printf '%s\n' \
		"ghcr.io/mamuer/project/user-service:${image_tag}" \
		"ghcr.io/mamuer/project/biometric-service:${image_tag}" \
		"ghcr.io/mamuer/project/training-service:${image_tag}" \
		"ghcr.io/mamuer/project/gateway:${image_tag}" \
		"ghcr.io/mamuer/project/classifier:${image_tag}" \
		"ghcr.io/mamuer/project/device-aggregator:${image_tag}" \
		"ghcr.io/mamuer/project/ml-generator:${image_tag}" \
		"ghcr.io/mamuer/project/open-wearables-backend:${image_tag}" \
		"ghcr.io/mamuer/project/open-wearables-frontend:${image_tag}" \
		"ghcr.io/mamuer/project/data-processor:${image_tag}" \
		"ghcr.io/mamuer/project/email-worker:${image_tag}" \
		"ghcr.io/mamuer/project/postgres-pgsodium:pg18" \
		"ghcr.io/mamuer/project/backup-tools:pg18" \
		"ghcr.io/mamuer/project/oncall-bot:${image_tag}"
}

pre_pull_images() {
	local image_tag remote_script
	image_tag=$(git rev-parse --short HEAD)
	if [[ -z "${GITHUB_ACTOR:-}" || -z "${GHCR_TOKEN:-}" ]]; then
		echo "::error::GITHUB_ACTOR and GHCR_TOKEN are required to pre-pull private GHCR images" >&2
		return 1
	fi

	log_info "-> Pre-pulling application images on VPS..."
	remote_script=$(
		printf 'GITHUB_ACTOR=%q\nGHCR_TOKEN=%q\n' \
			"$GITHUB_ACTOR" "$GHCR_TOKEN"
		# Список образов подставляем на этапе сборки скрипта: так он берётся из
		# deploy_image_refs() и не дублируется внутри heredoc.
		printf 'set -euo pipefail\n\nimages=(\n%s\n)\n' "$(deploy_image_refs "$image_tag")"
		cat <<'REMOTE'

for img in "${images[@]}"; do
	echo "-> Pulling $img (download progress hidden)..."
	if ctr_output=$(sudo k3s ctr -n k8s.io images pull --user "${GITHUB_ACTOR}:${GHCR_TOKEN}" "$img" 2>&1); then
		echo "Pulled $img"
		continue
	fi
	echo "::warning::containerd pull failed for $img; retrying through CRI"
	if cri_output=$(sudo k3s crictl pull --creds "${GITHUB_ACTOR}:${GHCR_TOKEN}" "$img" 2>&1); then
		echo "Pulled $img through CRI"
		continue
	fi
	echo "::error::Failed to pull $img from GHCR; last 30 lines from each method:" >&2
	printf '%s\n' "${ctr_output//"$GHCR_TOKEN"/[REDACTED]}" | tail -n 30 >&2
	printf '%s\n' "${cri_output//"$GHCR_TOKEN"/[REDACTED]}" | tail -n 30 >&2
	exit 1
done
REMOTE
	)
	if ! printf '%s\n' "$remote_script" |
		bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "bash -s"; then
		echo "::error::Failed to pre-pull application images; stopping deployment" >&2
		return 1
	fi
	log_info "Pre-pull step completed"
}
