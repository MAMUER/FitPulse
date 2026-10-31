#!/bin/bash
set -euo pipefail

echo '-> Deploying local-path-provisioner from local manifest...'
k3s kubectl apply -f /tmp/local-path-provisioner.yaml

echo '-> Verifying local-path-provisioner image is present in containerd...'
k3s crictl images | grep local-path || echo "WARNING: local-path image NOT found in crictl!"

echo '-> Waiting for local-path-provisioner pod to be Ready...'
if ! k3s kubectl -n local-path-storage wait --for=condition=Ready pod -l app=local-path-provisioner --timeout=180s; then
	echo "❌ local-path provisioner is not Ready after 180s. Debugging info:"
	k3s kubectl get pods -n local-path-storage -o wide || true
	echo "- Describe -"
	k3s kubectl describe pods -n local-path-storage -l app=local-path-provisioner || true
	echo "- Logs -"
	k3s kubectl logs -n local-path-storage -l app=local-path-provisioner --tail=100 || true
	echo "- Events -"
	k3s kubectl get events -n local-path-storage --sort-by='.lastTimestamp' | tail -50 || true
	echo "- Crictl images check -"
	k3s crictl images | grep local-path || echo "Image NOT found in crictl. Airgap preload may have failed."
	exit 1
fi

echo "✅ local-path provisioner is Ready"

echo '-> Smoke-testing PVC provisioning (WaitForFirstConsumer requires a consumer pod)...'
SMOKE_NAME="lp-smoke-$(date +%s)"

cat <<EOF | k3s kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ${SMOKE_NAME}
  namespace: local-path-storage
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
spec:
  restartPolicy: Never
  containers:
    - name: consumer
      image: busybox:1.37.0@sha256:9db7b59979c38555a39def84a31fb98b5296952f9e3afd4f6f11f05b07adfab0
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

cleanup_smoke() {
	k3s kubectl delete pod "${SMOKE_NAME}-consumer" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
	k3s kubectl delete pvc "${SMOKE_NAME}" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
}
trap cleanup_smoke EXIT

if ! k3s kubectl wait pvc "${SMOKE_NAME}" -n local-path-storage --for=condition=Bound --timeout=180s; then
	echo "❌ Smoke PVC did not become Bound. local-path provisioner is not functional."
	k3s kubectl get pvc,pods -n local-path-storage -o wide || true
	echo "- Describe PVC -"
	k3s kubectl describe pvc "${SMOKE_NAME}" -n local-path-storage || true
	echo "- Events -"
	k3s kubectl get events -n local-path-storage --sort-by='.lastTimestamp' | tail -80 || true
	echo "- Provisioner logs -"
	k3s kubectl logs -n local-path-storage -l app=local-path-provisioner --tail=100 || true
	exit 1
fi

echo "✅ Smoke PVC bound; local-path provisioning works"
cleanup_smoke
trap - EXIT

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
	k3s kubectl get storageclass "$sc_name" -o yaml || k3s kubectl get storageclass -o wide || true
	exit 1
fi

echo "✅ local-path StorageClass is WaitForFirstConsumer"
echo "✅ local-path-provisioner smoke test completed successfully"
