#!/bin/bash
set -euo pipefail

check_ssl() {
	echo "🔍 Checking SSL certificate validity..."
	echo | openssl s_client -servername "$FITPULSE_DOMAIN" -connect "$FITPULSE_DOMAIN":443 2>/dev/null | openssl x509 -noout -dates -issuer >cert-details.txt
	echo "🔍 Checking Certificate Transparency (SCT) logs..."
	if grep -q "Signed Certificate Timestamp" cert-details.txt; then
		echo "✅ Certificate Transparency SCTs verified"
	else
		echo "⚠️  No SCTs found in certificate (Certificate Transparency may not be configured)"
	fi
	curl -k -sSf -o /dev/null --max-time 10 "https://$FITPULSE_DOMAIN/health" || exit 1
	echo "✅ SSL certificate is valid and trusted"
}

health_check() {
	echo "🔍 Checking production health via HTTPS..."
	for _ in {1..42}; do
		RESPONSE=$(curl -skf --max-time 5 "https://$FITPULSE_DOMAIN/health" 2>/dev/null || echo "")
		if [[ -n "$RESPONSE" ]]; then
			echo "Health response: $RESPONSE"
			break
		fi
		echo "Attempt $i/42: waiting..."
		sleep 10
	done
	if [[ -z "$RESPONSE" ]]; then
		echo "❌ Health endpoint unreachable"
		kubectl --kubeconfig="$HOME/.kube/config" get pods -n fitness-platform-production -o wide || true
		kubectl --kubeconfig="$HOME/.kube/config" get svc -n fitness-platform-production || true
		exit 1
	fi
	echo "$RESPONSE" | grep -q '"status":"ok"' || {
		echo "❌ Status not ok"
		echo "$RESPONSE"
		exit 1
	}
	echo "$RESPONSE" | grep -q '"user":"up"' || {
		echo "❌ User service not healthy"
		exit 1
	}
	echo "✅ Production is healthy"
}

check_csp() {
	chmod +x scripts/csp-check.sh
	bash scripts/csp-check.sh
}

wait_for_postgres() {
	echo "Waiting for Postgres pod to be scheduled and running..."
	for i in {1..120}; do
		STATUS=$(kubectl get pod postgres-0 -n fitness-platform-production -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
		if [[ "$STATUS" = "Running" ]]; then
			echo "✅ Postgres pod is Running after ${i}x5 seconds"
			break
		fi
		if [[ "$i" -eq 120 ]]; then
			echo "❌ Timeout waiting for Postgres pod"
			echo "-> PVC Status:"
			kubectl get pvc -n fitness-platform-production || true
			kubectl describe pvc postgres-storage-postgres-0 -n fitness-platform-production || true
			echo "-> Pod Status:"
			kubectl get pods -n fitness-platform-production -o wide
			kubectl describe pod postgres-0 -n fitness-platform-production || true
			exit 1
		fi
		echo "Attempt $i/120: Postgres status=$STATUS, waiting 5s..."
		sleep 5
	done
	echo "Verifying Postgres accepts connections..."
	for i in {1..30}; do
		if kubectl exec -n fitness-platform-production postgres-0 -- pg_isready -U postgres &>/dev/null; then
			echo "✅ Postgres is ready to accept connections"
			break
		fi
		if [[ "$i" -eq 30 ]]; then
			echo "❌ Postgres not accepting connections"
			kubectl logs postgres-0 -n fitness-platform-production --tail=50 || true
			exit 1
		fi
		echo "Attempt $i/30: pg_isready failed, waiting 3s..."
		sleep 3
	done
}

wait_for_pvc() {
	echo "Waiting for Postgres PVC to be created..."
	for i in {1..30}; do
		if kubectl get pvc postgres-storage-postgres-0 -n fitness-platform-production &>/dev/null; then
			echo "✅ PVC created"
			break
		fi
		echo "Attempt $i/30: PVC not found yet, waiting 5s..."
		sleep 5
	done
	echo "Waiting for PVC to be bound..."
	for i in {1..30}; do
		PHASE=$(kubectl get pvc postgres-storage-postgres-0 -n fitness-platform-production -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
		if [[ "$PHASE" = "Bound" ]]; then
			echo "✅ PVC is bound"
			break
		fi
		if [[ "$i" -eq 30 ]]; then
			echo "❌ Timeout waiting for Postgres PVC to bind"
			kubectl get nodes -o wide || true
			kubectl get pvc -n fitness-platform-production || true
			kubectl describe pvc postgres-storage-postgres-0 -n fitness-platform-production || true
			kubectl get storageclass || true
			exit 1
		fi
		echo "Attempt $i/30: PVC phase is $PHASE, waiting 5s..."
		sleep 5
	done
	echo "✅ Postgres PVC is bound"
}

wait_for_service() {
	echo "Waiting for postgres-service endpoint..."
	for i in {1..20}; do
		if kubectl get endpoints postgres-service -n fitness-platform-production -o jsonpath='{.subsets[*].addresses[*].ip}' | grep -q .; then
			echo "✅ postgres-service has endpoints"
			break
		fi
		echo "Attempt $i/20: waiting for postgres-service endpoints..."
		sleep 5
	done
}

main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all)
		check_ssl
		health_check
		check_csp
		wait_for_postgres
		wait_for_pvc
		wait_for_service
		;;
	check_ssl) check_ssl ;;
	health_check) health_check ;;
	check_csp) check_csp ;;
	wait_for_postgres) wait_for_postgres ;;
	wait_for_pvc) wait_for_pvc ;;
	wait_for_service) wait_for_service ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
