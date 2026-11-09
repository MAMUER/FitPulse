#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/utils.sh"

create_secrets() {
	echo "Creating gRPC mTLS certificates via cert-manager..."
	kubectl apply -f configs/k8s/base/cert-manager/grpc-selfsigned-ca.yaml
	kubectl apply -f configs/k8s/base/deployments/grpc-server-cert.yaml -n fitness-platform-production
	kubectl apply -f configs/k8s/base/cert-manager/rabbitmq-cert.yaml
	echo "Waiting for gRPC certificates to be ready..."
	kubectl wait --for=condition=ready certificate/grpc-server-cert -n fitness-platform-production --timeout=300s || true
	kubectl wait --for=condition=ready certificate/rabbitmq-cert -n fitness-platform-production --timeout=300s || true
	echo "✅ gRPC mTLS certificates managed by cert-manager"

	echo "Creating argocd-secret..."
	if [[ -n "${ARGOCD_ADMIN_PASSWORD:-}" ]]; then
		ARGOCD_BCRYPT=$(python3 -c "import bcrypt; print(bcrypt.hashpw('${ARGOCD_ADMIN_PASSWORD}'.encode(), bcrypt.gensalt(rounds=12)).decode())" 2>/dev/null || echo "")
		if [[ -z "$ARGOCD_BCRYPT" ]]; then
			ARGOCD_BCRYPT=$(node -e "console.log(require('bcrypt').hashSync('${ARGOCD_ADMIN_PASSWORD}', 12))" 2>/dev/null || echo "")
		fi
		if [[ -n "$ARGOCD_BCRYPT" ]]; then
			kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f - || true
			kubectl create secret generic argocd-secret \
				-n argocd \
				--from-literal=admin.password="$ARGOCD_BCRYPT" \
				--from-literal=admin.passwordMTime="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
				--dry-run=client -o yaml | kubectl apply --validate=false -f -
			echo "✅ argocd-secret created/updated"
		else
			: # bcrypt not available, skipping silently
		fi
	else
		: # ARGOCD_ADMIN_PASSWORD empty, skipping silently
	fi

	echo "Creating external-secrets-db-credentials secret..."
	if [[ -n "${EXTERNAL_SECRETS_DB_PASSWORD:-}" ]]; then
		kubectl create namespace external-secrets --dry-run=client -o yaml | kubectl apply -f - || true
		kubectl create secret generic external-secrets-db-credentials \
			-n external-secrets \
			--from-literal=username=external_secrets \
			--from-literal=password="${EXTERNAL_SECRETS_DB_PASSWORD}" \
			--dry-run=client -o yaml | kubectl apply --validate=false -f - || echo "::warning::external-secrets-db-credentials creation failed" >&2
	fi

	echo "Creating app-secrets (direct fallback if ExternalSecrets missing)..."
	kubectl create namespace fitness-platform-production --dry-run=client -o yaml | kubectl apply -f - || true
	if ! kubectl get secret app-secrets -n fitness-platform-production >/dev/null 2>&1; then
		JWT_PRIV="${JWT_PRIVATE_KEY_PEM:-$(openssl genrsa 2048 2>/dev/null | openssl pkcs8 -topk8 -nocrypt 2>/dev/null || openssl genrsa 2048 2>/dev/null)}"
		JWT_PUB="${JWT_PUBLIC_KEY_PEM:-$(echo "$JWT_PRIV" | openssl rsa -pubout 2>/dev/null || true)}"
		POSTGRES_USER_VAL="${POSTGRES_USER:-fitpulse}"
		POSTGRES_PASS_VAL="${POSTGRES_PASSWORD:-fitpulse-$(date +%s)}"
		POSTGRES_DB_VAL="${POSTGRES_DB:-fitpulse}"
		RABBITMQ_URL_VAL="${RABBITMQ_URL:-amqp://rabbitmq:5672}"
		VALKEY_PASS_VAL="${VALKEY_PASSWORD:-valkey-$(date +%s)}"
		APP_BASE_URL_VAL="${APP_BASE_URL:-https://fittpulse.duckdns.org}"

		kubectl create secret generic app-secrets -n fitness-platform-production \
			--from-literal=JWT_PRIVATE_KEY_PEM="$JWT_PRIV" \
			--from-literal=JWT_PUBLIC_KEY_PEM="$JWT_PUB" \
			--from-literal=POSTGRES_USER="$POSTGRES_USER_VAL" \
			--from-literal=POSTGRES_PASSWORD="$POSTGRES_PASS_VAL" \
			--from-literal=POSTGRES_DB="$POSTGRES_DB_VAL" \
			--from-literal=RABBITMQ_URL="$RABBITMQ_URL_VAL" \
			--from-literal=VALKEY_PASSWORD="$VALKEY_PASS_VAL" \
			--from-literal=APP_BASE_URL="$APP_BASE_URL_VAL" \
			--from-literal=SMTP_HOST="${SMTP_HOST:-smtp.example.com}" \
			--from-literal=SMTP_PORT="${SMTP_PORT:-587}" \
			--from-literal=SMTP_USER="${SMTP_USER:-}" \
			--from-literal=SMTP_PASSWORD="${SMTP_PASSWORD:-}" \
			--from-literal=SMTP_FROM="${SMTP_FROM:-noreply@fittpulse.duckdns.org}" \
			--from-literal=YANDEX_CLIENT_ID="${YANDEX_CLIENT_ID:-}" \
			--from-literal=YANDEX_CLIENT_SECRET="${YANDEX_CLIENT_SECRET:-}" \
			--from-literal=YANDEX_REDIRECT_URL="${YANDEX_REDIRECT_URL:-}" \
			--from-literal=TOTP_ENCRYPTION_KEY="${TOTP_ENCRYPTION_KEY:-$(openssl rand -base64 32 2>/dev/null || echo 'changeme')}" \
			--from-literal=OPEN_WEARABLES_WEBHOOK_SECRET="${OPEN_WEARABLES_WEBHOOK_SECRET:-}" \
			--dry-run=client -o yaml | kubectl apply -f -
		echo "✅ app-secrets created (fallback)"
	else
		echo "✅ app-secrets already exists"
	fi
	# Ensure JWT keys exist in app-secrets even if the secret was created earlier
	# without them (prevents CrashLoopBackOff in gateway/user-service).
	if ! kubectl get secret app-secrets -n fitness-platform-production -o jsonpath='{.data.JWT_PRIVATE_KEY_PEM}' 2>/dev/null | grep -q .; then
		JWT_PRIV="${JWT_PRIVATE_KEY_PEM:-$(openssl genrsa 2048 2>/dev/null | openssl pkcs8 -topk8 -nocrypt 2>/dev/null || openssl genrsa 2048 2>/dev/null)}"
		JWT_PUB="${JWT_PUBLIC_KEY_PEM:-$(echo "$JWT_PRIV" | openssl rsa -pubout 2>/dev/null || true)}"
		kubectl patch secret app-secrets -n fitness-platform-production --type merge -p "{\"stringData\":{\"JWT_PRIVATE_KEY_PEM\":\"$JWT_PRIV\",\"JWT_PUBLIC_KEY_PEM\":\"$JWT_PUB\"}}" >/dev/null 2>&1 || true
	fi

	# Ensure POSTGRES_USER/PASSWORD/DB are present (fix empty/missing keys in pre-existing secrets)
	kubectl patch secret app-secrets -n fitness-platform-production --type=merge -p "{\"stringData\":{\"POSTGRES_USER\":\"${POSTGRES_USER:-fitness_admin}\",\"POSTGRES_PASSWORD\":\"${POSTGRES_PASSWORD:-}\",\"POSTGRES_DB\":\"${POSTGRES_DB:-fitness}\"}}" >/dev/null 2>&1 || true

	echo "Ensuring app-secrets contains OPEN_WEARABLES_WEBHOOK_SECRET..."
	kubectl patch secret app-secrets -n fitness-platform-production --type=merge -p "{\"stringData\":{\"OPEN_WEARABLES_WEBHOOK_SECRET\":\"${OPEN_WEARABLES_WEBHOOK_SECRET:-}\"}}" >/dev/null 2>&1 || true

	# Ensure SEED_ADMIN_EMAIL and SEED_ADMIN_PASSWORD exist in app-secrets (required by seed-admin job)
	kubectl patch secret app-secrets -n fitness-platform-production --type=merge -p "{\"stringData\":{\"SEED_ADMIN_EMAIL\":\"${SEED_ADMIN_EMAIL:-admin@fittpulse.duckdns.org}\",\"SEED_ADMIN_PASSWORD\":\"${SEED_ADMIN_PASSWORD:-FitPulseAdmin2026!}\"}}" >/dev/null 2>&1 || true

	echo "Creating per-service DB credentials (fallback)..."
	for svc in user-service training-service biometric-service data-processor; do
		sec="${svc}-db-credentials"
		kubectl create secret generic "$sec" -n fitness-platform-production \
			--from-literal=POSTGRES_USER="${POSTGRES_USER:-fitness_admin}" \
			--from-literal=POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}" \
			--from-literal=POSTGRES_DB="${POSTGRES_DB:-fitness}" \
			--dry-run=client -o yaml | kubectl apply -f - || echo "::warning::$sec creation failed" >&2
	done

	echo "Creating db-migrations ConfigMap..."
	if [[ -f "db/migrations/V1__full_schema.sql" ]]; then
		kubectl create configmap db-migrations -n fitness-platform-production \
			--from-file=V1__full_schema.sql=db/migrations/V1__full_schema.sql \
			--dry-run=client -o yaml | kubectl apply -f - || echo "::warning::db-migrations creation failed" >&2
	else
		echo "::warning::db/migrations/V1__full_schema.sql not found"
	fi

	kubectl apply -f configs/k8s/base/secrets/rabbitmq-secret.yaml
	echo "✅ rabbitmq-secret created"

	echo "Creating minio-credentials secret..."
	if [[ -n "${MINIO_SECRET_KEY:-}" ]]; then
		kubectl create namespace minio --dry-run=client -o yaml | kubectl apply -f - || true
		kubectl create secret generic minio-credentials \
			-n minio \
			--from-literal=root-user=fitpulse-admin \
			--from-literal=root-password="${MINIO_SECRET_KEY}" \
			--dry-run=client -o yaml | kubectl apply --validate=false -f - || echo "::warning::minio-credentials creation failed" >&2
	fi

	echo "Creating grafana-credentials secret..."
	if [[ -n "${GRAFANA_ADMIN_PASSWORD:-}" ]]; then
		kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f - || true
		kubectl create secret generic grafana-credentials \
			-n monitoring \
			--from-literal=admin-password="${GRAFANA_ADMIN_PASSWORD}" \
			--dry-run=client -o yaml | kubectl apply --validate=false -f - || echo "::warning::grafana-credentials creation failed" >&2
	fi

	echo "Creating oncall-bot-credentials secret..."
	if [[ -n "${TELEGRAM_BOT_TOKEN:-}" ]] && [[ -n "${TELEGRAM_CHAT_ID:-}" ]]; then
		kubectl create secret generic oncall-bot-credentials \
			-n monitoring \
			--from-literal=telegram-bot-token="${TELEGRAM_BOT_TOKEN}" \
			--from-literal=telegram-chat-id="${TELEGRAM_CHAT_ID}" \
			--dry-run=client -o yaml | kubectl apply --validate=false -f - || echo "::warning::oncall-bot-credentials creation failed" >&2
	fi

	echo "Creating elk-credentials secret..."
	if [[ -n "${ELASTICSEARCH_PASSWORD:-}" ]] && [[ -n "${ELASTICSEARCH_PLATFORM_PASSWORD:-}" ]] && [[ -n "${ELASTICSEARCH_AUDITOR_PASSWORD:-}" ]]; then
		kubectl create namespace elk --dry-run=client -o yaml | kubectl apply -f - || true
		kubectl create secret generic elk-credentials \
			-n elk \
			--from-literal=elasticsearch-password="${ELASTICSEARCH_PASSWORD}" \
			--from-literal=platform-team-password="${ELASTICSEARCH_PLATFORM_PASSWORD}" \
			--from-literal=auditor-password="${ELASTICSEARCH_AUDITOR_PASSWORD}" \
			--dry-run=client -o yaml | kubectl apply --validate=false -f - || echo "::warning::elk-credentials creation failed" >&2
	fi

	echo "Creating monitoring-secrets..."
	kubectl create secret generic monitoring-secrets -n fitness-platform-production \
		--from-literal=grafana-admin-password="${GRAFANA_ADMIN_PASSWORD}" \
		--dry-run=client -o yaml | kubectl apply --validate=false -f -
	echo "✅ monitoring-secrets created"

	if [[ -n "${CLOUDFLARE_TURNSTILE_SECRET_KEY:-}" ]]; then
		echo "Creating captcha-secrets..."
		kubectl create secret generic captcha-secrets \
			--namespace=fitness-platform-production \
			--from-literal=CLOUDFLARE_TURNSTILE_SECRET_KEY="$CLOUDFLARE_TURNSTILE_SECRET_KEY" \
			--dry-run=client -o yaml | kubectl apply --validate=false -f -
		echo "✅ captcha-secrets created"
	fi
}
