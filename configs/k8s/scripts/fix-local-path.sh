#!/bin/bash
set -euo pipefail

echo '-> Deploying local-path-provisioner from local manifest...'
k3s kubectl apply -f /tmp/local-path-provisioner.yaml

echo '-> Waiting for local-path-provisioner pod to be ready...'
for i in $(seq 1 30); do
	if k3s kubectl get pods -n local-path-storage -l app=local-path-provisioner --no-headers 2>/dev/null | grep -q Running; then
    	echo "✅ local-path provisioner is Running after ${i}x2s"
    	break
  	fi
  	if [ "$i" -eq 30 ]; then
    	echo "❌ local-path provisioner not ready"
    	k3s kubectl get pods -n local-path-storage
    	k3s kubectl describe pods -n local-path-storage -l app=local-path-provisioner
    	exit 1
  	fi
sleep 2
done

echo '-> Verifying StorageClass local-path exists and has WaitForFirstConsumer mode...'
for i in $(seq 1 10); do
  	MODE=$(k3s kubectl get storageclass local-path -o jsonpath='{.spec.volumeBindingMode}' 2>/dev/null || echo 'NotFound')
  	if [ "$MODE" = "WaitForFirstConsumer" ]; then
    	echo "✅ local-path StorageClass is WaitForFirstConsumer"
    	break
  	fi
  	if [ "$i" -eq 10 ]; then
    	echo "❌ StorageClass local-path not found or wrong mode: $MODE"
    	k3s kubectl get storageclass
    	exit 1
	fi
  	sleep 3
done

echo '-> Final StorageClasses:'
k3s kubectl get storageclass