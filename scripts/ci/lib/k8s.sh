#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/utils.sh"

# Remove resources that accumulated on the long-lived VPS:
#   * ReplicaSets older than 1h scaled to 0 (revisionHistoryLimit only prunes
#     new revisions; the 5-day-old controller history had dozens per service)
#   * completed/failed Jobs and CronJob Jobs stuck active for >30m
#   * orphan cert-manager HTTP-01 solver pods/services
#   * HPAs (min=max=1 was a no-op that only produced "cpu: <unknown>" errors)
#   * the pre-fix `postgres-wal-archive` PVC (the StatefulSet creates
#     `postgres-wal-archive-postgres-0`; CronJobs now reference that name)
cleanup_cluster_resources() {
	echo "=== Cleaning up stale cluster resources ==="
	local ns="fitness-platform-production"
	local stale

	for namespace in "$ns" ingress-nginx minio monitoring elk; do
		stale=$(kubectl get rs -n "$namespace" -o json 2>/dev/null |
			jq -r '.items[]
				| select((.spec.replicas // 0) == 0)
				| select(.metadata.creationTimestamp != null)
				| select((now - (.metadata.creationTimestamp | fromdateiso8601)) > 3600)
				| .metadata.name' 2>/dev/null || true)
		if [[ -n "$stale" ]]; then
			echo "-> Pruning $(printf '%s\n' "$stale" | wc -l) stale ReplicaSet(s) in $namespace"
			printf '%s\n' "$stale" | xargs -r -n 20 kubectl delete rs -n "$namespace" --wait=false >/dev/null 2>&1 || true
		fi
	done

	stale=$(kubectl get jobs -n "$ns" -o json 2>/dev/null |
		jq -r '.items[]
			| select(
				((.status.succeeded // 0) > 0)
				or ((.status.failed // 0) > 0)
				or (((.status.active // 0) > 0)
					and (.metadata.creationTimestamp != null)
					and ((now - (.metadata.creationTimestamp | fromdateiso8601)) > 1800))
			)
			| .metadata.name' 2>/dev/null || true)
	if [[ -n "$stale" ]]; then
		echo "-> Deleting $(printf '%s\n' "$stale" | wc -l) stale Job(s)"
		printf '%s\n' "$stale" | xargs -r -n 20 kubectl delete job -n "$ns" --wait=false >/dev/null 2>&1 || true
	fi

	# cert-manager HTTP-01 solver pods/services are left behind when the
	# ingress controller was crash-looping (the previous RBAC bug).
	kubectl get pods -n "$ns" -o name 2>/dev/null | grep 'cm-acme-http-solver' |
		xargs -r -n 20 kubectl delete -n "$ns" --wait=false >/dev/null 2>&1 || true
	kubectl get svc -n "$ns" -o name 2>/dev/null | grep 'cm-acme-http-solver' |
		xargs -r -n 20 kubectl delete -n "$ns" --wait=false >/dev/null 2>&1 || true

	kubectl delete hpa --all -n "$ns" --ignore-not-found=true >/dev/null 2>&1 || true

	# Ingresses removed from the production overlay. kubectl apply does not
	# prune resources deleted from kustomize, so remove the stale objects (and
	# their duplicate host routes) explicitly.
	kubectl delete ingress jaeger-ingress open-wearables-ingress -n "$ns" \
		--ignore-not-found=true >/dev/null 2>&1 || true

	# Orphan PVC from the pre-fix WAL CronJob name. Only delete it when no pod
	# references it; the live StatefulSet volume is postgres-wal-archive-postgres-0.
	if kubectl get pvc postgres-wal-archive -n "$ns" >/dev/null 2>&1 &&
		! { kubectl get pods -n "$ns" -o json 2>/dev/null |
			jq -e '.items[] | select(.spec.volumes[]? | .persistentVolumeClaim?.claimName == "postgres-wal-archive")' >/dev/null 2>&1; }; then
		echo "-> Deleting orphan PVC postgres-wal-archive"
		kubectl delete pvc postgres-wal-archive -n "$ns" --wait=false >/dev/null 2>&1 || true
	fi

	echo "✅ Cleanup complete"
}

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

	cleanup_cluster_resources

	echo "Deleting immutable Jobs for fresh recreation..."
	for job in migrate-db seed-admin create-open-wearables-db reset-open-wearables-db; do
		kubectl delete job "$job" -n fitness-platform-production --ignore-not-found=true --wait=false 2>/dev/null || true
	done
	if kubectl get namespace elk >/dev/null 2>&1; then
		kubectl delete job elasticsearch-init -n elk --ignore-not-found=true --wait=true --timeout=30s
	fi

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
	# open-wearables is intentionally excluded: its database is created later by
	# deploy_open_wearables, and restarting it before that only produces a
	# crash-loop plus an extra ReplicaSet. Its image is already rolled out by
	# apply_manifests.
	for deploy in user-service biometric-service training-service gateway classifier ml-generator device-aggregator data-processor; do
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
	local namespace="fitness-platform-production"
	local job="migrate-db"
	local job_conditions job_query_error

	echo "Running database migrations (V1__full_schema.sql)..."
	kubectl delete job "$job" -n "$namespace" --ignore-not-found=true --wait=true
	kubectl apply -f configs/k8s/base/jobs/migrate-db.yaml -n "$namespace"
	echo "Waiting for migrate-db job to complete or fail..."
	for attempt in {1..60}; do
		job_query_error=""
		if job_conditions=$(kubectl get job "$job" -n "$namespace" \
			-o jsonpath='{range .status.conditions[*]}{.type}={.status}{"\n"}{end}' 2>&1); then
			if grep -q '^Complete=True$' <<<"$job_conditions"; then
				echo "✅ Database migrations completed successfully"
				if ! kubectl logs "job/$job" -n "$namespace" --all-containers=true \
					--timestamps --tail=200 2>&1; then
					echo "::warning::Could not retrieve completed migration logs" >&2
				fi
				break
			fi
			if grep -q '^Failed=True$' <<<"$job_conditions"; then
				echo "::error::migrate-db job failed; collecting diagnostics before pod TTL cleanup" >&2
				diagnose_job "$namespace" "$job"
				return 1
			fi
		else
			job_query_error="$job_conditions"
		fi
		if ((attempt % 6 == 0)); then
			if [[ -n "$job_query_error" ]]; then
				echo "::warning::Unable to read migrate-db job status: $job_query_error" >&2
			else
				echo "  [$(date +%H:%M:%S)] Job still active after $((attempt * 5)) seconds"
			fi
		fi
		if [[ "$attempt" -eq 60 ]]; then
			echo "::error::migrate-db job did not complete within 300 seconds" >&2
			diagnose_job "$namespace" "$job"
			return 1
		fi
		sleep 5
	done
	echo "Verifying tables were created..."
	kubectl exec -n fitness-platform-production postgres-0 -- \
		psql -U "${POSTGRES_USER:-fitness_admin}" -d "${POSTGRES_DB:-fitness}" -c "SELECT table_name FROM information_schema.tables WHERE table_schema='public' ORDER BY table_name;" || echo "::warning::table verification failed" >&2
}

diagnose_job() {
	local namespace="$1"
	local job="$2"
	local pods pod containers restart_status container restart_count

	echo "=== Job diagnostics: $namespace/$job ==="
	kubectl describe job "$job" -n "$namespace" 2>&1 || true
	pods=$(kubectl get pods -n "$namespace" -l "job-name=$job" -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)
	if [[ -z "$pods" ]]; then
		echo "No pods found for job $job"
	else
		while IFS= read -r pod; do
			[[ -z "$pod" ]] && continue
			echo "--- Pod: $pod ---"
			kubectl describe pod "$pod" -n "$namespace" 2>&1 || true
			containers=$(kubectl get pod "$pod" -n "$namespace" -o jsonpath='{range .spec.initContainers[*]}{.name}{"\n"}{end}{range .spec.containers[*]}{.name}{"\n"}{end}' 2>/dev/null || true)
			restart_status=$(kubectl get pod "$pod" -n "$namespace" -o jsonpath='{range .status.initContainerStatuses[*]}{.name}={.restartCount}{"\n"}{end}{range .status.containerStatuses[*]}{.name}={.restartCount}{"\n"}{end}' 2>/dev/null || true)
			while IFS= read -r container; do
				[[ -z "$container" ]] && continue
				echo "--- Current logs: $container ---"
				kubectl logs "$pod" -n "$namespace" -c "$container" --timestamps --tail=200 2>&1 || true
				restart_count=$(printf '%s\n' "$restart_status" | awk -F= -v name="$container" '$1 == name {print $2; exit}')
				if [[ "$restart_count" =~ ^[0-9]+$ ]] && ((restart_count > 0)); then
					echo "--- Previous logs: $container (restart count: $restart_count) ---"
					kubectl logs "$pod" -n "$namespace" -c "$container" --previous --timestamps --tail=200 2>&1 || true
				fi
			done <<<"$containers"
			echo "--- Events for pod $pod ---"
			kubectl get events -n "$namespace" --field-selector "involvedObject.name=$pod" --sort-by=.lastTimestamp 2>&1 || true
		done <<<"$pods"
	fi
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
	echo "-> Monitoring pod status..."
	MONITOR_START=$(date +%s)
	JOB_COMPLETE=false
	for i in {1..30}; do
		JOB_CONDITIONS=$(kubectl get job seed-admin -n fitness-platform-production -o jsonpath='{range .status.conditions[*]}{.type}={.status}{"\n"}{end}' 2>/dev/null || echo "NotFound")
		if grep -q '^Complete=True$' <<<"$JOB_CONDITIONS"; then
			echo "✅ Job completed successfully"
			JOB_COMPLETE=true
			break
		fi
		if grep -q '^Failed=True$' <<<"$JOB_CONDITIONS" || [[ "$JOB_CONDITIONS" == "NotFound" ]]; then
			echo "::error::seed-admin job failed before completion" >&2
			break
		fi
		POD_NAME=$(kubectl get pods -n fitness-platform-production -l app=seed-admin -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
		if [[ -n "$POD_NAME" ]]; then
			POD_STATE=$(kubectl get pod "$POD_NAME" -n fitness-platform-production -o jsonpath='{.status.phase}{" "}{range .status.containerStatuses[*]}{.state.waiting.reason}{.state.terminated.reason} restarts={.restartCount}{" "}{end}' 2>/dev/null || echo "Unknown")
			echo "  [$(date +%H:%M:%S)] Pod $POD_NAME: ${POD_STATE:-status pending}"
		else
			echo "  [$(date +%H:%M:%S)] Waiting for seed-admin pod..."
		fi
		if [[ "$(($(date +%s) - MONITOR_START))" -ge 300 ]]; then
			echo "::error::Timed out waiting for seed-admin job after 300 seconds" >&2
			break
		fi
		sleep 10
	done

	SEED_STATUS=0
	if [[ "$JOB_COMPLETE" == "true" ]]; then
		kubectl logs job/seed-admin -n fitness-platform-production --all-containers --timestamps --tail=200 2>&1 || true
	else
		echo "::error::seed-admin job did not complete; collecting diagnostics before cleanup" >&2
		diagnose_job fitness-platform-production seed-admin
		echo "-> Checking whether admin exists using the encrypted-email hash..."
		# shellcheck disable=SC2016
		DB_CHECK=$(
			kubectl exec -i -n fitness-platform-production postgres-0 -- sh -c '
			export PGPASSWORD="$(cat /etc/secrets/POSTGRES_PASSWORD)"
			exec psql -U "$(cat /etc/secrets/POSTGRES_USER)" \
				-d "$(cat /etc/secrets/POSTGRES_DB)" \
				-v ON_ERROR_STOP=1 -v email="$1" -At
		' sh "$SEED_ADMIN_EMAIL" 2>&1 <<'SQL'
		SELECT EXISTS (
			SELECT 1 FROM users
			WHERE email_hash = encode(convert_to(btrim(:'email'), 'UTF8'), 'hex')
		);
SQL
		) || {
			echo "::error::Could not verify whether the admin user exists: $DB_CHECK" >&2
			SEED_STATUS=1
		}
		if [[ "$DB_CHECK" == "t" ]]; then
			echo "✅ Admin user exists in database (idempotent check passed)"
		else
			echo "::error::seed-admin job failed and admin user was not found in database" >&2
			SEED_STATUS=1
		fi
	fi
	echo "-> Cleaning up completed job..."
	kubectl delete job seed-admin -n fitness-platform-production --ignore-not-found=true 2>&1 || true
	echo "✅ seed-admin step completed"
	return "$SEED_STATUS"
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
	echo "Waiting for all application deployments to be ready..."
	for deploy in user-service biometric-service training-service gateway classifier ml-generator device-aggregator data-processor open-wearables open-wearables-frontend; do
		if ! kubectl rollout status deployment/"$deploy" -n fitness-platform-production --timeout=180s; then
			echo "::error::Deployment $deploy did not become ready" >&2
			kubectl get pods -n fitness-platform-production -l "app=$deploy" -o wide >&2 || true
			kubectl describe deployment "$deploy" -n fitness-platform-production >&2 || true
			return 1
		fi
	done
	if ! kubectl rollout status deployment/ingress-nginx-controller -n ingress-nginx --timeout=120s; then
		echo "::error::ingress-nginx controller did not become ready" >&2
		kubectl get pods -n ingress-nginx -o wide >&2 || true
		return 1
	fi
	echo "✅ All application deployments are ready"
}

diagnose_unready_pods() {
	local namespace="$1"
	local pods pod phase ready container_names restart_status container restart_count state_summary

	echo "=== Resource status in namespace $namespace ==="
	kubectl get deploy,statefulset,job,pods -n "$namespace" -o wide 2>&1 || true
	pods=$(kubectl get pods -n "$namespace" -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)

	while IFS= read -r pod; do
		[[ -z "$pod" ]] && continue
		phase=$(kubectl get pod "$pod" -n "$namespace" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
		[[ "$phase" == "Succeeded" ]] && continue
		ready=$(kubectl get pod "$pod" -n "$namespace" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "Unknown")
		if [[ "$ready" == "True" && "$phase" != "Failed" ]]; then
			continue
		fi

		echo "=== Pod diagnostics: $namespace/$pod (phase=$phase, ready=$ready) ==="
		kubectl describe pod "$pod" -n "$namespace" 2>&1 || true
		container_names=$(kubectl get pod "$pod" -n "$namespace" -o jsonpath='{range .spec.initContainers[*]}{.name}{"\n"}{end}{range .spec.containers[*]}{.name}{"\n"}{end}' 2>/dev/null || true)
		restart_status=$(kubectl get pod "$pod" -n "$namespace" -o jsonpath='{range .status.initContainerStatuses[*]}{.name}={.restartCount}{"\n"}{end}{range .status.containerStatuses[*]}{.name}={.restartCount}{"\n"}{end}' 2>/dev/null || true)
		state_summary=$(kubectl get pod "$pod" -n "$namespace" -o jsonpath='{range .status.initContainerStatuses[*]}init/{.name}: waiting={.state.waiting.reason} terminated={.state.terminated.reason} last={.lastState.terminated.reason} exit={.lastState.terminated.exitCode} message={.lastState.terminated.message} restarts={.restartCount}{"\n"}{end}{range .status.containerStatuses[*]}{.name}: waiting={.state.waiting.reason} terminated={.state.terminated.reason} last={.lastState.terminated.reason} exit={.lastState.terminated.exitCode} message={.lastState.terminated.message} restarts={.restartCount}{"\n"}{end}' 2>/dev/null || true)
		echo "--- Container states ---"
		if [[ -n "$state_summary" ]]; then
			printf '%s\n' "$state_summary"
		else
			echo "No container status has been reported yet"
		fi
		while IFS= read -r container; do
			[[ -z "$container" ]] && continue
			echo "--- Current logs: $container ---"
			kubectl logs "$pod" -n "$namespace" -c "$container" --timestamps --tail=150 2>&1 || true
			restart_count=$(printf '%s\n' "$restart_status" | awk -F= -v name="$container" '$1 == name {print $2; exit}')
			if [[ "$restart_count" =~ ^[0-9]+$ ]] && ((restart_count > 0)); then
				echo "--- Previous logs: $container (restart count: $restart_count) ---"
				kubectl logs "$pod" -n "$namespace" -c "$container" --previous --timestamps --tail=150 2>&1 || true
			fi
		done <<<"$container_names"
		echo "--- Pod events ---"
		kubectl get events -n "$namespace" --field-selector "involvedObject.name=$pod" --sort-by=.lastTimestamp 2>&1 || true
	done <<<"$pods"

	echo "=== Recent events in namespace $namespace ==="
	kubectl get events -n "$namespace" --sort-by=.lastTimestamp 2>&1 | tail -50 || true
}

run_open_wearables_db_job() {
	kubectl delete job create-open-wearables-db -n fitness-platform-production --ignore-not-found=true --wait=true
	kubectl apply -f configs/k8s/base/jobs/create-open-wearables-db.yaml -n fitness-platform-production
	echo "Waiting for Open Wearables database job..."
	if ! kubectl wait --for=condition=complete job/create-open-wearables-db -n fitness-platform-production --timeout=180s; then
		echo "::error::Open Wearables database job failed; collecting diagnostics" >&2
		diagnose_job fitness-platform-production create-open-wearables-db
		return 1
	fi
	kubectl logs job/create-open-wearables-db -n fitness-platform-production --all-containers --timestamps --tail=100 2>&1 || true
}

open_wearables_has_revision_drift() {
	local logs
	logs=$(kubectl logs -n fitness-platform-production -l app=open-wearables --all-containers --tail=200 2>/dev/null || true)
	grep -q "Can't locate revision identified by" <<<"$logs"
}

reset_open_wearables_database() {
	# The drop itself retries inside the pod, but only start once postgres is
	# actually Ready to avoid spending most of the job budget on a crash-looping
	# or not-yet-scheduled database pod.
	kubectl wait --for=condition=Ready pod -l app=postgres -n fitness-platform-production --timeout=120s 2>/dev/null || true
	kubectl delete job reset-open-wearables-db -n fitness-platform-production --ignore-not-found=true --wait=true
	kubectl apply -f configs/k8s/base/jobs/reset-open-wearables-db.yaml -n fitness-platform-production
	if ! kubectl wait --for=condition=complete job/reset-open-wearables-db -n fitness-platform-production --timeout=240s; then
		echo "::error::Failed to reset Open Wearables database; collecting diagnostics" >&2
		diagnose_job fitness-platform-production reset-open-wearables-db
		return 1
	fi
	kubectl logs job/reset-open-wearables-db -n fitness-platform-production --all-containers --timestamps --tail=100 2>&1 || true
	kubectl delete job reset-open-wearables-db -n fitness-platform-production --ignore-not-found=true 2>&1 || true
}

heal_open_wearables_drift() {
	echo "::warning::Open Wearables database has an alembic revision unknown to the pinned image; recreating the database" >&2
	reset_open_wearables_database || return 1
	run_open_wearables_db_job || return 1
	kubectl rollout restart deployment/open-wearables -n fitness-platform-production
	return 0
}

deploy_open_wearables() {
	echo "=== Deploying Open Wearables ==="
	run_open_wearables_db_job || return 1
	# Do NOT re-apply the whole production overlay here: apply_manifests has
	# already done it. Re-applying reset the `kubectl.kubernetes.io/restartedAt`
	# annotation added by restart_app_deployments, which triggered a second
	# rollout of gateway/open-wearables and another pile of ReplicaSets.
	echo "Waiting for Open Wearables deployments to be ready..."
	# The pinned upstream image may not ship every alembic revision recorded in
	# the existing database (e.g. after pinning to an older tag). In that case
	# alembic aborts with "Can't locate revision identified by <id>" and the
	# backend crash-loops. Detect it early instead of waiting out the full
	# 10-minute rollout timeout, then recreate the database and retry once.
	local healed=1 ready
	for _ in {1..12}; do
		# Check readiness first: an old terminating pod may still log the drift
		# error after the database was already healed, and a false positive
		# would drop the database again on every deploy.
		ready=$(kubectl get deployment open-wearables -n fitness-platform-production -o jsonpath='{.status.readyReplicas}' 2>/dev/null || true)
		if [[ "${ready:-0}" -ge 1 ]]; then
			break
		fi
		if open_wearables_has_revision_drift; then
			if ! heal_open_wearables_drift; then
				diagnose_unready_pods fitness-platform-production
				return 1
			fi
			healed=0
			break
		fi
		sleep 10
	done
	# Use rollout status, not a pod-label wait: during a rollout the old pod is
	# terminating and may never become Ready, which made `kubectl wait` time out
	# even though the new pod became Ready moments later. The backend also runs
	# migrations on start, so allow 10 minutes.
	if ! kubectl rollout status deployment/open-wearables -n fitness-platform-production --timeout=600s; then
		# A slow first start may hide the drift from the fast path above.
		if [[ "$healed" -ne 0 ]] && open_wearables_has_revision_drift && heal_open_wearables_drift; then
			if ! kubectl rollout status deployment/open-wearables -n fitness-platform-production --timeout=600s; then
				echo "::error::Open Wearables backend not ready after database reset; collecting pod diagnostics" >&2
				diagnose_unready_pods fitness-platform-production
				return 1
			fi
		else
			echo "::error::Open Wearables backend not ready; collecting pod diagnostics" >&2
			diagnose_unready_pods fitness-platform-production
			return 1
		fi
	fi
	if ! kubectl rollout status deployment/open-wearables-frontend -n fitness-platform-production --timeout=600s; then
		echo "::error::Open Wearables frontend not ready; collecting pod diagnostics" >&2
		diagnose_unready_pods fitness-platform-production
		return 1
	fi
	echo "✅ Open Wearables deployed"
}

collect_cluster_diagnostics() {
	local namespace="fitness-platform-production"
	local workload logs dns_errors

	echo "::group::Kubernetes nodes and CoreDNS"
	kubectl get nodes -o wide 2>&1 || true
	kubectl -n kube-system get pods -o wide 2>&1 || true
	kubectl -n kube-system get pods -l k8s-app=kube-dns -o wide 2>&1 || true
	kubectl -n kube-system describe pods -l k8s-app=kube-dns 2>&1 || true
	kubectl -n kube-system get service kube-dns -o yaml 2>&1 || true
	kubectl -n kube-system get endpoints kube-dns -o yaml 2>&1 || true
	kubectl -n kube-system get endpointslices.discovery.k8s.io \
		-l kubernetes.io/service-name=kube-dns -o wide 2>&1 || true
	kubectl -n kube-system get configmap coredns -o yaml 2>&1 || true
	echo "--- CoreDNS current logs (last 30 minutes) ---"
	kubectl -n kube-system logs -l k8s-app=kube-dns --all-containers=true \
		--prefix --timestamps --since=30m --tail=200 2>&1 || true
	echo "--- CoreDNS previous-container logs ---"
	kubectl -n kube-system logs -l k8s-app=kube-dns --all-containers=true \
		--prefix --timestamps --previous --tail=200 2>&1 || true
	kubectl -n kube-system get events --field-selector=type=Warning \
		--sort-by=.metadata.creationTimestamp 2>&1 | tail -n 100 || true
	echo "::endgroup::"

	echo "::group::cert-manager, DuckDNS webhook, and ACME resources"
	kubectl -n cert-manager get pods,deployments,services,endpoints -o wide 2>&1 || true
	kubectl get certificates,certificaterequests,orders,challenges -A -o wide 2>&1 || true
	kubectl get apiservice v1alpha1.acme.duckdns.org -o yaml 2>&1 || true
	echo "--- cert-manager controller logs (last 30 minutes) ---"
	kubectl -n cert-manager logs deployment/cert-manager --all-containers=true \
		--prefix --timestamps --since=30m --tail=200 2>&1 || true
	echo "--- DuckDNS webhook logs (last 30 minutes) ---"
	kubectl -n cert-manager logs deployment/cert-manager-webhook-duckdns \
		--all-containers=true --prefix --timestamps --since=30m --tail=200 2>&1 || true
	kubectl -n cert-manager get events --field-selector=type=Warning \
		--sort-by=.metadata.creationTimestamp 2>&1 | tail -n 100 || true
	echo "::endgroup::"

	echo "::group::Recent application DNS errors"
	kubectl -n "$namespace" get pods -o wide 2>&1 || true
	for workload in user-service training-service biometric-service gateway device-aggregator classifier; do
		echo "--- DNS-related log lines: deployment/$workload (current, last 30 minutes) ---"
		logs=$(kubectl -n "$namespace" logs "deployment/$workload" \
			--all-containers=true --prefix --timestamps --since=30m --tail=500 2>&1 || true)
		dns_errors=$(printf '%s\n' "$logs" |
			grep -Ei '(^|[^[:alpha:]])(dns|lookup|no such host|connection refused|i/o timeout|temporary failure in name resolution)([^[:alpha:]]|$)|:53' ||
			true)
		if [[ -n "$dns_errors" ]]; then
			printf '%s\n' "$dns_errors"
		else
			echo "No DNS-related errors found in available current logs."
		fi

		echo "--- DNS-related log lines: deployment/$workload (previous container) ---"
		logs=$(kubectl -n "$namespace" logs "deployment/$workload" \
			--all-containers=true --prefix --timestamps --previous --tail=500 2>&1 || true)
		dns_errors=$(printf '%s\n' "$logs" |
			grep -Ei '(^|[^[:alpha:]])(dns|lookup|no such host|connection refused|i/o timeout|temporary failure in name resolution)([^[:alpha:]]|$)|:53' ||
			true)
		if [[ -n "$dns_errors" ]]; then
			printf '%s\n' "$dns_errors"
		else
			echo "No DNS-related errors found in available previous-container logs."
		fi
	done
	echo "::endgroup::"
}

collect_vps_diagnostics() {
	local remote_command

	if [[ -z "${VPS_HOST:-}" || -z "${VPS_USER:-}" ]]; then
		echo "::warning::Skipping VPS diagnostics because VPS_HOST or VPS_USER is unavailable"
		return 0
	fi

	remote_command=$(
		cat <<'REMOTE'
echo '=== k3s service state ==='
sudo systemctl show k3s --property=ActiveState,SubState,MainPID --no-pager 2>&1 || true
echo '=== k3s version ==='
sudo k3s --version 2>&1 || true
echo '=== VPS resolver configuration ==='
cat /etc/resolv.conf 2>&1 || true
echo '=== K3s pod resolver configuration ==='
cat /etc/rancher/k3s/resolv.conf 2>&1 || true
echo '=== DNS lookups from the VPS ==='
for domain in ghcr.io registry-1.docker.io acme-v02.api.letsencrypt.org; do
	echo "--- $domain ---"
	getent ahostsv4 "$domain" 2>&1 | head -n 5 || true
done
echo '=== Recent k3s networking and DNS messages ==='
sudo journalctl -u k3s --since '30 minutes ago' --no-pager -n 500 2>&1 |
	grep -Ei 'coredns|dns|cni|flannel|kube-proxy|iptables|nft|network|no route|refused|timeout' |
	tail -n 200 || true
echo '=== DuckDNS webhook image in k3s containerd ==='
sudo k3s ctr -n k8s.io images ls 2>&1 |
	grep -F 'cert-manager-webhook-duckdns' || true
REMOTE
	)

	echo "::group::VPS k3s, resolver, and containerd diagnostics"
	if ! SSH_RETRY_MAX_ATTEMPTS=1 SSH_RETRY_TIMEOUT=90 \
		bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "$remote_command" 2>&1; then
		echo "::warning::Could not collect VPS diagnostics over SSH"
	fi
	echo "::endgroup::"
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
	diagnose_unready_pods fitness-platform-production
	diagnose_unready_pods ingress-nginx
	collect_cluster_diagnostics
	collect_vps_diagnostics
}

check_logs() {
	echo "Waiting for pods to stabilize..."
	sleep 30
	echo "Checking pod status..."
	kubectl get pods -n fitness-platform-production -o wide
	echo "Checking logs for critical errors..." >&2
	ERROR_FOUND=0
	UNREADY_FOUND=0
	for dep in user-service training-service rabbitmq gateway; do
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
	for app in user-service training-service biometric-service gateway device-aggregator classifier valkey ml-generator open-wearables open-wearables-frontend; do
		READY_PODS=$(kubectl get pods -l "app=$app" -n fitness-platform-production -o jsonpath='{range .items[*]}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}' 2>/dev/null | grep -c '^True$' || true)
		if [[ "$READY_PODS" -eq 0 ]]; then
			echo "::warning::No Ready pod found for $app"
			ERROR_FOUND=1
			UNREADY_FOUND=1
		fi
	done
	READY_INGRESS_PODS=$(kubectl get pods -l app.kubernetes.io/name=ingress-nginx -n ingress-nginx -o jsonpath='{range .items[*]}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}' 2>/dev/null | grep -c '^True$' || true)
	if [[ "$READY_INGRESS_PODS" -eq 0 ]]; then
		echo "::warning::No Ready ingress-nginx pod found"
		ERROR_FOUND=1
		UNREADY_FOUND=1
	fi
	if [[ $ERROR_FOUND -eq 1 ]]; then
		echo "::warning::Critical errors detected during readiness checks" >&2
	else
		echo "✅ No critical errors found, all services are healthy" >&2
	fi
	# ELK/monitoring are not critical for the API, but a CrashLoop/Init:Error
	# here means the deploy left the observability stack broken. Surface it.
	for ns in elk monitoring; do
		BAD_PODS=$(kubectl get pods -n "$ns" -o jsonpath='{range .items[*]}{.metadata.name}{" phase="}{.status.phase}{" ready="}{.status.containerStatuses[0].ready}{"\n"}{end}' 2>/dev/null | grep -Ev 'phase=Running ready=true|phase=Succeeded' || true)
		if [[ -n "$BAD_PODS" ]]; then
			echo "::warning::Unhealthy pods in $ns:" >&2
			echo "$BAD_PODS" >&2
			ERROR_FOUND=1
		fi
	done
	if [[ "$UNREADY_FOUND" -eq 1 ]]; then
		echo "::error::Production services are not Ready; stopping deployment before external tests" >&2
		return 1
	fi
}

verify_tls() {
	local secret="fittpulse-duckdns-tls"
	local domain="${FITPULSE_DOMAIN:-fittpulse.duckdns.org}"
	if ! kubectl wait --for=condition=ready certificate/"$secret" \
		-n fitness-platform-production --timeout=300s; then
		echo "::error::Ingress TLS certificate $secret is not Ready" >&2
		kubectl describe certificate/"$secret" -n fitness-platform-production >&2 || true
		kubectl get order,challenge -n fitness-platform-production >&2 || true
		kubectl describe challenge -n fitness-platform-production >&2 || true
		kubectl get pods -n fitness-platform-production -l acme.cert-manager.io/http01-solver=true -o wide >&2 || true
		kubectl describe networkpolicy -n fitness-platform-production allow-ingress-nginx-to-acme-solver >&2 || true
		return 1
	fi
	echo "✅ Ingress TLS certificate $secret is Ready"
	if ! curl -fsS --max-time 15 -o /dev/null "https://${domain}/health"; then
		echo "::error::Public HTTPS endpoint does not serve the trusted certificate" >&2
		curl -v --max-time 15 "https://${domain}/health" 2>&1 | sed -n '1,40p' >&2 || true
		return 1
	fi
	echo "✅ Public HTTPS endpoint serves the trusted Let's Encrypt certificate"
}
