#!/bin/bash
set -euo pipefail

echo '-> Deploying local-path-provisioner from local manifest...'
k3s kubectl apply -f /tmp/local-path-provisioner.yaml
sleep 5

echo '-> Configuring StorageClass local-path with Immediate binding mode...'
# Патчим ConfigMap, чтобы provisioner знал про Immediate режим при пересоздании
k3s kubectl patch configmap local-path-config -n local-path-storage \
    --type='json' \
    -p='[{"op":"replace","path":"/data/STORAGECLASS_EXTRA_PARAMS","value":"{\"volumeBindingMode\":\"Immediate\"}"}]' 2>/dev/null || \
k3s kubectl patch configmap local-path-config -n local-path-storage \
    --type='json' \
    -p='[{"op":"add","path":"/data/STORAGECLASS_EXTRA_PARAMS","value":"{\"volumeBindingMode\":\"Immediate\"}"}]' || true

echo '-> Deleting existing StorageClass to force recreation with Immediate mode...'
k3s kubectl delete storageclass local-path --ignore-not-found=true --timeout=30s || true
sleep 3

echo '-> Restarting provisioner to apply ConfigMap changes...'
# Принудительно удаляем pod, чтобы он перезапустился и попытался пересоздать StorageClass
k3s kubectl delete pod -n local-path-storage -l app=local-path-provisioner --ignore-not-found=true --force --grace-period=0 || true
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
        exit 1
    fi
    sleep 2
done

echo '-> Waiting for StorageClass to be recreated with Immediate mode...'
for i in $(seq 1 10); do
    MODE=$(k3s kubectl get storageclass local-path -o jsonpath='{.spec.volumeBindingMode}' 2>/dev/null || echo 'NotFound')
    if [ "$MODE" = "Immediate" ]; then
        echo "local-path StorageClass is Immediate after ${i}x3s"
        break
    fi
    sleep 3
done

FINAL_MODE=$(k3s kubectl get storageclass local-path -o jsonpath='{.spec.volumeBindingMode}' 2>/dev/null || echo 'NotFound')

# Fallback: если provisioner не пересоздал StorageClass (частая ситуация в k3s), создаем его явно
if [ "$FINAL_MODE" != "Immediate" ]; then
    echo "-> Provisioner did not recreate StorageClass with Immediate mode (current: $FINAL_MODE). Creating it manually..."
    cat <<EOF | k3s kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: local-path
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: rancher.io/local-path
volumeBindingMode: Immediate
reclaimPolicy: Delete
EOF
fi

echo "-> Final local-path StorageClass binding mode: $(k3s kubectl get storageclass local-path -o jsonpath='{.spec.volumeBindingMode}' 2>/dev/null || echo 'NotFound')"
echo '-> Final StorageClasses:'
k3s kubectl get storageclass
