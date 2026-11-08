#!/bin/bash
set -euo pipefail

echo '-> Deploying local-path-provisioner from local manifest...'
k3s kubectl apply -f /tmp/local-path-provisioner.yaml

echo '-> Verifying local-path-provisioner image is present in containerd...'
k3s crictl images | grep local-path || echo "WARNING: local-path image NOT found in crictl!"

echo '-> Cleaning stale local-path smoke resources from previous runs...'
for pod in $(k3s kubectl get pods -n local-path-storage -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | grep -E '^(lp-smoke-|ci-lp-smoke-)' || true); do
	k3s kubectl delete pod "$pod" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
done
for pvc in $(k3s kubectl get pvc -n local-path-storage -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | grep -E '^(lp-smoke-|ci-lp-smoke-)' || true); do
	k3s kubectl delete pvc "$pvc" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
done
sleep 5

echo '-> Waiting for local-path-provisioner pod to be Ready...'
if ! k3s kubectl -n local-path-storage wait --for=condition=Ready pod -l app=local-path-provisioner --timeout=180s; then
	echo "❌ local-path provisioner is not Ready after 180s. Debugging info:"
	k3s kubectl get pods -n local-path-storage -o wide || true
	k3s kubectl describe pods -n local-path-storage -l app=local-path-provisioner || true
	k3s kubectl logs -n local-path-storage -l app=local-path-provisioner --tail=100 || true
	k3s kubectl get events -n local-path-storage --sort-by='.lastTimestamp' | tail -50 || true
	exit 1
fi
echo "✅ local-path provisioner is Ready"

# 🔥 КРИТИЧЕСКИЙ ФИКС: Принудительный рестарт для сброса зависших watch-каналов
echo '-> Restarting provisioner deployment to clear potential stale watches...'
k3s kubectl rollout restart deployment/local-path-provisioner -n local-path-storage
k3s kubectl rollout status deployment/local-path-provisioner -n local-path-storage --timeout=120s
echo "✅ Provisioner restarted and ready"

echo '-> Smoke-testing PVC provisioning...'
SMOKE_NAME="lp-smoke-$(date +%s)"
BOUND=false

# Создаём PVC и Pod АТОДАРНО, но без cleanup trap.
# Cleanup будет выполнен явно после проверки.
cat <<EOF | k3s kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ${SMOKE_NAME}
  namespace: local-path-storage
  labels:
    app.kubernetes.io/name: lp-smoke
spec:
  accessModes: ["ReadWriteOnce"]
  resources:
    requests:
      storage: 1Mi
---
apiVersion: v1
kind: Pod
metadata:
  name: ${SMOKE_NAME}-consumer
  namespace: local-path-storage
  labels:
    app.kubernetes.io/name: lp-smoke
spec:
  restartPolicy: Never
  containers:
    - name: consumer
      image: docker.io/rancher/mirrored-library-busybox:1.37.0
      imagePullPolicy: IfNotPresent
      command: ["sleep", "3600"]
      volumeMounts:
        - name: data
          mountPath: /data
  volumes:
    - name: data
      persistentVolumeClaim:
        claimName: ${SMOKE_NAME}
EOF

echo "-> Created PVC '${SMOKE_NAME}' and consumer pod. Waiting for Bound..."

# Ждём до 180 секунд, проверяя phase напрямую
for i in $(seq 1 60); do
	phase=$(k3s kubectl get pvc "${SMOKE_NAME}" -n local-path-storage -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
	if [[ "$phase" == "Bound" ]]; then
		BOUND=true
		echo "✅ Smoke PVC became Bound after $((i * 3))s"
		break
	fi
	if ((i % 10 == 0)); then
		echo "  Still waiting... attempt $i/60, phase=${phase:-NotFound}"
		# Диагностика каждые 30 секунд
		k3s kubectl get pods -n local-path-storage -l app.kubernetes.io/name=lp-smoke -o wide || true
		k3s kubectl get events -n local-path-storage --sort-by='.lastTimestamp' | tail -5 || true
	fi
	sleep 3
done

# Явная очистка — ТОЛЬКО после завершения ожидания
echo "-> Cleaning up smoke resources..."
k3s kubectl delete pod "${SMOKE_NAME}-consumer" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
k3s kubectl delete pvc "${SMOKE_NAME}" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true

if [[ "$BOUND" != true ]]; then
	echo "❌ Smoke PVC did not become Bound within 180s."
	echo "--- Final diagnostics ---"
	k3s kubectl get pvc,pods -n local-path-storage -o wide || true
	k3s kubectl describe pvc "${SMOKE_NAME}" -n local-path-storage || true
	k3s kubectl get events -n local-path-storage --sort-by='.lastTimestamp' | tail -30 || true
	k3s kubectl logs -n local-path-storage -l app=local-path-provisioner --tail=50 || true
	k3s crictl images | grep busybox || echo "BusyBox image NOT found locally"
	exit 1
fi

echo "✅ Smoke PVC bound; local-path provisioning works"

echo '-> Verifying StorageClass local-path exists and has WaitForFirstConsumer mode...'
sc_name="local-path"
if ! k3s kubectl get storageclass "$sc_name" >/dev/null 2>&1; then
	echo "❌ StorageClass $sc_name not found"
	k3s kubectl get storageclass -o wide || true
	exit 1
fi

mode=$(k3s kubectl get storageclass "$sc_name" -o jsonpath='{.volumeBindingMode}' 2>/dev/null || echo "")
if [[ "$mode" != "WaitForFirstConsumer" ]]; then
	echo "❌ StorageClass $sc_name found but wrong volumeBindingMode: ${mode:-<empty>}"
	k3s kubectl get storageclass "$sc_name" -o yaml || true
	exit 1
fi

echo "✅ local-path StorageClass is WaitForFirstConsumer"
echo "✅ local-path-provisioner smoke test completed successfully"
