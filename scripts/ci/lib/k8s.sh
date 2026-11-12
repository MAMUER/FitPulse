#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/utils.sh"

apply_manifests() {
	echo "Waiting for k8s API internal availability..."
	for i in {1..30}; do
		if kubectl get --raw /healthz >/dev/null 2>&1; then
			echo "API internal health OK"
			break
		fi
		echo "Attempt $i/30: waiting for internal API..."
		sleep 5
	done
	echo "Waiting for k8s API nodes..."
	kubectl wait --for=condition=Ready --timeout=5m nodes --all || {
		kubectl cluster-info || true
		kubectl get nodes -o wide || true
		exit 1
	}

	echo "Deleting immutable Jobs for fresh recreation..."
	for job in migrate-db seed-admin create-open-wearables-db; do
		kubectl delete job "$job" -n fitness-platform-production --ignore-not-found=true --wait=false 2>/dev/null || true
	done

	echo "Applying manifests idempotently..."
	cd configs/k8s/overlays/production
	if ! kustomize build --load-restrictor=LoadRestrictionsNone . | kubectl apply --validate=false --timeout=60s -f -; then
		echo "Failed to apply manifests" >&2
		kubectl get pods -A -o wide || true
		exit 1
	fi
	echo "Production overlay applied successfully"

	echo "Waiting for ConfigMaps to propagate..."
	sleep 15
	cd - >/dev/null
}

# Kept separate from apply_manifests on purpose: app pods must not be restarted
# before the database schema exists, otherwise user-service crash-loops on
# "relation pgsodium.key does not exist" and burns its restart budget while
# migrate-db is still running.
restart_app_deployments() {
	echo "Rolling restart of application deployments..."
	for deploy in user-service biometric-service training-service gateway classifier ml-generator device-aggregator data-processor open-wearables open-wearables-frontend; do
		kubectl rollout restart deployment/"$deploy" -n fitness-platform-production 2>/dev/null || true
	done
}

restart_rabbitmq_pod() {
	echo '-> Refreshing RabbitMQ pod to pick up updated StatefulSet spec/volumes...'
	if kubectl get pod rabbitmq-0 -n fitness-platform-production --ignore-not-found=true >/dev/null 2>&1; then
		kubectl delete pod rabbitmq-0 -n fitness-platform-production --request-timeout=20s --wait=false >/dev/null 2>&1 || true
		sleep 15
		echo '-> RabbitMQ pod deleted; StatefulSet will recreate it with the current spec'
	else
		echo '-> RabbitMQ pod not present yet, skipping refresh'
	fi
}

wait_for_rabbitmq() {
	echo "Waiting for RabbitMQ pod to be scheduled..."
	for i in {1..60}; do
		POD_NAME=$(kubectl get pods -l app=rabbitmq -n fitness-platform-production -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
		if [[ -n "$POD_NAME" ]]; then
			echo "✅ RabbitMQ pod found: $POD_NAME"
			break
		fi
		echo "Attempt $i/60: waiting for RabbitMQ pod..."
		sleep 2
	done
	if [[ -z "$POD_NAME" ]]; then
		echo "::warning::RabbitMQ pod not scheduled. Checking events..."
		kubectl get events -n fitness-platform-production --sort-by=.lastTimestamp | grep -i rabbitmq || true
		kubectl describe statefulset rabbitmq -n fitness-platform-production || true
	fi
	echo "Waiting for RabbitMQ StatefulSet to be ready..."
	if kubectl rollout status statefulset/rabbitmq -n fitness-platform-production --timeout=180s; then
		echo "✅ RabbitMQ is ready"
	else
		echo "::error::RabbitMQ is not ready — every backend service depends on it" >&2
		kubectl get pods -n fitness-platform-production -l app=rabbitmq -o wide >&2 || true
		kubectl describe statefulset rabbitmq -n fitness-platform-production >&2 || true
		kubectl get events -n fitness-platform-production --field-selector involvedObject.name=rabbitmq-0 --sort-by=.lastTimestamp 2>/dev/null | tail -20 >&2 || true
		return 1
	fi
}

verify_postgres() {
	echo "=== Checking resources in namespace ==="
	kubectl get all -n fitness-platform-production
	echo "=== Checking StatefulSet specifically ==="
	if ! kubectl get statefulset postgres -n fitness-platform-production >/dev/null 2>&1; then
		echo "::warning::StatefulSet 'postgres' not found. Re-applying manifests..."
		cd configs/k8s/overlays/production
		kustomize build --load-restrictor=LoadRestrictionsNone . | timeout 120 kubectl apply --validate=false --timeout=30s --force -f - || true
		echo "Waiting 30s for StatefulSet to appear..."
		sleep 30
	fi
	kubectl get statefulset postgres -n fitness-platform-production -o wide || true
	kubectl describe statefulset postgres -n fitness-platform-production || true
	echo "=== Checking pods ==="
	kubectl get pods -n fitness-platform-production -o wide
}

run_migrations() {
	echo "Running database migrations (V1__full_schema.sql)..."
	kubectl delete job migrate-db -n fitness-platform-production --ignore-not-found=true || true
	kubectl apply -f configs/k8s/base/jobs/migrate-db.yaml -n fitness-platform-production
	echo "Waiting for migrate-db job to complete..."
	if kubectl wait --for=condition=complete job/migrate-db -n fitness-platform-production --timeout=300s; then
		echo "✅ Database migrations completed successfully"
	else
		# A failed migration used to only warn, which let the deploy continue with
		# an empty schema: every service then crash-looped (user-service fatal on
		# "relation pgsodium.key does not exist"). Surface the reason and stop.
		echo "::error::migrate-db job did not complete — dumping diagnostics" >&2
		kubectl describe job migrate-db -n fitness-platform-production || true
		kubectl describe pod -n fitness-platform-production -l app=migrate-db || true
		kubectl logs job/migrate-db -n fitness-platform-production --all-containers --tail=100 || true
		kubectl get pods -n fitness-platform-production -l app=migrate-db -o wide || true
		return 1
	fi
	echo "Verifying tables were created..."
	kubectl exec -n fitness-platform-production postgres-0 -- \
		psql -U "${POSTGRES_USER:-fitness_admin}" -d "${POSTGRES_DB:-fitness}" -c "SELECT table_name FROM information_schema.tables WHERE table_schema='public' ORDER BY table_name;" || echo "::warning::table verification failed" >&2
}

run_seed_admin() {
	echo "=== Running seed-admin job ==="
	echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	echo "-> Checking required secrets..."
	if [[ -z "${SEED_ADMIN_EMAIL}" ]]; then
		echo "❌ SEED_ADMIN_EMAIL is empty!"
		exit 1
	fi
	if [[ -z "${SEED_ADMIN_PASSWORD}" ]]; then
		echo "❌ SEED_ADMIN_PASSWORD is empty!"
		exit 1
	fi
	echo "✅ Secrets present (values hidden)"
	echo "-> Cleaning up existing seed-admin job..."
	kubectl delete job seed-admin -n fitness-platform-production --ignore-not-found=true --wait=true 2>&1 || true
	sleep 5
	echo "-> Applying seed-admin Job from YAML..."
	kubectl apply -f configs/k8s/base/jobs/seed-admin.yaml -n fitness-platform-production 2>&1
	echo "-> Waiting for seed-admin pod to be scheduled..."
	POD_NAME=""
	for i in {1..30}; do
		POD_NAME=$(kubectl get pods -n fitness-platform-production -l app=seed-admin -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
		if [[ -n "$POD_NAME" ]]; then
			echo "✅ Pod found: $POD_NAME"
			echo "-> Pod status: $(kubectl get pod "$POD_NAME" -n fitness-platform-production -o jsonpath='{.status.phase}' 2>/dev/null || echo 'unknown')"
			break
		fi
		echo "  Attempt $i/30: waiting for pod... (5s)"
		sleep 5
	done
	if [[ -z "$POD_NAME" ]]; then
		echo "::warning::Failed to find seed-admin pod after 150s" >&2
		kubectl get pods -n fitness-platform-production -o wide 2>&1 || true
		return 0
	fi
	echo "-> Starting background log streaming for pod $POD_NAME..."
	kubectl logs -f -n fitness-platform-production "$POD_NAME" --tail=100 2>&1 &
	LOG_STREAM_PID=$!
	echo "-> Monitoring pod status..."
	MONITOR_START=$(date +%s)
	while true; do
		CURRENT_PHASE=$(kubectl get pod "$POD_NAME" -n fitness-platform-production -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
		RESTART_COUNT=$(kubectl get pod "$POD_NAME" -n fitness-platform-production -o jsonpath='{.status.containerStatuses[0].restartCount}' 2>/dev/null || echo "0")
		echo "  [$(date +%H:%M:%S)] Phase: $CURRENT_PHASE, Restarts: $RESTART_COUNT"
		if [[ "$CURRENT_PHASE" == "Failed" || "$CURRENT_PHASE" == "CrashLoopBackOff" || "$RESTART_COUNT" -gt 2 ]]; then
			echo "::warning::Pod is in error state, fetching diagnostics..." >&2
			kubectl describe pod "$POD_NAME" -n fitness-platform-production 2>&1 || true
			kubectl logs "$POD_NAME" -n fitness-platform-production --tail=200 2>&1 || true
			break
		fi
		if kubectl wait --for=condition=complete job/seed-admin -n fitness-platform-production --timeout=10s 2>/dev/null; then
			echo "✅ Job completed successfully"
			break
		fi
		ELAPSED=$(($(date +%s) - MONITOR_START))
		if [[ "$ELAPSED" -ge 300 ]]; then
			echo "⏱️  Monitoring timeout reached (300s)"
			break
		fi
		sleep 10
	done
	if [[ -n "$LOG_STREAM_PID" ]]; then
		kill $LOG_STREAM_PID 2>/dev/null || true
		wait $LOG_STREAM_PID 2>/dev/null || true
	fi
	echo "-> Final job status check..."
	if kubectl wait --for=condition=complete job/seed-admin -n fitness-platform-production --timeout=30s 2>/dev/null; then
		echo "✅ Admin user seeded successfully"
		kubectl logs job/seed-admin -n fitness-platform-production --tail=100 2>&1 || true
	else
		echo "::warning::Job did not complete successfully, detailed diagnostics..."
		POD_NAME=$(kubectl get pods -n fitness-platform-production -l app=seed-admin -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "unknown")
		if [[ "$POD_NAME" != "unknown" ]] && [[ -n "$POD_NAME" ]]; then
			kubectl describe pod "$POD_NAME" -n fitness-platform-production 2>&1 || true
			kubectl logs "$POD_NAME" -n fitness-platform-production --tail=300 2>&1 || true
		fi
		echo "::warning::Checking if admin exists in database (idempotent check)..."
		DB_CHECK=$(kubectl exec -n fitness-platform-production postgres-0 -- \
			psql -U postgres -d fitness -tAc \
			"SELECT COUNT(*) FROM users WHERE email='${SEED_ADMIN_EMAIL}'" 2>/dev/null || echo "0")
		if [[ "$DB_CHECK" = "1" ]]; then
			echo "✅ Admin user already exists in database (idempotent check passed)"
		else
			echo "::warning::seed-admin job failed AND admin user not found in database" >&2
		fi
	fi
	echo "-> Cleaning up completed job..."
	kubectl delete job seed-admin -n fitness-platform-production --ignore-not-found=true 2>&1 || true
	echo "✅ seed-admin step completed"
}

verify_deployment() {
	for i in {1..30}; do
		if kubectl get --raw /healthz &>/dev/null; then
			break
		fi
		if [[ "$i" -eq 30 ]]; then
			echo "API not available, restarting k3s..."
			./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo systemctl restart k3s"
			sleep 30
		fi
		sleep 5
	done
	echo "Verifying deployment..."
	echo "Waiting for gateway to be ready (up to 50s)..."
	kubectl get pods -n fitness-platform-production
	kubectl wait --for=condition=ready pod -l app=gateway -n fitness-platform-production --timeout=50s || echo "::warning::Gateway not ready yet"
	sleep 30
}

deploy_open_wearables() {
	echo "=== Deploying Open Wearables ==="
	kubectl apply -f configs/k8s/base/jobs/create-open-wearables-db.yaml -n fitness-platform-production
	kubectl apply -k configs/k8s/overlays/production/ --validate=false || true
	echo "Waiting for Open Wearables pods to be ready..."
	kubectl wait --for=condition=ready pod -l app=open-wearables -n fitness-platform-production --timeout=300s || echo "::warning::Open Wearables backend not ready yet"
	kubectl wait --for=condition=ready pod -l app=open-wearables-frontend -n fitness-platform-production --timeout=300s || echo "::warning::Open Wearables frontend not ready yet"
	echo "✅ Open Wearables deployed"
}

debug_pod_status() {
	for i in {1..30}; do
		if kubectl get --raw /healthz &>/dev/null; then
			break
		fi
		if [[ "$i" -eq 30 ]]; then
			echo "API not available, restarting k3s..."
			./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo systemctl restart k3s"
			sleep 30
		fi
		sleep 5
	done
	echo "=== Checking pod status ==="
	kubectl get pods -n fitness-platform-production -o wide || true
	echo ""
	echo "=== Logs for gateway deployment ==="
	kubectl logs deployment/gateway -n fitness-platform-production --tail=100 || echo "No logs for gateway"
	echo ""
	echo "=== Logs for user-service deployment ==="
	kubectl logs deployment/user-service -n fitness-platform-production --tail=100 || echo "No logs for user-service"
	echo ""
	echo "=== Logs for RabbitMQ (if exists) ==="
	kubectl logs statefulset/rabbitmq -n fitness-platform-production --tail=50 || echo "RabbitMQ not running"
	echo ""
	echo "=== Recent events ==="
	kubectl get events -n fitness-platform-production --sort-by=.lastTimestamp | tail -20 || true
}

check_logs() {
	echo "Waiting for pods to stabilize..."
	sleep 30
	echo "Checking pod status..."
	kubectl get pods -n fitness-platform-production
	echo "Checking logs for critical errors..." >&2
	ERROR_FOUND=0
	for dep in user-service training-service rabbitmq; do
		echo "=== Logs for $dep ==="
		if [[ "$dep" == "rabbitmq" ]]; then
			LOGS=$(kubectl logs statefulset/rabbitmq -n fitness-platform-production --tail=50 2>&1 || echo "Failed to get logs") >&2
		else
			LOGS=$(kubectl logs deployment/$dep -n fitness-platform-production --tail=50 2>&1 || echo "Failed to get logs") >&2
		fi
		echo "$LOGS"
		if echo "$LOGS" | grep -qE "too many colons|deprecated environment variables|RABBITMQ_DEFAULT_PASS_FILE|RABBITMQ_DEFAULT_USER_FILE|RABBITMQ_VM_MEMORY_HIGH_WATERMARK|Failed to listen"; then
			echo "::warning::Found critical error in $dep logs" >&2
			ERROR_FOUND=1
		fi
	done
	for app in user-service training-service biometric-service gateway device-aggregator classifier valkey; do
		if ! kubectl get pods -l app=$app -n fitness-platform-production | grep -q Running; then
			echo "::warning::Pod(s) for $app are not Running"
			ERROR_FOUND=1
		fi
	done
	if [[ $ERROR_FOUND -eq 1 ]]; then
		echo "::warning::Critical errors detected, continuing deployment" >&2
	else
		echo "✅ No critical errors found, all services are healthy" >&2
	fi
}
