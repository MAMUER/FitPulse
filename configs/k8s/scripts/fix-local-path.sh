#!/bin/bash
set -euo pipefail

echo '-> Deploying local-path-provisioner from local manifest...'
k3s kubectl apply -f /tmp/local-path-provisioner.yaml
sleep 5

echo '-> Configuring StorageClass local-path with WaitForFirstConsumer binding mode...'
# Патчим ConfigMap, чтобы provisioner знал про WaitForFirstConsumer режим при пересоздании
k3s kubectl patch configmap local-path-config -n local-path-storage \
	--type='json' \
	-p='[{"op":"replace","path":"/data/STORAGECLASS_EXTRA_PARAMS","value":"{\"volumeBindingMode\":\"WaitForFirstConsumer\"}"}]' 2>/dev/null ||
	k3s kubectl patch configmap local-path-config -n local-path-storage \
		--type='json' \
		-p='[{"op":"add","path":"/data/STORAGECLASS_EXTRA_PARAMS","value":"{\"volumeBindingMode\":\"WaitForFirstConsumer\"}"}]' || true

echo '-> Deleting existing StorageClass to force recreation with WaitForFirstConsumer mode...'
k3s kubectl delete storageclass local-path --ignore-not-found=true --timeout=30s || true
sleep 3

echo '-> Restarting provisioner to apply ConfigMap changes...'
# Принудительно удаляем pod, чтобы он перезапустился и попытался пересоздать StorageClass
k3s kubectl delete pod -n local-path-storage -l app=local-path-provisioner --ignore-not-found=true --force --grace-period=0 || true
sleep 10

echo '-> Waiting for StorageClass to be recreated with WaitForFirstConsumer mode...'
for i in $(seq 1 10); do
	MODE=$(k3s kubectl get storageclass local-path -o jsonpath='{.spec.volumeBindingMode}' 2>/dev/null || echo 'NotFound')
	if [ "$MODE" = "WaitForFirstConsumer" ]; then
		echo "local-path StorageClass is WaitForFirstConsumer after ${i}x3s"
		break
	fi
	sleep 3
done

FINAL_MODE=$(k3s kubectl get storageclass local-path -o jsonpath='{.spec.volumeBindingMode}' 2>/dev/null || echo 'NotFound')

# Fallback: если provisioner не пересоздал StorageClass (частая ситуация в k3s), создаем его явно
if [ "$FINAL_MODE" != "WaitForFirstConsumer" ]; then
	echo "-> Provisioner did not recreate StorageClass with WaitForFirstConsumer mode (current: $FINAL_MODE). Creating it manually..."
	cat <<EOF | k3s kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: local-path
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: rancher.io/local-path
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Delete
EOF
fi

echo "-> Final local-path StorageClass binding mode: $(k3s kubectl get storageclass local-path -o jsonpath='{.spec.volumeBindingMode}' 2>/dev/null || echo 'NotFound')"
echo '-> Final StorageClasses:'
k3s kubectl get storageclass
