#!/bin/bash
set -euo pipefail

echo '-> Deploying local-path-provisioner from local manifest...'
k3s kubectl apply -f /tmp/local-path-provisioner.yaml
sleep 10

echo '-> Waiting for provisioner to be ready...'
for i in $(seq 1 30); do
	if k3s kubectl get pods -n local-path-storage -l app=local-path-provisioner --no-headers 2>/dev/null | grep -q Running; then
		echo "local-path provisioner is Running after ${i}x2s"
		break
	fi
	if [ "$i" -eq 30 ]; then
		echo "local-path provisioner not ready"
		k3s kubectl get pods -n local-path-storage
		PROVISIONER_POD=$(k3s kubectl get pods -n local-path-storage -l app=local-path-provisioner -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo '')
		if [ -n "$PROVISIONER_POD" ]; then
			echo "-> Pod logs:"
			k3s kubectl logs -n local-path-storage "$PROVISIONER_POD" --tail=50 2>/dev/null || true
			echo "-> Pod describe:"
			k3s kubectl describe pod -n local-path-storage "$PROVISIONER_POD" 2>/dev/null | tail -40 || true
		fi
		exit 1
	fi
	sleep 2
done

echo '-> Configuring StorageClass local-path with Immediate binding mode...'
k3s kubectl patch configmap local-path-config -n local-path-storage \
	--type='json' \
	-p='[{"op":"replace","path":"/data/STORAGECLASS_EXTRA_PARAMS","value":"{\"volumeBindingMode\":\"Immediate\"}"}]' 2>/dev/null || \
k3s kubectl patch configmap local-path-config -n local-path-storage \
	--type='json' \
	-p='[{"op":"add","path":"/data/STORAGECLASS_EXTRA_PARAMS","value":"{\"volumeBindingMode\":\"Immediate\"}"}]'

echo '-> Restarting provisioner to apply ConfigMap changes...'
k3s kubectl delete pod -n local-path-storage -l app=local-path-provisioner --ignore-not-found=true --timeout=30s
sleep 10

echo '-> Waiting for provisioner to be ready after restart...'
for i in $(seq 1 30); do
	if k3s kubectl get pods -n local-path-storage -l app=local-path-provisioner --no-headers 2>/dev/null | grep -q Running; then
		echo "local-path provisioner is Running after ${i}x2s"
		break
	fi
	if [ "$i" -eq 30 ]; then
		echo "local-path provisioner not ready after restart"
		k3s kubectl get pods -n local-path-storage
		PROVISIONER_POD=$(k3s kubectl get pods -n local-path-storage -l app=local-path-provisioner -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo '')
		if [ -n "$PROVISIONER_POD" ]; then
			echo "-> Pod logs:"
			k3s kubectl logs -n local-path-storage "$PROVISIONER_POD" --tail=50 2>/dev/null || true
		fi
		exit 1
	fi
	sleep 2
done

echo '-> Deleting existing StorageClass to force recreation with Immediate mode...'
k3s kubectl delete storageclass local-path --ignore-not-found=true --timeout=30s || true
sleep 3
if k3s kubectl get storageclass local-path &>/dev/null; then
	echo 'StorageClass still exists after deletion, retrying...'
	sleep 5
	k3s kubectl delete storageclass local-path --ignore-not-found=true --timeout=30s || true
	sleep 3
fi

echo '-> Waiting for StorageClass to be recreated with Immediate mode...'
for i in $(seq 1 30); do
	MODE=$(k3s kubectl get storageclass local-path -o jsonpath='{.spec.volumeBindingMode}' 2>/dev/null || echo 'NotFound')
	if [ "$MODE" = "Immediate" ]; then
		echo "local-path StorageClass is Immediate after ${i}x3s"
		break
	fi
	if [ "$MODE" = "NotFound" ]; then
		sleep 3
		continue
	fi
	echo "Attempt $i/30: mode=$MODE, waiting..."
	sleep 3
done

FINAL_MODE=$(k3s kubectl get storageclass local-path -o jsonpath='{.spec.volumeBindingMode}' 2>/dev/null || echo 'NotFound')
echo "-> local-path StorageClass binding mode: $FINAL_MODE"
if [ "$FINAL_MODE" = "Immediate" ]; then
	echo 'local-path StorageClass is Immediate'
else
	echo "StorageClass mode is '$FINAL_MODE', expected Immediate. Exiting."
	exit 1
fi

echo '-> Final StorageClasses:'
k3s kubectl get storageclass
