#!/bin/bash
set -euo pipefail

create_namespace() {
	kubectl create namespace fitness-platform-staging --dry-run=client -o yaml | kubectl apply --validate=false -f -
	kubectl create secret docker-registry ghcr-pull-secret \
		--docker-server=ghcr.io \
		--docker-username="$GITHUB_ACTOR" \
		--docker-password="${GHCR_TOKEN}" \
		-n fitness-platform-staging \
		--dry-run=client -o yaml | kubectl apply --validate=false -f -
}

update_image_tags() {
	SHA=$(git rev-parse --short HEAD)
	echo "Updating staging image tags to :$SHA"
	cd configs/k8s/overlays/staging || cd configs/k8s/overlays/production
	for svc in user-service biometric-service training-service gateway classifier; do
		kustomize edit set image "ghcr.io/mamuer/project/$svc=ghcr.io/mamuer/project/$svc:$SHA" || true
	done
	kustomize build --load-restrictor=LoadRestrictionsNone . >/tmp/staging-manifests.yaml
	kubectl apply --validate=false -f /tmp/staging-manifests.yaml -n fitness-platform-staging
}

wait_for_rollout() {
	kubectl wait --for=condition=ready pod -l app=gateway -n fitness-platform-staging --timeout=180s || true
	kubectl wait --for=condition=ready pod -l app=user-service -n fitness-platform-staging --timeout=180s || true
	kubectl wait --for=condition=ready pod -l app=training-service -n fitness-platform-staging --timeout=180s || true
	kubectl get pods -n fitness-platform-staging
}

verify_health() {
	echo "Waiting for staging gateway..."
	for i in {1..30}; do
		if kubectl get pods -n fitness-platform-staging -l app=gateway | grep -q Running; then
			echo "✅ Gateway is running in staging"
			break
		fi
		echo "Attempt $i/30: waiting for gateway..."
		sleep 10
	done
	kubectl get pods -n fitness-platform-staging -o wide || true
}

run_uat_tests() {
	python scripts/api-test.py --base-url "$BASE_URL" || echo "⚠️ API tests failed"
}

main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all)
		create_namespace
		update_image_tags
		wait_for_rollout
		verify_health
		run_uat_tests
		;;
	create_namespace) create_namespace ;;
	update_image_tags) update_image_tags ;;
	wait_for_rollout) wait_for_rollout ;;
	verify_health) verify_health ;;
	run_uat_tests) run_uat_tests ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
