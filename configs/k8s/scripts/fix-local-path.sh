#!/bin/bash
set -euo pipefail

echo '-> Deploying local-path-provisioner from local manifest...'
k3s kubectl apply -f /tmp/local-path-provisioner.yaml

echo '-> Waiting for local-path-provisioner pod to be ready...'
for i in $(seq 1 60); do
	if k3s kubectl get pods -n local-path-storage -l app=local-path-provisioner --no-headers 2>/dev/null | grep -q Running; then
		echo "local-path provisioner is Running after $((i * 2))s"
		break
	fi

	if [ $((i % 10)) -eq 0 ]; then
		echo "Still waiting... ($((i * 2))s)"
		k3s kubectl get pods -n local-path-storage -l app=local-path-provisioner || true
	fi

	if [ "$i" -eq 60 ]; then
		echo "local-path provisioner not ready after 120s. Debugging info:"
		k3s kubectl get pods -n local-path-storage -o wide || true
		echo "--- Describe ---"
		k3s kubectl describe pods -n local-path-storage -l app=local-path-provisioner || true
		echo "--- Logs ---"
		k3s kubectl logs -n local-path-storage -l app=local-path-provisioner --tail=50 || true
		exit 1
	fi
	sleep 2
done

echo '-> Verifying StorageClass local-path exists and has WaitForFirstConsumer mode...'
for i in $(seq 1 10); do
	sc_name="local-path"
	if ! k3s kubectl get storageclass "$sc_name" >/dev/null 2>&1; then
		echo "StorageClass $sc_name not found"
		k3s kubectl get storageclass -o wide || true
		exit 1
	fi
	mode=$(k3s kubectl get storageclass "$sc_name" -o jsonpath='{.volumeBindingMode}' 2>/dev/null || echo "")
	if [ "$mode" != "WaitForFirstConsumer" ]; then
		echo "StorageClass $sc_name found but wrong volumeBindingMode: ${mode:-<empty>}"
		k3s kubectl get storageclass "$sc_name" -o yaml || k3s kubectl get storageclass -o wide || true
		exit 1
	fi
	echo "local-path StorageClass is WaitForFirstConsumer"
	break
	sleep 3
done

echo '-> Final StorageClasses:'
k3s kubectl get storageclass