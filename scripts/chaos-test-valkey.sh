#!/usr/bin/env bash
set -euo pipefail

# Chaos Test: Valkey master failure
# This script simulates a Valkey master failure and tests failover/recovery
#
# Usage: chaos-test-valkey.sh [namespace]
#   namespace - Kubernetes namespace (default: fitness-platform-production)

NAMESPACE="${1:-fitness-platform-production}"
SERVICE_NAME="valkey-service"
TEST_DURATION="${TEST_DURATION:-300}"      # 5 minutes
RECOVERY_TIMEOUT="${RECOVERY_TIMEOUT:-60}" # 60 seconds

echo "=== Chaos Test: Valkey Master Failure ==="
echo "Namespace: $NAMESPACE"
echo "Service: $SERVICE_NAME"
echo "Test duration: ${TEST_DURATION}s"
echo "Recovery timeout: ${RECOVERY_TIMEOUT}s"
echo ""

# Check if service exists
if ! kubectl get service "$SERVICE_NAME" -n "$NAMESPACE" >/dev/null 2>&1; then
	echo "ERROR: Service $SERVICE_NAME not found in namespace $NAMESPACE"
	exit 1
fi

# Get initial pod count
INITIAL_PODS=$(kubectl get pods -n "$NAMESPACE" -l app=valkey --no-headers 2>/dev/null | wc -l)
echo "Initial Valkey pods: $INITIAL_PODS"

# Get service endpoint before failure
SERVICE_EP=$(kubectl get service "$SERVICE_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}' 2>/dev/null)
echo "Service ClusterIP: $SERVICE_EP"

# Record start time
START_TIME=$(date +%s)

echo ""
echo "Step 1: Deleting Valkey master pod..."
kubectl delete pod -n "$NAMESPACE" -l app=valkey --grace-period=0 --force

echo ""
echo "Step 2: Waiting for pod to be recreated..."
for i in $(seq 1 "$RECOVERY_TIMEOUT"); do
	PODS=$(kubectl get pods -n "$NAMESPACE" -l app=valkey --no-headers 2>/dev/null | grep -c Running)
	if [ "$PODS" -ge 1 ]; then
		echo "Valkey pod is running again after ${i}s"
		break
	fi
	if [ "$i" -eq "$RECOVERY_TIMEOUT" ]; then
		echo "ERROR: Valkey pod did not recover within ${RECOVERY_TIMEOUT}s"
		exit 1
	fi
	sleep 1
done

# Record end time
END_TIME=$(date +%s)
RECOVERY_TIME=$((END_TIME - START_TIME))

echo ""
echo "Step 3: Verifying service connectivity..."
for i in $(seq 1 10); do
	if kubectl run -i --tty --rm valkey-test --image=valkey:8-alpine \
		-n "$NAMESPACE" --restart=Never --rm -- \
		valkey-cli -h "$SERVICE_NAME" ping >/dev/null 2>&1; then
		echo "Service connectivity verified after ${i}s"
		break
	fi
	if [ "$i" -eq 10 ]; then
		echo "WARNING: Service connectivity not verified within 10s"
	fi
	sleep 1
done

echo ""
echo "Step 4: Checking data persistence..."
kubectl run -i --tty --rm valkey-test --image=valkey:8-alpine \
	-n "$NAMESPACE" --restart=Never --rm -- \
	valkey-cli -h "$SERVICE_NAME" DBSIZE 2>/dev/null || echo "WARNING: Could not verify data"

echo ""
echo "=== Chaos Test Results ==="
echo "Recovery time: ${RECOVERY_TIME}s"
echo "Test duration: ${TEST_DURATION}s"

if [ "$RECOVERY_TIME" -lt "$RECOVERY_TIMEOUT" ]; then
	echo "✅ PASS: Valkey recovered within ${RECOVERY_TIMEOUT}s"
	exit 0
else
	echo "❌ FAIL: Valkey did not recover within ${RECOVERY_TIMEOUT}s"
	exit 1
fi
