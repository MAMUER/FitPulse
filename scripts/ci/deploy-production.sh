#!/usr/bin/env bash
set -euo pipefail

# === НАСТРОЙКА SSH known_hosts ===
if [[ -n "${VPS_HOST:-}" ]]; then
	bash scripts/ci/ci-helpers.sh add_vps_to_known_hosts
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/utils.sh"
source "$SCRIPT_DIR/lib/secrets.sh"
source "$SCRIPT_DIR/lib/images.sh"
source "$SCRIPT_DIR/lib/k8s.sh"
source "$SCRIPT_DIR/lib/network.sh"
deploy_all() {
	log_info "Starting production deployment"
	ensure_service_account
	create_image_pull_secret
	create_secrets
	import_problem_images
	pre_pull_images
	setup_local_path_provisioner 2>/dev/null || true
	install_ingress_nginx 2>/dev/null || true
	cleanup_traefik 2>/dev/null || true
	apply_manifests
	restart_rabbitmq_pod
	wait_for_rabbitmq
	verify_postgres
	run_migrations
	run_seed_admin
	restart_app_deployments
	check_logs 2>/dev/null || true
	verify_deployment
	deploy_open_wearables
	log_info "Deployment completed successfully"
}

main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all) deploy_all ;;
	secrets) create_secrets ;;
	create_secrets) create_secrets ;;
	cleanup_disk) cleanup_disk ;;
	images) import_problem_images ;;
	update_image_tags) update_image_tags ;;
	pre_pull_images) pre_pull_images ;;
	manifests) apply_manifests ;;
	apply_manifests) apply_manifests ;;
	cleanup_cluster) cleanup_cluster_resources ;;
	migrations) run_migrations ;;
	run_migrations) run_migrations ;;
	verify) verify_deployment ;;
	verify_deployment) verify_deployment ;;
	ensure_service_account) ensure_service_account ;;
	create_image_pull_secret) create_image_pull_secret ;;
	wait_for_rabbitmq) wait_for_rabbitmq ;;
	verify_postgres) verify_postgres ;;
	run_seed_admin) run_seed_admin ;;
	restart_app_deployments) restart_app_deployments ;;
	check_logs) check_logs ;;
	debug_pod_status) debug_pod_status ;;
	deploy_open_wearables) deploy_open_wearables ;;
	*)
		log_error "Unknown command: $cmd"
		exit 1
		;;
	esac
}

main "$@"
