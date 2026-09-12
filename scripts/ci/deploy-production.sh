#!/bin/bash
set -euo pipefail

create_secrets() {
	echo "Generating gRPC mTLS certificates..."
	mkdir -p /tmp/grpc-certs
	cd /tmp/grpc-certs
	openssl genrsa -out ca.key 4096
	openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt -subj "/CN=fitpulse-ca"
	openssl genrsa -out server.key 2048
	openssl req -new -key server.key -out server.csr -subj "/CN=*.fitness-platform-production.svc.cluster.local"
	cat >server-ext.cnf <<'EOF'
[v3_ext]
subjectAltName = DNS:*.fitness-platform-production.svc.cluster.local,DNS:fitness-platform-production.svc.cluster.local
EOF
	openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -sha256 -extfile server-ext.cnf -extensions v3_ext
	openssl genrsa -out client.key 2048
	openssl req -new -key client.key -out client.csr -subj "/CN=fitpulse-client"
	openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 365 -sha256
	echo "✅ gRPC mTLS certificates generated"

	echo "Creating grpc-tls secret..."
	kubectl create secret generic grpc-tls \
		-n fitness-platform-production \
		--from-file=server.crt=/tmp/grpc-certs/server.crt \
		--from-file=server.key=/tmp/grpc-certs/server.key \
		--from-file=ca.crt=/tmp/grpc-certs/ca.crt \
		--from-file=client.crt=/tmp/grpc-certs/client.crt \
		--from-file=client.key=/tmp/grpc-certs/client.key \
		--dry-run=client -o yaml | kubectl apply --validate=false -f -
	echo "✅ grpc-tls secret created"

	echo "Applying ExternalSecret manifests..."
	kubectl apply -f configs/k8s/base/external-secrets/ -n fitness-platform-production
	echo "✅ ExternalSecrets applied"

	echo "Waiting for secrets to propagate..."
	sleep 15

	echo "Creating rabbitmq-secret with rabbitmq.conf..."
	kubectl apply -f configs/k8s/base/secrets/rabbitmq-secret.yaml
	echo "✅ rabbitmq-secret created"

	echo "Creating monitoring-secrets..."
	kubectl create secret generic monitoring-secrets -n fitness-platform-production \
		--from-literal=grafana-admin-password="${GRAFANA_ADMIN_PASSWORD}" \
		--dry-run=client -o yaml | kubectl apply --validate=false -f -
	echo "✅ monitoring-secrets created"
}

ensure_service_account() {
	echo "Checking service account app-service-account..."
	kubectl create serviceaccount app-service-account \
		-n fitness-platform-production \
		--dry-run=client -o yaml | kubectl apply -f -
	kubectl create clusterrolebinding app-service-account-binding \
		--clusterrole=edit \
		--serviceaccount=fitness-platform-production:app-service-account \
		--dry-run=client -o yaml | kubectl apply -f - || true
	echo "ServiceAccount ready"
}

update_image_tags() {
	SHA=$(git rev-parse --short HEAD)
	echo "Updating image tags to :$SHA"
	cd configs/k8s/overlays/production
	for svc in user-service biometric-service training-service gateway classifier; do
		kustomize edit set image "ghcr.io/mamuer/project/$svc=ghcr.io/mamuer/project/$svc:$SHA"
	done
}

cleanup_disk() {
	echo "-> Cleaning up disk space on VPS..."
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
echo '=== Disk usage before cleanup ==='
df -h /
echo ''
echo '=== Docker disk usage ==='
docker system df || true
echo ''
echo '-> Removing dangling Docker images...'
docker image prune -f
echo '-> Removing unused Docker volumes...'
docker volume prune -f || true
echo '-> Removing unused Docker networks...'
docker network prune -f || true
echo '-> Removing stopped containers...'
docker container prune -f || true
echo '-> Cleaning Docker build cache...'
docker builder prune -f --filter 'until=24h' || true
echo ''
echo '-> Cleaning containerd cache (k3s internal)...'
k3s crictl rmi --prune 2>/dev/null || crictl rmi --prune 2>/dev/null || true
echo ''
echo '-> Removing old application images (keep only last 3 versions)...'
for svc in user-service biometric-service training-service gateway device-aggregator classifier; do
	echo \"Cleaning \$svc...\"
	docker images \"ghcr.io/mamuer/project/\$svc\" --format '{{.Repository}}:{{.Tag}} {{.CreatedAt}}' | \
		sort -k2 -r | \
		tail -n +4 | \
		awk '{print \$1}' | \
		xargs -r docker rmi || true
done
echo ''
echo '-> Cleaning system logs older than 7 days...'
sudo journalctl --vacuum-time=7d || true
echo ''
echo '-> Cleaning /tmp files older than 1 day...'
sudo find /tmp -type f -mtime +1 -delete || true
echo ''
echo '=== Disk usage after cleanup ==='
df -h /
echo ''
echo '=== Docker disk usage after cleanup ==='
docker system df || true
echo ''
echo '✅ Disk cleanup completed'
"
}

pre_pull_images() {
	SHA=$(git rev-parse --short HEAD)
	echo "-> Pre-pulling application images with tag $SHA on VPS..."
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
echo 'Logging in to GHCR on VPS...'
echo '${GHCR_TOKEN}' | docker login ghcr.io -u '$GITHUB_ACTOR' --password-stdin || true
for svc in user-service biometric-service training-service gateway device-aggregator classifier; do
	echo \"Pulling ghcr.io/mamuer/project/\$svc:$SHA...\"
	docker pull ghcr.io/mamuer/project/\$svc:$SHA || echo \"Failed to pull \$svc, continuing...\"
done
echo 'Docker images after pre-pull:'
docker images | grep mamuer || true
echo '-> Waiting 60s to let node recover from disk I/O...'
sleep 60
"
}

apply_manifests() {
	echo "Waiting for k8s API internal availability..."
	for i in {1..30}; do
		if kubectl get --raw /healthz &>/dev/null; then
			echo "✅ API internal health OK"
			break
		fi
		echo "Attempt $i/30: waiting for internal API..."
		sleep 5
	done
	echo "Waiting for k8s API nodes..."
	kubectl wait --for=condition=Ready --timeout=5m nodes --all || {
		kubectl cluster-info
		kubectl get nodes -o wide
		exit 1
	}
	echo "Deleting immutable Jobs to allow updates..."
	kubectl delete job migrate-db seed-admin -n fitness-platform-production --ignore-not-found=true || true
	echo "Deleting existing PVCs and PVs to force recreation with fixed provisioner..."
	kubectl delete pvc -n fitness-platform-production --all --ignore-not-found=true || true
	kubectl get pv -o json | jq -r '.items[] | select(.spec.claimRef.namespace == "fitness-platform-production") | .metadata.name' | xargs -r kubectl delete pv || true
	cd configs/k8s/overlays/production
	kustomize build . >/tmp/manifests.yaml
	kubectl delete pod rabbitmq-0 -n fitness-platform-production --ignore-not-found=true || true
	kubectl delete pod -l app=user-service -n fitness-platform-production --ignore-not-found=true || true
	kubectl delete pod -l app=training-service -n fitness-platform-production --ignore-not-found=true || true
	echo "Applying manifests with retries..."
	MAX_RETRIES=5
	for i in $(seq 1 "$MAX_RETRIES"); do
		if kubectl apply --validate=false -f /tmp/manifests.yaml; then
			echo "Production overlay applied successfully"
			break
		else
			echo "Attempt $i/$MAX_RETRIES failed."
			if [ "$i" -lt "$MAX_RETRIES" ]; then
				echo "Waiting 20s before retry..."
				sleep 20
				if ! kubectl get nodes >/dev/null 2>&1; then
					echo "API server is unresponsive. Restarting k3s via SSH..."
					./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo systemctl restart k3s" || true
					echo "Waiting 40s for k3s to restart..."
					sleep 40
					for _j in {1..30}; do
						if kubectl get nodes >/dev/null 2>&1; then
							echo "k3s API is back online."
							break
						fi
						sleep 5
					done
				fi
			else
				echo "Failed to apply manifests after $MAX_RETRIES attempts."
				kubectl get pods -A -o wide || true
				exit 1
			fi
		fi
	done
	echo "Waiting for ConfigMaps to propagate..."
	sleep 15
}

wait_for_rabbitmq() {
	echo "Waiting for RabbitMQ pod to be scheduled..."
	for i in {1..60}; do
		POD_NAME=$(kubectl get pods -l app=rabbitmq -n fitness-platform-production -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
		if [ -n "$POD_NAME" ]; then
			echo "✅ RabbitMQ pod found: $POD_NAME"
			break
		fi
		echo "Attempt $i/60: waiting for RabbitMQ pod..."
		sleep 2
	done
	if [ -z "$POD_NAME" ]; then
		echo "❌ RabbitMQ pod not scheduled. Checking events..."
		kubectl get events -n fitness-platform-production --sort-by='.lastTimestamp' | grep -i rabbitmq || true
		kubectl describe statefulset rabbitmq -n fitness-platform-production || true
		exit 1
	fi
	echo "Waiting for RabbitMQ StatefulSet to be ready..."
	kubectl rollout status statefulset/rabbitmq -n fitness-platform-production --timeout=180s || {
		echo "❌ RabbitMQ not ready. Logs:"
		kubectl logs statefulset/rabbitmq -n fitness-platform-production --tail=100 || true
		exit 1
	}
	echo "✅ RabbitMQ is ready"
}

verify_postgres() {
	echo "=== Checking resources in namespace ==="
	kubectl get all -n fitness-platform-production
	echo "=== Checking StatefulSet specifically ==="
	if ! kubectl get statefulset postgres -n fitness-platform-production >/dev/null 2>&1; then
		echo "❌ StatefulSet 'postgres' not found. Re-applying manifests..."
		cd configs/k8s/overlays/production
		kustomize build . | kubectl apply --validate=false -f -
		echo "Waiting 30s for StatefulSet to appear..."
		sleep 30
	fi
	kubectl get statefulset postgres -n fitness-platform-production -o wide || true
	kubectl describe statefulset postgres -n fitness-platform-production || true
	echo "=== Checking pods ==="
	kubectl get pods -n fitness-platform-production -o wide
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

debug_pod_status() {
	for i in {1..30}; do
		if kubectl get --raw=/healthz &>/dev/null; then
			break
		fi
		if [ "$i" -eq 30 ]; then
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
	kubectl get events -n fitness-platform-production --sort-by='.lastTimestamp' | tail -20 || true
}

run_migrations() {
	echo "Running database migrations via Flyway..."
	kubectl delete job migrate-db -n fitness-platform-production --ignore-not-found=true || true
	kubectl apply -f configs/k8s/base/jobs/migrate-db.yaml -n fitness-platform-production
	echo "Waiting for migrate-db job to complete..."
	kubectl wait --for=condition=complete job/migrate-db -n fitness-platform-production --timeout=300s
	echo "✅ Database migrations completed successfully"
	echo "Verifying tables were created..."
	kubectl exec -n fitness-platform-production postgres-0 -- \
		psql -U postgres -d fitness -c "SELECT table_name FROM information_schema.tables WHERE table_schema='public' ORDER BY table_name;"
}

run_seed_admin() {
	echo "=== Running seed-admin job ==="
	echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	echo "-> Checking required secrets..."
	if [ -z "${SEED_ADMIN_EMAIL}" ]; then
		echo "❌ SEED_ADMIN_EMAIL is empty!"
		exit 1
	fi
	if [ -z "${SEED_ADMIN_PASSWORD}" ]; then
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
		if [ -n "$POD_NAME" ]; then
			echo "✅ Pod found: $POD_NAME"
			echo "-> Pod status: $(kubectl get pod "$POD_NAME" -n fitness-platform-production -o jsonpath='{.status.phase}' 2>/dev/null || echo 'unknown')"
			break
		fi
		echo "  Attempt $i/30: waiting for pod... (5s)"
		sleep 5
	done
	if [ -z "$POD_NAME" ]; then
		echo "❌ Failed to find seed-admin pod after 150s"
		echo "-> Checking all pods in namespace:"
		kubectl get pods -n fitness-platform-production -o wide 2>&1 || true
		exit 1
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
			echo "⚠️ Pod is in error state, fetching diagnostics..."
			echo "→ Pod describe:"
			kubectl describe pod "$POD_NAME" -n fitness-platform-production 2>&1 || true
			echo "→ Last 200 log lines:"
			kubectl logs "$POD_NAME" -n fitness-platform-production --tail=200 2>&1 || true
			echo "→ Pod events:"
			kubectl get events -n fitness-platform-production --field-selector involvedObject.name="$POD_NAME" --sort-by='.lastTimestamp' 2>&1 | tail -20 || true
			break
		fi
		if kubectl wait --for=condition=complete job/seed-admin -n fitness-platform-production --timeout=10s 2>/dev/null; then
			echo "✅ Job completed successfully"
			break
		fi
		ELAPSED=$(($(date +%s) - MONITOR_START))
		if [ "$ELAPSED" -ge 300 ]; then
			echo "⏱️  Monitoring timeout reached (300s)"
			break
		fi
		sleep 10
	done
	if [ -n "$LOG_STREAM_PID" ]; then
		kill $LOG_STREAM_PID 2>/dev/null || true
		wait $LOG_STREAM_PID 2>/dev/null || true
	fi
	echo "-> Final job status check..."
	if kubectl wait --for=condition=complete job/seed-admin -n fitness-platform-production --timeout=30s 2>/dev/null; then
		echo "✅ Admin user seeded successfully"
		echo "-> Final logs:"
		kubectl logs job/seed-admin -n fitness-platform-production --tail=100 2>&1 || true
	else
		echo "⚠️ Job did not complete successfully, detailed diagnostics..."
		POD_NAME=$(kubectl get pods -n fitness-platform-production -l app=seed-admin -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "unknown")
		echo "Current pod name: $POD_NAME"
		if [ "$POD_NAME" != "unknown" ] && [ -n "$POD_NAME" ]; then
			echo "→ Pod describe:"
			kubectl describe pod "$POD_NAME" -n fitness-platform-production 2>&1 || true
			echo "→ Pod logs (last 300 lines):"
			kubectl logs "$POD_NAME" -n fitness-platform-production --tail=300 2>&1 || true
			echo "→ Previous container logs (if any):"
			kubectl logs "$POD_NAME" -n fitness-platform-production --previous --tail=100 2>&1 || true
			echo "→ Pod events:"
			kubectl get events -n fitness-platform-production --field-selector involvedObject.name="$POD_NAME" --sort-by='.lastTimestamp' 2>&1 | tail -30 || true
		else
			echo "→ No active seed-admin pod found"
			echo "→ Checking recent events in namespace:"
			kubectl get events -n fitness-platform-production --sort-by='.lastTimestamp' 2>&1 | grep -E "(seed-admin|Error|Failed|BackOff)" | tail -30 || true
		fi
		echo "→ Job describe:"
		kubectl describe job seed-admin -n fitness-platform-production 2>&1 || true
		echo "→ Checking if admin exists in database (idempotent check)..."
		DB_CHECK=$(kubectl exec -n fitness-platform-production postgres-0 -- \
			psql -U postgres -d fitness -tAc \
			"SELECT COUNT(*) FROM users WHERE email='${SEED_ADMIN_EMAIL}'" 2>/dev/null || echo "0")
		if [ "$DB_CHECK" = "1" ]; then
			echo "✅ Admin user already exists in database (idempotent check passed)"
		else
			echo "❌ seed-admin job failed AND admin user not found in database"
			echo "Database check result: $DB_CHECK"
			exit 1
		fi
	fi
	echo "-> Cleaning up completed job..."
	kubectl delete job seed-admin -n fitness-platform-production --ignore-not-found=true 2>&1 || true
	echo "✅ seed-admin step completed"
}

check_logs() {
	echo "Waiting for pods to stabilize..."
	sleep 30
	echo "Checking pod status..."
	kubectl get pods -n fitness-platform-production
	echo "Checking logs for critical errors..."
	ERROR_FOUND=0
	for dep in user-service training-service rabbitmq; do
		echo "=== Logs for $dep ==="
		if [[ "$dep" == "rabbitmq" ]]; then
			LOGS=$(kubectl logs statefulset/rabbitmq -n fitness-platform-production --tail=50 2>&1 || echo "Failed to get logs")
		else
			LOGS=$(kubectl logs deployment/$dep -n fitness-platform-production --tail=50 2>&1 || echo "Failed to get logs")
		fi
		echo "$LOGS"
		if echo "$LOGS" | grep -qE "too many colons|deprecated environment variables|RABBITMQ_DEFAULT_PASS_FILE|RABBITMQ_DEFAULT_USER_FILE|RABBITMQ_VM_MEMORY_HIGH_WATERMARK|Failed to listen"; then
			echo "❌ Found critical error in $dep logs"
			ERROR_FOUND=1
		fi
	done
	for app in user-service training-service biometric-service gateway device-aggregator classifier valkey; do
		if ! kubectl get pods -l app=$app -n fitness-platform-production | grep -q Running; then
			echo "❌ Pod(s) for $app are not Running"
			ERROR_FOUND=1
		fi
	done
	if [ $ERROR_FOUND -eq 1 ]; then
		echo "❌ Critical errors detected, failing the deployment"
		exit 1
	else
		echo "✅ No critical errors found, all services are healthy"
	fi
}

verify_deployment() {
	for i in {1..30}; do
		if kubectl get --raw=/healthz &>/dev/null; then
			break
		fi
		if [ "$i" -eq 30 ]; then
			echo "API not available, restarting k3s..."
			./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo systemctl restart k3s"
			sleep 30
		fi
		sleep 5
	done
	echo "Verifying deployment..."
	echo "Waiting for gateway to be ready (up to 50s)..."
	kubectl get pods -n fitness-platform-production
	kubectl wait --for=condition=ready pod -l app=gateway -n fitness-platform-production --timeout=50s || echo "Gateway not ready yet"
	sleep 30
}

deploy_open_wearables() {
	echo "=== Deploying Open Wearables ==="
	kubectl apply -k configs/k8s/base/ -n fitness-platform-production
	kubectl apply -f configs/k8s/base/jobs/create-open-wearables-db.yaml -n fitness-platform-production
	kubectl apply -k configs/k8s/overlays/production/ -n fitness-platform-production
	echo "Waiting for Open Wearables pods to be ready..."
	kubectl wait --for=condition=ready pod -l app=open-wearables -n fitness-platform-production --timeout=300s || echo "Open Wearables backend not ready yet"
	kubectl wait --for=condition=ready pod -l app=open-wearables-frontend -n fitness-platform-production --timeout=300s || echo "Open Wearables frontend not ready yet"
	echo "✅ Open Wearables deployed"
}

main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all)
		create_secrets
		ensure_service_account
		update_image_tags
		cleanup_disk
		pre_pull_images
		apply_manifests
		wait_for_rabbitmq
		verify_postgres
		run_migrations
		run_seed_admin
		check_logs
		verify_deployment
		deploy_open_wearables
		;;
	create_secrets) create_secrets ;;
	ensure_service_account) ensure_service_account ;;
	update_image_tags) update_image_tags ;;
	cleanup_disk) cleanup_disk ;;
	pre_pull_images) pre_pull_images ;;
	apply_manifests) apply_manifests ;;
	wait_for_rabbitmq) wait_for_rabbitmq ;;
	verify_postgres) verify_postgres ;;
	create_image_pull_secret) create_image_pull_secret ;;
	debug_pod_status) debug_pod_status ;;
	run_migrations) run_migrations ;;
	run_seed_admin) run_seed_admin ;;
	check_logs) check_logs ;;
	verify_deployment) verify_deployment ;;
	deploy_open_wearables) deploy_open_wearables ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
