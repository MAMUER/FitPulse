#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/utils.sh"

# The Go services sign JWTs with ES256 and parse the private key through
# x509.ParseECPrivateKey, which only accepts a SEC1 "EC PRIVATE KEY" block on
# P-256. Legacy deploys generated RSA keys here, so every login failed with
# "failed to generate token". These helpers validate the key shape and rotate
# invalid (RSA) keys instead of silently reusing them.

is_es256_private_key() {
	local pem="${1:-}"
	[[ -n "$pem" ]] || return 1
	printf '%s\n' "$pem" | grep -q -- '-----BEGIN EC PRIVATE KEY-----' || return 1
	printf '%s\n' "$pem" | openssl ec -noout >/dev/null 2>&1 || return 1
	printf '%s\n' "$pem" | openssl ec -noout -text 2>/dev/null | grep -q 'prime256v1'
}

is_es256_public_key() {
	local pem="${1:-}"
	[[ -n "$pem" ]] || return 1
	printf '%s\n' "$pem" | grep -q -- '-----BEGIN PUBLIC KEY-----' || return 1
	printf '%s\n' "$pem" | openssl ec -pubin -noout >/dev/null 2>&1 || return 1
	printf '%s\n' "$pem" | openssl ec -pubin -noout -text 2>/dev/null | grep -q 'prime256v1'
}

generate_es256_private_key() {
	openssl ecparam -name prime256v1 -genkey -noout 2>/dev/null
}

es256_public_from_private() {
	printf '%s\n' "${1:-}" | openssl ec -pubout 2>/dev/null
}

JWT_KEY_PRIVATE=""
JWT_KEY_PUBLIC=""
# Prefer the keys already stored in the cluster (keeps sessions valid), then a
# valid ES256 pair from GitHub secrets, and finally generate a new ES256 pair.
resolve_es256_keypair() {
	local current_private="${1:-}"
	local current_public="${2:-}"

	if is_es256_private_key "$current_private"; then
		JWT_KEY_PRIVATE="$current_private"
		if is_es256_public_key "$current_public"; then
			JWT_KEY_PUBLIC="$current_public"
		else
			JWT_KEY_PUBLIC="$(es256_public_from_private "$current_private")"
		fi
		return 0
	fi
	if is_es256_private_key "${JWT_PRIVATE_KEY_PEM:-}"; then
		JWT_KEY_PRIVATE="$JWT_PRIVATE_KEY_PEM"
		if is_es256_public_key "${JWT_PUBLIC_KEY_PEM:-}"; then
			JWT_KEY_PUBLIC="$JWT_PUBLIC_KEY_PEM"
		else
			JWT_KEY_PUBLIC="$(es256_public_from_private "$JWT_PRIVATE_KEY_PEM")"
		fi
		return 0
	fi
	if [[ -n "${JWT_PRIVATE_KEY_PEM:-}${JWT_PUBLIC_KEY_PEM:-}${current_private}${current_public}" ]]; then
		echo "::warning::JWT keys are not ES256/P-256; generating a new ES256 keypair" >&2
	fi
	JWT_KEY_PRIVATE="$(generate_es256_private_key)"
	JWT_KEY_PUBLIC="$(es256_public_from_private "$JWT_KEY_PRIVATE")"
	if [[ -z "$JWT_KEY_PRIVATE" ]] || [[ -z "$JWT_KEY_PUBLIC" ]]; then
		echo "::error::failed to generate ES256 JWT keypair" >&2
		return 1
	fi
}

# JSON-encode multiline PEMs before patching; interpolating them into a JSON
# literal directly is invalid and made the old patch silently fail.
patch_app_secret_jwt_keys() {
	local patch_file
	patch_file="$(mktemp)"
	JWT_PRIVATE_PATCH="$1" JWT_PUBLIC_PATCH="$2" python3 - "$patch_file" <<'PY'
import json
import os
import sys

with open(sys.argv[1], "w", encoding="utf-8") as fh:
    json.dump(
        {
            "stringData": {
                "JWT_PRIVATE_KEY_PEM": os.environ["JWT_PRIVATE_PATCH"],
                "JWT_PUBLIC_KEY_PEM": os.environ["JWT_PUBLIC_PATCH"],
            }
        },
        fh,
    )
PY
	kubectl patch secret app-secrets -n fitness-platform-production \
		--type=merge -p "$(cat "$patch_file")" >/dev/null
	rm -f "$patch_file"
}

read_stored_jwt_secret() {
	local field="$1"
	kubectl get secret app-secrets -n fitness-platform-production \
		-o jsonpath="{.data.${field}}" 2>/dev/null | base64 -d 2>/dev/null || true
}

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
	existing_jwt_private="$(read_stored_jwt_secret JWT_PRIVATE_KEY_PEM)"
	existing_jwt_public="$(read_stored_jwt_secret JWT_PUBLIC_KEY_PEM)"
	resolve_es256_keypair "$existing_jwt_private" "$existing_jwt_public"
	if ! kubectl get secret app-secrets -n fitness-platform-production >/dev/null 2>&1; then
		POSTGRES_USER_VAL="${POSTGRES_USER:-fitpulse}"
		POSTGRES_PASS_VAL="${POSTGRES_PASSWORD:-fitpulse-$(date +%s)}"
		POSTGRES_DB_VAL="${POSTGRES_DB:-fitpulse}"
		RABBITMQ_USER_VAL="${RABBITMQ_DEFAULT_USER:-fitpulse}"
		RABBITMQ_PASS_VAL="${RABBITMQ_DEFAULT_PASS:-$(openssl rand -base64 32 2>/dev/null || echo 'change-me-rabbitmq')}"
		RABBITMQ_URL_VAL="amqp://${RABBITMQ_USER_VAL}:${RABBITMQ_PASS_VAL}@rabbitmq:5672"
		VALKEY_PASS_VAL="${VALKEY_PASSWORD:-valkey-$(date +%s)}"
		APP_BASE_URL_VAL="${APP_BASE_URL:-https://fitpulse.tail6f5dca.ts.net}"

		kubectl create secret generic app-secrets -n fitness-platform-production \
			--from-literal=JWT_PRIVATE_KEY_PEM="$JWT_KEY_PRIVATE" \
			--from-literal=JWT_PUBLIC_KEY_PEM="$JWT_KEY_PUBLIC" \
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
			--from-literal=SMTP_FROM="${SMTP_FROM:-noreply@fitpulse.tail6f5dca.ts.net}" \
			--from-literal=RABBITMQ_DEFAULT_USER="${RABBITMQ_DEFAULT_USER:-fitpulse}" \
			--from-literal=RABBITMQ_DEFAULT_PASS="${RABBITMQ_DEFAULT_PASS:-$(openssl rand -base64 32 2>/dev/null || echo 'change-me-rabbitmq')}" \
			--from-literal=YANDEX_CLIENT_ID="${YANDEX_CLIENT_ID:-}" \
			--from-literal=YANDEX_CLIENT_SECRET="${YANDEX_CLIENT_SECRET:-}" \
			--from-literal=YANDEX_REDIRECT_URL="${YANDEX_REDIRECT_URL:-}" \
			--from-literal=TOTP_ENCRYPTION_KEY="${TOTP_ENCRYPTION_KEY:-$(openssl rand -base64 32 2>/dev/null || echo 'changeme')}" \
			--from-literal=DB_ENCRYPTION_KEY="${DB_ENCRYPTION_KEY:-$(openssl rand -hex 32 2>/dev/null || echo 'dev-encryption-key-change-me')}" \
			--from-literal=OPEN_WEARABLES_WEBHOOK_SECRET="${OPEN_WEARABLES_WEBHOOK_SECRET:-}" \
			--dry-run=client -o yaml | kubectl apply -f -
		echo "✅ app-secrets created (fallback)"
	else
		echo "✅ app-secrets already exists"
	fi
	# Ensure JWT keys exist in app-secrets and are ES256/P-256. This also
	# rotates the RSA keys generated by older deploys, which the Go services
	# cannot parse (x509.ParseECPrivateKey rejects RSA).
	stored_jwt_private="$(read_stored_jwt_secret JWT_PRIVATE_KEY_PEM)"
	stored_jwt_public="$(read_stored_jwt_secret JWT_PUBLIC_KEY_PEM)"
	if ! is_es256_private_key "$stored_jwt_private" || ! is_es256_public_key "$stored_jwt_public"; then
		patch_app_secret_jwt_keys "$JWT_KEY_PRIVATE" "$JWT_KEY_PUBLIC"
		echo "✅ app-secrets JWT keys rotated to ES256/P-256"
	fi

	# Ensure POSTGRES_USER/PASSWORD/DB are present (fix empty/missing keys in pre-existing secrets)
	kubectl patch secret app-secrets -n fitness-platform-production --type=merge -p "{\"stringData\":{\"POSTGRES_USER\":\"${POSTGRES_USER:-fitness_admin}\",\"POSTGRES_PASSWORD\":\"${POSTGRES_PASSWORD:-}\",\"POSTGRES_DB\":\"${POSTGRES_DB:-fitness}\"}}" >/dev/null 2>&1 || true

	# BACKUP_KEY encrypts the nightly pg_dump. Keep an existing key stable;
	# only generate one when the secret predates backups.
	if [[ -n "${BACKUP_KEY:-}" ]]; then
		kubectl patch secret app-secrets -n fitness-platform-production --type=merge -p "{\"stringData\":{\"BACKUP_KEY\":\"${BACKUP_KEY}\"}}" >/dev/null 2>&1 || true
	elif ! kubectl get secret app-secrets -n fitness-platform-production -o jsonpath='{.data.BACKUP_KEY}' 2>/dev/null | grep -q .; then
		BACKUP_KEY_GENERATED="$(openssl rand -hex 32 2>/dev/null || echo 'fitpulse-backup-key-change-me')"
		kubectl patch secret app-secrets -n fitness-platform-production --type=merge -p "{\"stringData\":{\"BACKUP_KEY\":\"${BACKUP_KEY_GENERATED}\"}}" >/dev/null 2>&1 || true
	fi

	echo "Ensuring app-secrets contains OPEN_WEARABLES_WEBHOOK_SECRET..."
	kubectl patch secret app-secrets -n fitness-platform-production --type=merge -p "{\"stringData\":{\"OPEN_WEARABLES_WEBHOOK_SECRET\":\"${OPEN_WEARABLES_WEBHOOK_SECRET:-}\"}}" >/dev/null 2>&1 || true

	# TELEGRAM_* — для отправки жалоб (напр. на фото) в oncall-bot.
	kubectl patch secret app-secrets -n fitness-platform-production --type=merge -p "{\"stringData\":{\"TELEGRAM_BOT_TOKEN\":\"${TELEGRAM_BOT_TOKEN:-}\",\"TELEGRAM_CHAT_ID\":\"${TELEGRAM_CHAT_ID:-}\"}}" >/dev/null 2>&1 || true

	# Ensure SEED_ADMIN_EMAIL and SEED_ADMIN_PASSWORD exist in app-secrets (required by seed-admin job).
	# Никаких хардкод-креденшелов: если не заданы — провижининг пропускается.
	if [[ -n "${SEED_ADMIN_EMAIL}" && -n "${SEED_ADMIN_PASSWORD}" ]]; then
		kubectl patch secret app-secrets -n fitness-platform-production --type=merge -p "{\"stringData\":{\"SEED_ADMIN_EMAIL\":\"${SEED_ADMIN_EMAIL}\",\"SEED_ADMIN_PASSWORD\":\"${SEED_ADMIN_PASSWORD}\"}}" >/dev/null 2>&1 || true

		# seed-admin cannot produce Argon2id hashes with pgcrypto, so precompute the
		# PHC string through the same code path as user-service and ship it in
		# app-secrets. This must stay in sync with the password patched above.
		SEED_ADMIN_HASH="$(printf '%s' "${SEED_ADMIN_PASSWORD}" | go run ./cmd/hash-password)"
		if [[ -z "${SEED_ADMIN_HASH}" ]]; then
			echo "::error::failed to generate SEED_ADMIN_HASH via cmd/hash-password" >&2
			return 1
		fi
		kubectl patch secret app-secrets -n fitness-platform-production --type=merge -p "{\"stringData\":{\"SEED_ADMIN_HASH\":\"${SEED_ADMIN_HASH}\"}}" >/dev/null 2>&1 || true
	else
		echo "::warning::SEED_ADMIN_EMAIL/SEED_ADMIN_PASSWORD not set — skipping seed-admin provisioning" >&2
	fi

	# Ensure DB_ENCRYPTION_KEY exists in app-secrets (required by user-service
	# pgsodium keyring). Never rotate it automatically: pgsodium derives the PII
	# keyring from this root key, so a new value makes every existing ciphertext
	# undecryptable. Prefer the pinned Actions secret; generate only when the
	# key is genuinely absent (fresh cluster).
	if [[ -n "${DB_ENCRYPTION_KEY:-}" ]]; then
		kubectl patch secret app-secrets -n fitness-platform-production --type=merge -p "{\"stringData\":{\"DB_ENCRYPTION_KEY\":\"${DB_ENCRYPTION_KEY}\"}}" >/dev/null 2>&1 || true
	elif ! kubectl get secret app-secrets -n fitness-platform-production -o jsonpath='{.data.DB_ENCRYPTION_KEY}' 2>/dev/null | grep -q .; then
		DB_ENCRYPTION_KEY_GENERATED="$(openssl rand -hex 32 2>/dev/null || echo 'dev-encryption-key-change-me')"
		kubectl patch secret app-secrets -n fitness-platform-production --type=merge -p "{\"stringData\":{\"DB_ENCRYPTION_KEY\":\"${DB_ENCRYPTION_KEY_GENERATED}\"}}" >/dev/null 2>&1 || true
		echo "✅ app-secrets DB_ENCRYPTION_KEY generated"
	else
		echo "✅ app-secrets DB_ENCRYPTION_KEY already exists"
	fi

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
		# Backup/WAL CronJobs run in fitness-platform-production and cannot
		# reference a Secret from the minio namespace, so mirror the S3
		# credentials there.
		kubectl create secret generic minio-credentials \
			-n fitness-platform-production \
			--from-literal=root-user=fitpulse-admin \
			--from-literal=root-password="${MINIO_SECRET_KEY}" \
			--from-literal=S3_ENDPOINT="http://minio.minio.svc.cluster.local:9000" \
			--dry-run=client -o yaml | kubectl apply --validate=false -f - || echo "::warning::minio-credentials (app namespace) creation failed" >&2
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
