#!/bin/bash
set -euo pipefail

# Automated first-deploy of the FitPulse application tier to the production VPS.
# Invoked from .github/workflows/ci.yml (job: deploy-vps); every value comes from
# GitHub secrets via the job env — nothing is prompted for.
#
# Safety rules:
#   1. Swap on the VPS is ensured by the provision job (configure_swap).

NAMESPACE="fitness-platform-production"
OVERLAY="configs/k8s/overlays/vps-minimal"
SECRETS_DIR="/etc/fitpulse/secrets"
IMAGE_TAG="${IMAGE_TAG:-$(git rev-parse --short HEAD)}"
CLASSIFIER_TAG="${CLASSIFIER_TAG:-0.0.0-dev}"
# DB identity comes from repository variables (POSTGRES_USER/POSTGRES_DB)
POSTGRES_USER="${POSTGRES_USER:-fitness_admin}"
POSTGRES_DB="${POSTGRES_DB:-fitness}"
NODE_PORT="${NODE_PORT:-30081}"

log() { echo "[deploy-vps]] $*"; }
die() {
	echo "::error::deploy-vps: $*" >&2
	exit 1
}

require_env() {
	local missing=()
	for var in POSTGRES_PASSWORD RABBITMQ_PASS GHCR_TOKEN VPS_HOST VPS_USER; do
		[[ -n "${!var:-}" ]] || missing+=("$var")
	done
	[[ ${#missing[@]} -eq 0 ]] || die "missing required env: ${missing[*]}"
}

ensure_remote_secrets() {
	# Generates long-lived secrets once on the VPS and reuses them on redeploys
	# (JWT keys, TOTP key, self-signed TLS for gRPC/RabbitMQ).
	log "ensuring long-lived secrets on VPS (${SECRETS_DIR})"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo mkdir -p ${SECRETS_DIR} && sudo chmod 700 ${SECRETS_DIR}"

	local need_jwt need_totp need_tls
	need_jwt=$(./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" \
		"sudo test -s ${SECRETS_DIR}/JWT_PRIVATE_KEY_PEM && echo yes || echo no" | tr -d '\r')
	need_totp=$(./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" \
		"sudo test -s ${SECRETS_DIR}/TOTP_ENCRYPTION_KEY && echo yes || echo no" | tr -d '\r')
	need_tls=$(./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" \
		"sudo test -s ${SECRETS_DIR}/tls.crt && echo yes || echo no" | tr -d '\r')

	if [[ "$need_jwt" = "no" ]]; then
		if [[ -n "${JWT_PRIVATE_KEY_PEM:-}" ]] && [[ -n "${JWT_PUBLIC_KEY_PEM:-}" ]]; then
			printf '%s\n' "$JWT_PRIVATE_KEY_PEM" | ./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo tee ${SECRETS_DIR}/JWT_PRIVATE_KEY_PEM >/dev/null"
			printf '%s\n' "$JWT_PUBLIC_KEY_PEM" | ./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo tee ${SECRETS_DIR}/JWT_PUBLIC_KEY_PEM >/dev/null"
			log "JWT keys taken from GitHub secrets"
		else
			./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" \
				"sudo openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out ${SECRETS_DIR}/JWT_PRIVATE_KEY_PEM && sudo openssl pkey -in ${SECRETS_DIR}/JWT_PRIVATE_KEY_PEM -pubout -out ${SECRETS_DIR}/JWT_PUBLIC_KEY_PEM"
			log "JWT keys generated on VPS"
		fi
	fi
	if [[ "$need_totp" = "no" ]]; then
		if [[ -n "${TOTP_ENCRYPTION_KEY:-}" ]]; then
			printf '%s\n' "$TOTP_ENCRYPTION_KEY" | ./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo tee ${SECRETS_DIR}/TOTP_ENCRYPTION_KEY >/dev/null"
		else
			./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" \
				"sudo openssl rand -hex 32 | sudo tee ${SECRETS_DIR}/TOTP_ENCRYPTION_KEY >/dev/null"
		fi
		log "TOTP encryption key stored"
	fi
	if [[ "$need_tls" = "no" ]]; then
		./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" \
			"sudo openssl req -x509 -newkey rsa:2048 -nodes -days 825 -keyout ${SECRETS_DIR}/tls.key -out ${SECRETS_DIR}/tls.crt -subj '/CN=fitpulse.internal/O=FitPulse' -addext 'subjectAltName=DNS:localhost,DNS:*.fitness-platform-production.svc.cluster.local' && sudo cp ${SECRETS_DIR}/tls.crt ${SECRETS_DIR}/ca.crt"
		log "self-signed TLS material generated on VPS"
	fi
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo chmod 600 ${SECRETS_DIR}/*"
}

create_secrets() {
	log "creating/updating Kubernetes secrets in ${NAMESPACE}"
	kubectl get namespace "$NAMESPACE" >/dev/null 2>&1 || kubectl create namespace "$NAMESPACE"

	# Fetch long-lived material from the VPS secret store
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo cat ${SECRETS_DIR}/JWT_PRIVATE_KEY_PEM" >/tmp/jwt_private.pem
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo cat ${SECRETS_DIR}/JWT_PUBLIC_KEY_PEM" >/tmp/jwt_public.pem
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo cat ${SECRETS_DIR}/TOTP_ENCRYPTION_KEY" >/tmp/totp.key
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo cat ${SECRETS_DIR}/tls.crt" >/tmp/tls.crt
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo cat ${SECRETS_DIR}/tls.key" >/tmp/tls.key
	cp /tmp/tls.crt /tmp/ca.crt

	kubectl create secret generic app-secrets -n "$NAMESPACE" \
		--from-literal=POSTGRES_USER="$POSTGRES_USER" \
		--from-literal=POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
		--from-literal=POSTGRES_DB="$POSTGRES_DB" \
		--from-literal=RABBITMQ_URL="amqp://${RABBITMQ_USER:-fitpulse}:${RABBITMQ_PASS}@rabbitmq:5672/" \
		--from-literal=VALKEY_PASSWORD="${VALKEY_PASSWORD:-}" \
		--from-literal=APP_BASE_URL="${APP_BASE_URL:-https://fittpulse.duckdns.org}" \
		--from-literal=SMTP_HOST="${SMTP_HOST:-}" \
		--from-literal=SMTP_PORT="${SMTP_PORT:-}" \
		--from-literal=SMTP_USER="${SMTP_USER:-}" \
		--from-literal=SMTP_PASSWORD="${SMTP_PASSWORD:-}" \
		--from-literal=SMTP_FROM="${SMTP_FROM:-}" \
		--from-literal=YANDEX_CLIENT_ID="${YANDEX_CLIENT_ID:-}" \
		--from-literal=YANDEX_CLIENT_SECRET="${YANDEX_CLIENT_SECRET:-}" \
		--from-literal=YANDEX_REDIRECT_URL="${YANDEX_REDIRECT_URL:-${APP_BASE_URL:-https://fittpulse.duckdns.org}/api/v1/auth/yandex/callback}" \
		--from-literal=OPEN_WEARABLES_WEBHOOK_SECRET="${OPEN_WEARABLES_WEBHOOK_SECRET:-}" \
		--from-literal=SEED_ADMIN_EMAIL="${SEED_ADMIN_EMAIL:-}" \
		--from-literal=SEED_ADMIN_PASSWORD="${SEED_ADMIN_PASSWORD:-}" \
		--from-file=JWT_PRIVATE_KEY_PEM=/tmp/jwt_private.pem \
		--from-file=JWT_PUBLIC_KEY_PEM=/tmp/jwt_public.pem \
		--from-file=TOTP_ENCRYPTION_KEY=/tmp/totp.key \
		--dry-run=client -o yaml | kubectl apply -f -
	rm -f /tmp/jwt_private.pem /tmp/jwt_public.pem /tmp/totp.key

	for svc in user-service biometric-service training-service; do
		kubectl create secret generic "${svc}-db-credentials" -n "$NAMESPACE" \
			--from-literal=POSTGRES_USER="$POSTGRES_USER" \
			--from-literal=POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
			--from-literal=POSTGRES_DB="$POSTGRES_DB" \
			--dry-run=client -o yaml | kubectl apply -f -
	done

	for s in grpc-server-tls rabbitmq-tls; do
		kubectl create secret generic "$s" -n "$NAMESPACE" \
			--from-file=tls.crt=/tmp/tls.crt \
			--from-file=tls.key=/tmp/tls.key \
			--from-file=ca.crt=/tmp/ca.crt \
			--dry-run=client -o yaml | kubectl apply -f -
	done
	rm -f /tmp/tls.crt /tmp/tls.key /tmp/ca.crt
	log "secrets applied"
}

create_pull_secret() {
	log "creating ghcr-pull-secret"
	kubectl create secret docker-registry ghcr-pull-secret -n "$NAMESPACE" \
		--docker-server=ghcr.io \
		--docker-username="${GITHUB_ACTOR:-$(git config user.name)}" \
		--docker-password="${GHCR_TOKEN}" \
		--dry-run=client -o yaml | kubectl apply -f -
}

apply_overlay() {
	log "applying overlay ${OVERLAY} (IMAGE_TAG=${IMAGE_TAG})"
	(
		cd "$OVERLAY"
		for svc in user-service biometric-service training-service gateway device-aggregator; do
			kustomize edit set image "ghcr.io/mamuer/project/${svc}=ghcr.io/mamuer/project/${svc}:${IMAGE_TAG}"
		done
		kustomize edit set image "ghcr.io/mamuer/project/classifier=ghcr.io/mamuer/project/classifier:${CLASSIFIER_TAG:-0.0.0-dev}"
	)
	kubectl apply -k "$OVERLAY"

	for res in deployment/postgres statefulset/rabbitmq deployment/valkey \
		deployment/user-service deployment/biometric-service \
		deployment/training-service deployment/classifier \
		deployment/device-aggregator deployment/gateway; do
		kubectl -n "$NAMESPACE" rollout status "$res" --timeout=300s ||
			log "::warning::$res not ready yet"
	done

	log "waiting for migrate-db"
	kubectl -n "$NAMESPACE" delete job migrate-db --ignore-not-found=true
	kubectl apply -k "$OVERLAY"
	kubectl -n "$NAMESPACE" wait --for=condition=complete job/migrate-db --timeout=300s || {
		kubectl -n "$NAMESPACE" logs job/migrate-db --tail=50 || true
		die "migrate-db failed"
	}

	log "waiting for seed-admin"
	kubectl -n "$NAMESPACE" wait --for=condition=complete job/seed-admin --timeout=300s ||
		log "::warning::seed-admin did not complete (admin may already exist)"
}

verify() {
	kubectl -n "$NAMESPACE" get pods -o wide
	local ip
	ip=$(./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "hostname -I | awk '{print \$1}'" | tr -d '\r')
	log "gateway health: ${ip}:${NODE_PORT}/health"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "curl -fsS --max-time 10 http://localhost:${NODE_PORT}/health" || # NOSONAR S5332
		log "::warning::gateway health check failed"
}

main() {
	require_env
	ensure_remote_secrets
	create_pull_secret
	create_secrets
	apply_overlay
	verify
	log "deploy finished"
}

case "${1:-all}" in
ensure_remote_secrets) ensure_remote_secrets ;;
create_pull_secret) create_pull_secret ;;
create_secrets) create_secrets ;;
apply_overlay) apply_overlay ;;
verify) verify ;;
all) main ;;
*)
	echo "Unknown command: $1" >&2
	echo "Available: ensure_remote_secrets create_pull_secret create_secrets apply_overlay verify all" >&2
	exit 1
	;;
esac
