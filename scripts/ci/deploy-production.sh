#!/usr/bin/env bash
set -euo pipefail

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
	wait_for_rabbitmq
	verify_postgres
	run_migrations
	run_seed_admin
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
		images) import_problem_images ;;
		manifests) apply_manifests ;;
		migrations) run_migrations ;;
		verify) verify_deployment ;;
		*)
			log_error "Unknown command: $cmd"
			exit 1
			;;
	esac
}

main "$@"
