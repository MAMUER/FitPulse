#!/bin/bash
set -euo pipefail

configure_swap() {
	echo "🔄 Проверка доступности VPS по SSH (попыток: 3)..."

	local max_attempts=3
	local attempt=1
	local connected=false

	while [[ $attempt -le $max_attempts ]]; do
		echo "➡️ Попытка $attempt из $max_attempts..."
		if timeout 30 ssh -p "${VPS_PORT:-22}" -o BatchMode=yes -o ConnectTimeout=15 -o StrictHostKeyChecking=accept-new "${VPS_USER}@${VPS_HOST}" "echo ok" >/dev/null 2>&1; then
			connected=true
			echo "✅ VPS успешно доступен по SSH!"
			break
		else
			echo "⚠️ Попытка $attempt не удалась. Ждем 15 секунд перед следующей попыткой (возможно, обновляется DuckDNS)..."
			sleep 15
			((attempt++))
		fi
	done

	if [[ "$connected" = false ]]; then
		echo "❌ VPS не доступен по SSH после $max_attempts попыток — прерывание"
		echo "💡 Проверьте: 1) Работает ли ПК, 2) Обновился ли IP в DuckDNS, 3) Проброшен ли порт 22."
		return 1
	fi

	if timeout 10 ssh -p "${VPS_PORT:-22}" -o BatchMode=yes -o ConnectTimeout=10 "${VPS_USER}@${VPS_HOST}" "test -f /tmp/.swap-configured" >/dev/null 2>&1; then
		echo "✅ Swap уже настроен, пропускаем"
		return 0
	fi

	echo "-> Настройка swap на VPS..."
	./scripts/ssh-retry.sh scp configs/k8s/scripts/configure-swap.sh "${VPS_USER}@${VPS_HOST}:/tmp/configure-swap.sh"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo chmod +x /tmp/configure-swap.sh && sudo /tmp/configure-swap.sh && touch /tmp/.swap-configured"
}

check_disk_space() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
DISK_USAGE=$(df / | tail -1 | awk '{print $5}' | sed 's/%//')
echo "Current disk usage: ${DISK_USAGE}%"
if [[ "$DISK_USAGE" -gt 90 ]]; then
	echo "⚠️ WARNING: Disk usage is over 90%!"
	curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
		-H "Content-Type: application/json" \
		-d "{\"chat_id\": \"${TELEGRAM_CHAT_ID}\", \"text\": \"⚠️ VPS Disk usage is ${DISK_USAGE}%!\"}" || true
fi
EOF
}

cleanup_old_k3s() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
set -euo pipefail
echo '-> Performing full k3s cleanup...'
systemctl stop k3s k3s-agent 2>/dev/null || true
pkill -9 -f 'k3s' 2>/dev/null || true
if [ -f /usr/local/bin/k3s-killall.sh ]; then
    echo '-> Running k3s-killall.sh to clean network interfaces...'
    /usr/local/bin/k3s-killall.sh || true
fi
rm -rf /var/lib/rancher/k3s
rm -rf /etc/rancher/k3s
rm -rf /var/lib/cni
rm -rf /var/lib/kubelet
rm -rf /run/k3s
rm -rf /run/flannel
rm -f /usr/local/bin/k3s
rm -f /usr/local/bin/kubectl
rm -f /usr/local/bin/crictl
rm -f /usr/local/bin/ctr
rm -f /etc/systemd/system/k3s.service
rm -f /etc/systemd/system/k3s.service.env
rm -f /etc/systemd/system/k3s-agent.service
rm -f /etc/systemd/system/multi-user.target.wants/k3s.service
rm -f /etc/systemd/system/multi-user.target.wants/k3s-agent.service
systemctl daemon-reload
systemctl reset-failed
rm -rf /opt/local-path-provisioner
echo '✅ Full k3s cleanup completed successfully'
"
}

configure_cpu_governor() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
set -euo pipefail
echo "-> Configuring CPU governor..."
if [[ -d /sys/devices/system/cpu/cpufreq ]]; then
	for gov in /sys/devices/system/cpu/cpufreq/policy*/scaling_governor; do
		[[ -f "$gov" ]] && echo "performance" | sudo tee "$gov" >/dev/null || true
	done
	echo "✅ CPU governor set to performance"
else
	echo "⚠️ CPU frequency scaling not available, skipping governor"
fi
echo "-> Configuring k3s systemd watchdog..."
sudo mkdir -p /etc/systemd/system/k3s.service.d
cat <<EOT | sudo tee /etc/systemd/system/k3s.service.d/99-performance.conf
[Service]
CPUQuota=180%
WatchdogSec=30s
Restart=always
RestartSec=10s
EOT
sudo systemctl daemon-reload
echo "✅ k3s watchdog and CPU quota configured"
EOF
}
configure_disk_encryption() {
	./scripts/ssh-retry.sh scp configs/k8s/scripts/configure-storage-encryption.sh "${VPS_USER}@${VPS_HOST}:/tmp/configure-storage-encryption.sh"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo chmod +x /tmp/configure-storage-encryption.sh && sudo /tmp/configure-storage-encryption.sh --yes || true"
}

fix_dns() {
	echo "-> Uploading DNS fix script to VPS..."
	./scripts/ssh-retry.sh scp configs/k8s/scripts/fix-dns.sh "${VPS_USER}@${VPS_HOST}:/tmp/fix-dns.sh"
	echo "-> Running DNS fix script on VPS..."
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
sudo chmod +x /tmp/fix-dns.sh
sudo /tmp/fix-dns.sh
rm -f /tmp/fix-dns.sh
"
}

prepare_k3s_config() {
	./configs/k8s/scripts/generate-k3s-config.sh "${VPS_HOST}" /tmp/k3s-config-remote.yaml
	if command -v python3 &>/dev/null; then
		python3 -c "import yaml; yaml.safe_load(open('/tmp/k3s-config-remote.yaml'))" || {
			echo "❌ Generated config is invalid YAML"
			cat /tmp/k3s-config-remote.yaml
			exit 1
		}
		echo "✅ Config YAML is valid"
	fi

	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo mkdir -p /etc/rancher/k3s"
	./scripts/ssh-retry.sh scp /tmp/k3s-config-remote.yaml "${VPS_USER}@${VPS_HOST}:/tmp/k3s-config-remote.yaml"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo cp /tmp/k3s-config-remote.yaml /etc/rancher/k3s/config.yaml && sudo chmod 600 /etc/rancher/k3s/config.yaml"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
echo '=== Config written to /etc/rancher/k3s/config.yaml ==='
sudo cat /etc/rancher/k3s/config.yaml
"
}

restart_k3s_if_needed() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
set -euo pipefail
if [[ ! -f /etc/systemd/system/k3s.service ]] && [[ ! -f /usr/local/bin/k3s ]]; then
	echo '? k3s is not installed yet, skipping restart (will be installed in next steps)'
	exit 0
fi
LOCAL_HASH=\$(sudo sha256sum /etc/rancher/k3s/config.yaml 2>/dev/null | awk '{print \$1}' || echo 'none')
SAVED_HASH=\$(sudo cat /etc/rancher/k3s/.config-hash 2>/dev/null || echo 'none')
if [[ \"\$LOCAL_HASH\" != \"\$SAVED_HASH\" ]]; then
	echo '-> Config changed (or first run), restarting k3s...'
	sudo sysctl -w net.ipv4.ip_no_pmtu_disc=1
	sudo touch /etc/sysctl.conf 2>/dev/null
	grep -q 'net.ipv4.ip_no_pmtu_disc=1' /etc/sysctl.conf || echo 'net.ipv4.ip_no_pmtu_disc=1' | sudo tee -a /etc/sysctl.conf > /dev/null
	sudo sysctl -p 2>/dev/null
	sudo systemctl restart k3s
	for i in \$(seq 1 90); do
		if sudo k3s kubectl cluster-info &>/dev/null; then
			echo \"? k3s ready after \${i}s\"
			break
		fi
		if [[ \"\$i\" -eq 90 ]]; then
			echo '? Timeout waiting for k3s restart. Last 30 lines of journal:'
			sudo journalctl -u k3s -n 30 --no-pager
			exit 1
		fi
		sleep 2
	done
	echo '-> Saving config hash for future comparisons...'
	echo \"\$LOCAL_HASH\" | sudo tee /etc/rancher/k3s/.config-hash > /dev/null
	echo '? k3s restarted with new config'
	sudo k3s kubectl get nodes -o wide
else
	echo '? Config unchanged, skipping k3s restart'
fi
"
}

start_k3s() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
set -euo pipefail
echo '-> Starting k3s...'
		echo '-> Cleaning stuck pods from previous runs BEFORE restart...'
		sudo k3s kubectl delete pods --all -n kube-system --force --grace-period=0 2>/dev/null || true
		sudo k3s kubectl delete pods --all -n local-path-storage --force --grace-period=0 2>/dev/null || true
sudo sysctl -w net.ipv4.ip_no_pmtu_disc=1
sudo touch /etc/sysctl.conf 2>/dev/null
grep -q 'net.ipv4.ip_no_pmtu_disc=1' /etc/sysctl.conf || echo 'net.ipv4.ip_no_pmtu_disc=1' | sudo tee -a /etc/sysctl.conf > /dev/null
sudo sysctl -p 2>/dev/null
echo '-> Restarting k3s to apply network fixes to containerd...'
sudo systemctl daemon-reload
sudo systemctl enable --now k3s
echo '-> Waiting for readiness...'
for i in \$(seq 1 120); do
	if sudo k3s kubectl cluster-info &> /dev/null; then
		echo \"k3s ready after \${i}s\"
		break
	fi
	if [[ \"\$i\" -eq 120 ]]; then
		echo '? Timeout waiting for k3s'
		sudo journalctl -u k3s -n 30 --no-pager
		exit 1
	fi
	sleep 2
done
echo '=== Cluster status (snapshot; readiness is gated by wait_for_critical_pods) ==='
sudo k3s kubectl get nodes -o wide
sudo k3s kubectl get pods -A
"
}

install_k3s() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
set -euo pipefail
echo '-> [CRITICAL] Applying MTU fix BEFORE downloading k3s to prevent TLS drops...'
		sudo sysctl -w net.ipv4.ip_no_pmtu_disc=1
		echo '-> Forcing TLS 1.2 for root curl (fixes curl:35 resets)...'
		printf 'tls-max = 1.2\ntlsv1.2\n' | sudo tee /root/.curlrc > /dev/null
sudo touch /etc/sysctl.conf 2>/dev/null
grep -q 'net.ipv4.ip_no_pmtu_disc=1' /etc/sysctl.conf || echo 'net.ipv4.ip_no_pmtu_disc=1' | sudo tee -a /etc/sysctl.conf > /dev/null
sudo sysctl -p 2>/dev/null

echo '-> Downloading and installing k3s directly on VPS (much faster and more reliable than scp)...'
curl -sfL https://get.k3s.io | sudo INSTALL_K3S_BIN_DIR=/usr/local/bin sh -s - server --write-kubeconfig-mode 644

echo 'k3s installed successfully'
"
}

preload_airgap_images() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
		set -euo pipefail
		sudo sysctl -w net.ipv4.ip_no_pmtu_disc=1
		K3S_VER=\$(k3s --version 2>/dev/null | head -1 | awk '{print \$3}')
		if [[ -z \"\${K3S_VER}\" ]]; then
			echo 'ERROR: Could not detect k3s version'
			exit 1
		fi
		echo \"-> Detected k3s version: \${K3S_VER}\"
		IMG_DIR=/var/lib/rancher/k3s/agent/images
		sudo mkdir -p \${IMG_DIR}

		echo '-> Downloading k3s airgap images bundle (contains local-path-provisioner & pause)...'
		wget -q --tries=5 --timeout=60 -O /tmp/k3s-airgap.tar.gz \"https://github.com/k3s-io/k3s/releases/download/\${K3S_VER}/k3s-airgap-images-amd64.tar.gz\"

		echo '-> Importing airgap images into containerd...'
		sudo k3s ctr -n k8s.io images import /tmp/k3s-airgap.tar.gz
		sudo rm -f /tmp/k3s-airgap.tar.gz

		echo '-> Restarting k3s to apply imported images...'
		sudo systemctl restart k3s

		echo '-> Verifying images are loaded:'
		sudo k3s crictl images | grep -E 'local-path-provisioner|pause' || true
	"
}

wait_for_critical_pods() {
	echo '-> Waiting for critical system pods, node, StorageClass and PVC provisioning readiness...'

	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
set -euo pipefail

overall_fail=0

echo '• checking node Ready'
if ! sudo k3s kubectl wait --for=condition=Ready node --all --timeout=120s; then
	echo '  ❌ node is not Ready'
	sudo k3s kubectl get nodes -o wide || true
	overall_fail=1
fi

CRITICAL=(
	"kube-system:k8s-app=kube-dns:coredns"
	"local-path-storage:app=local-path-provisioner:local-path-provisioner"
	"kube-system:k8s-app=metrics-server:metrics-server"
)

for entry in "${CRITICAL[@]}"; do
	ns="${entry%%:*}"
	rest="${entry#*:}"
	sel="${rest%%:*}"
	name="${rest#*:}"

	echo "  • checking $name (ns=$ns, selector=$sel)"

	appeared=false
	for i in $(seq 1 60); do
		if sudo k3s kubectl get pods -n "$ns" -l "$sel" --no-headers 2>/dev/null | grep -q .; then
			appeared=true
			break
		fi
		sleep 2
	done

	if [[ "$appeared" != true ]]; then
		echo "  ❌ $name: no pods found by selector '$sel' in $ns within 120s"
		overall_fail=1
		continue
	fi

	if ! sudo k3s kubectl wait --for=condition=Ready pod -l "$sel" -n "$ns" --timeout=300s; then
		echo "  ❌ $name: not Ready within 300s. Diagnostics:"
		sudo k3s kubectl get pods -n "$ns" -l "$sel" -o wide || true
		sudo k3s kubectl describe pods -n "$ns" -l "$sel" | tail -80 || true
		sudo k3s kubectl logs -n "$ns" -l "$sel" --tail=50 || true
		overall_fail=1
	else
		echo "  ✅ $name is Ready"
	fi
done

echo '• checking StorageClass local-path'
if ! sudo k3s kubectl get storageclass local-path >/dev/null 2>&1; then
	echo '  ❌ StorageClass local-path missing'
	overall_fail=1
else
	default_class=$(sudo k3s kubectl get storageclass local-path -o jsonpath='{.metadata.annotations.storageclass\.kubernetes\.io/is-default-class}' 2>/dev/null || echo '')
	if [[ "$default_class" != "true" ]]; then
		echo '  ⚠️ StorageClass local-path is not default, patching...'
		if ! sudo k3s kubectl patch storageclass local-path -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'; then
			echo '  ❌ failed to patch StorageClass local-path'
			overall_fail=1
		fi
	fi

	mode=$(sudo k3s kubectl get storageclass local-path -o jsonpath='{.volumeBindingMode}' 2>/dev/null || echo '')
	if [[ "$mode" != "WaitForFirstConsumer" ]]; then
		echo "  ❌ StorageClass local-path has wrong volumeBindingMode: ${mode:-<empty>}"
		overall_fail=1
	fi
fi

echo '• smoke-testing PVC provisioning'
SMOKE_NAME="ci-lp-smoke-$(date +%s)"

cat <<MANIFEST | sudo k3s kubectl apply -f -
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
MANIFEST

cleanup_smoke() {
	sudo k3s kubectl delete pod "${SMOKE_NAME}-consumer" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
	sudo k3s kubectl delete pvc "${SMOKE_NAME}" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
}
trap cleanup_smoke EXIT

if ! sudo k3s kubectl wait pvc "${SMOKE_NAME}" -n local-path-storage --for=condition=Bound --timeout=180s; then
	echo '  ❌ Smoke PVC did not become Bound; local-path provisioner is not functional'
	sudo k3s kubectl get pvc,pods -n local-path-storage -o wide || true
	sudo k3s kubectl describe pvc "${SMOKE_NAME}" -n local-path-storage || true
	sudo k3s kubectl get events -n local-path-storage --sort-by=.lastTimestamp | tail -80 || true
	sudo k3s kubectl logs -n local-path-storage -l app=local-path-provisioner --tail=100 || true
	overall_fail=1
else
	echo '  ✅ Smoke PVC bound; local-path provisioning works'
fi

cleanup_smoke
trap - EXIT

for ns in kube-system local-path-storage; do
	bad=$(
		sudo k3s kubectl get pods -n "$ns" --no-headers 2>/dev/null |
			awk '$3 != "Running" && $3 != "Completed" && $3 != "Terminating" {print $1 " " $2 " " $3}' || true
	)

	if [[ -n "$bad" ]]; then
		echo "  ❌ non-ready pods in $ns:"
		echo "$bad"
		overall_fail=1
	fi
done

echo '=== Final pod snapshot ==='
sudo k3s kubectl get pods -A || true

if [[ "$overall_fail" -ne 0 ]]; then
	echo '❌ CRITICAL PODS/NODE/STORAGE/PVC NOT READY — failing provision job'
	exit 1
fi

echo '✅ All critical system components and PVC provisioning are Ready'
EOF
}

ensure_local_path_dir() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
set -euo pipefail
echo '-> Ensuring /opt/local-path-provisioner exists with correct permissions...'
if [[ ! -d /opt/local-path-provisioner ]]; then
	sudo mkdir -p /opt/local-path-provisioner
	echo 'Directory created: /opt/local-path-provisioner'
fi
sudo chmod 0777 /opt/local-path-provisioner
echo 'Permissions set to 0777'
ls -ld /opt/local-path-provisioner
EOF
}

main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all)
		configure_swap
		check_disk_space
		cleanup_old_k3s
		configure_cpu_governor
		configure_disk_encryption
		fix_dns
		prepare_k3s_config
		restart_k3s_if_needed
		ensure_local_path_dir
		start_k3s
		install_k3s
		preload_airgap_images
		;;
	configure_swap) configure_swap ;;
	check_disk_space) check_disk_space ;;
	cleanup_old_k3s) cleanup_old_k3s ;;
	configure_cpu_governor) configure_cpu_governor ;;
	configure_disk_encryption) configure_disk_encryption ;;
	fix_dns) fix_dns ;;
	prepare_k3s_config) prepare_k3s_config ;;
	restart_k3s_if_needed) restart_k3s_if_needed ;;
	start_k3s) start_k3s ;;
	install_k3s) install_k3s ;;
	preload_airgap_images) preload_airgap_images ;;
	wait_for_critical_pods) wait_for_critical_pods ;;
	ensure_local_path_dir) ensure_local_path_dir ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
