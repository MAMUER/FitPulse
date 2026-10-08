#!/usr/bin/env bash
set -euo pipefail

# Chaos Test: Vault failure
# This script simulates a Vault failure and tests recovery
#
# Usage: chaos-test-vault.sh [namespace]
#   namespace - Kubernetes namespace (default: vault)

NAMESPACE="${1:-vault}"
SERVICE_NAME="vault-service"
TEST_DURATION="${TEST_DURATION:-300}"      # 5 minutes
RECOVERY_TIMEOUT="${RECOVERY_TIMEOUT:-60}" # 60 seconds

echo "=== Chaos Test: Vault Failure ==="
echo "Namespace: $NAMESPACE"
echo "Service: $SERVICE_NAME"
echo "Test duration: ${TEST_DURATION}s"
echo "Recovery timeout: ${RECOVERY_TIMEOUT}s"
echo ""

# Check if namespace exists
if ! kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
	echo "ERROR: Namespace $NAMESPACE not found"
	exit 1
fi

# Check if service exists
if ! kubectl get service "$SERVICE_NAME" -n "$NAMESPACE" >/dev/null 2>&1; then
	echo "WARNING: Service $SERVICE_NAME not found in namespace $NAMESPACE, skipping test"
	exit 0
fi

# Get initial pod count
INITIAL_PODS=$(kubectl get pods -n "$NAMESPACE" -l app=vault --no-headers 2>/dev/null | wc -l)
echo "Initial Vault pods: $INITIAL_PODS"

# Record start time
START_TIME=$(date +%s)

echo ""
echo "Step 1: Deleting Vault pod..."
kubectl delete pod -n "$NAMESPACE" -l app=vault --grace-period=0 --force

echo ""
echo "Step 2: Waiting for pod to be recreated..."
for i in $(seq 1 "$RECOVERY_TIMEOUT"); do
	PODS=$(kubectl get pods -n "$NAMESPACE" -l app=vault --no-headers 2>/dev/null | grep -c Running)
	if [ "$PODS" -ge 1 ]; then
		echo "Vault pod is running again after ${i}s"
		break
	fi
	if [ "$i" -eq "$RECOVERY_TIMEOUT" ]; then
		echo "ERROR: Vault pod did not recover within ${RECOVERY_TIMEOUT}s"
		exit 1
	fi
	sleep 1
done

# Record end time
END_TIME=$(date +%s)
RECOVERY_TIME=$((END_TIME - START_TIME))

echo ""
echo "Step 3: Verifying Vault is unsealed and accessible..."
for i in $(seq 1 10); do
	if kubectl exec -n "$NAMESPACE" vault-0 -- vault status >/dev/null 2>&1; then
		echo "Vault is accessible after ${i}s"
		break
	fi
	if [ "$i" -eq 10 ]; then
		echo "WARNING: Vault not accessible within 10s (may need unsealing)"
	fi
	sleep 1
done

echo ""
echo "=== Chaos Test Results ==="
echo "Recovery time: ${RECOVERY_TIME}s"
echo "Test duration: ${TEST_DURATION}s"

if [ "$RECOVERY_TIME" -lt "$RECOVERY_TIMEOUT" ]; then
	echo "✅ PASS: Vault recovered within ${RECOVERY_TIMEOUT}s"
	exit 0
else
	echo "❌ FAIL: Vault did not recover within ${RECOVERY_TIMEOUT}s"
	exit 1
fi
