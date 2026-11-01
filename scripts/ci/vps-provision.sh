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
	local tg_bot="${TELEGRAM_BOT_TOKEN:-}"
	local tg_chat="${TELEGRAM_CHAT_ID:-}"

	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "TG_BOT='${tg_bot}' TG_CHAT='${tg_chat}' bash -s" <<'EOF'
set -euo pipefail

DISK_USAGE=$(df / | tail -1 | awk '{print $5}' | sed 's/%//')
echo "Current disk usage: ${DISK_USAGE}%"

if [[ "$DISK_USAGE" -gt 90 ]]; then
	echo "WARNING: Disk usage is over 90%!"

	if [[ -n "${TG_BOT:-}" && -n "${TG_CHAT:-}" ]]; then
		curl -s -X POST "https://api.telegram.org/bot${TG_BOT}/sendMessage" \
			-H "Content-Type: application/json" \
			-d "{\"chat_id\": \"${TG_CHAT}\", \"text\": \"VPS Disk usage is ${DISK_USAGE}%!\"}" || true
	else
		echo "Telegram credentials are not set, skipping notification"
	fi
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
Restart=always
RestartSec=10s
EOT
sudo systemctl daemon-reload
echo "✅ k3s watchdog and CPU quota configured"
EOF
}
configure_disk_encryption() {
	./scripts/ssh-retry.sh scp configs/k8s/scripts/configure-storage-encryption.sh "${VPS_USER}@${VPS_HOST}:/tmp/configure-storage-encryption.sh"

	if ./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo chmod +x /tmp/configure-storage-encryption.sh && sudo /tmp/configure-storage-encryption.sh --yes"; then
		echo "Disk encryption configuration completed"
	else
		echo "::warning::Disk encryption configuration failed"

		if [[ "${ALLOW_DISK_ENCRYPTION_FAILURE:-false}" != "true" ]]; then
			echo "Failing provision because disk encryption is required."
			echo "Set ALLOW_DISK_ENCRYPTION_FAILURE=true only if you intentionally want to bypass this."
			exit 1
		fi

		echo "ALLOW_DISK_ENCRYPTION_FAILURE=true, continuing despite encryption failure"
	fi
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
sudo touch /etc/sysctl.conf 2>/dev/null
grep -q 'net.ipv4.ip_no_pmtu_disc=1' /etc/sysctl.conf || echo 'net.ipv4.ip_no_pmtu_disc=1' | sudo tee -a /etc/sysctl.conf > /dev/null
sudo sysctl -p 2>/dev/null

echo '-> Downloading and installing k3s directly on VPS (much faster and more reliable than scp)...'
		curl --tlsv1.2 -sfL https://get.k3s.io -o /tmp/k3s-install.sh
		sed -i 's/curl -/curl --tlsv1.2 -/g' /tmp/k3s-install.sh
		sudo INSTALL_K3S_BIN_DIR=/usr/local/bin sh /tmp/k3s-install.sh server --write-kubeconfig-mode 644

echo 'k3s installed successfully'
"
}

preload_airgap_images() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
set -euo pipefail
echo '-> Preloading airgap images with strict timeout and quiet mode...'

# Применяем MTU фикс перед началом
sudo sysctl -w net.ipv4.ip_no_pmtu_disc=1 >/dev/null 2>&1 || true

K3S_VER=$(k3s --version 2>/dev/null | head -1 | awk '{print $3}')
if [[ -z "${K3S_VER}" ]]; then
    echo 'ERROR: Could not detect k3s version'
    exit 1
fi
echo "-> Detected k3s version: ${K3S_VER}"

IMG_DIR=/var/lib/rancher/k3s/agent/images
sudo mkdir -p ${IMG_DIR}

ARCHIVE_NAME="k3s-airgap-images-amd64.tar.gz"
REMOTE_PATH="/tmp/${ARCHIVE_NAME}"

echo "-> Downloading airgap bundle directly to node..."
# Скачиваем wget с таймаутом соединения и чтения
wget -q --tries=3 --timeout=60 --read-timeout=120 -O "${REMOTE_PATH}" \
    "https://github.com/k3s-io/k3s/releases/download/${K3S_VER}/${ARCHIVE_NAME}"

FILE_SIZE=$(du -h "${REMOTE_PATH}" | cut -f1)
echo "-> Bundle downloaded (${FILE_SIZE}). Starting import..."

# ИМПОРТ С ТАЙМАУТОМ И БЕЗ ВЫВОДА ПРОГРЕССА
# 1. timeout 300s убьет процесс, если он висит дольше 5 минут.
# 2. > /dev/null 2>&1 скрывает шумные progress bars от SSH.
# 3. || true позволяет продолжить, даже если импорт частично упал (проверим ниже).
echo "-> Running ctr import (quiet mode, 5min timeout)..."
if ! timeout 300 sudo k3s ctr -n k8s.io images import "${REMOTE_PATH}" > /dev/null 2>&1; then
    echo "⚠️ ctr import timed out or failed silently. Checking if images are present anyway..."
fi

# Удаляем архив сразу после попытки импорта, чтобы освободить диск
rm -f "${REMOTE_PATH}"

echo "-> Verifying critical images are loaded..."
MISSING_IMAGES=false

for IMG in "local-path-provisioner" "pause" "coredns"; do
    if ! sudo k3s crictl images | grep -q "$IMG"; then
        echo "❌ Image missing: $IMG"
        MISSING_IMAGES=true
    else
        echo "✅ Found: $IMG"
    fi
done

if [[ "$MISSING_IMAGES" == true ]]; then
    echo "❌ Airgap preload incomplete. Required images are missing."
    sudo k3s crictl images || true

    if [[ "${ALLOW_AIRGAP_FALLBACK:-false}" != "true" ]]; then
        echo "❌ Failing provision because airgap images are incomplete and network fallback is not explicitly allowed."
        exit 1
    fi

    echo "::warning::ALLOW_AIRGAP_FALLBACK=true, continuing despite incomplete airgap images."
else
    echo "✅ All critical airgap images verified."
fi

echo "-> Restarting k3s to apply imported images..."
sudo systemctl restart k3s

echo "-> Waiting for k3s API readiness after restart..."
for i in $(seq 1 60); do
    if sudo k3s kubectl cluster-info &>/dev/null; then
        echo "✅ k3s ready after ${i}s"
        break
    fi
    if [[ "$i" -eq 60 ]]; then
        echo "❌ Timeout waiting for k3s restart"
        sudo journalctl -u k3s -n 30 --no-pager
        exit 1
    fi
    sleep 2
done
EOF
}

preload_busybox_image() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'INNEREOF'
set -euo pipefail

BUSYBOX_TARGET="docker.io/library/busybox:1.37.0"
BUSYBOX_ALIAS="busybox:1.37.0"

ensure_alias() {
	sudo k3s ctr -n k8s.io images rm "${BUSYBOX_ALIAS}" >/dev/null 2>&1 || true
	sudo k3s ctr -n k8s.io images tag "${BUSYBOX_TARGET}" "${BUSYBOX_ALIAS}" || true
}

echo '-> Checking if canonical BusyBox image is already present locally...'
if sudo k3s ctr -n k8s.io images ls -q 2>/dev/null | grep -Fxq "${BUSYBOX_TARGET}"; then
	ensure_alias
	echo "BusyBox already present as ${BUSYBOX_TARGET}"
	exit 0
fi

pulled=false

for src in "mirror.gcr.io/library/busybox:1.37.0" "docker.m.daocloud.io/library/busybox:1.37.0" "${BUSYBOX_TARGET}"; do
	echo "-> Trying to pull from ${src}..."

	if timeout 120 sudo k3s crictl pull "${src}"; then
		if [[ "${src}" != "${BUSYBOX_TARGET}" ]]; then
			echo "-> Retagging ${src} to canonical name ${BUSYBOX_TARGET}"
			sudo k3s ctr -n k8s.io images rm "${BUSYBOX_TARGET}" >/dev/null 2>&1 || true
			sudo k3s ctr -n k8s.io images tag "${src}" "${BUSYBOX_TARGET}"
		fi

		ensure_alias
		pulled=true
		break
	fi

	echo "Failed to pull from ${src}. Trying next source..."
done

if [[ "${pulled}" != true ]]; then
	echo "Could not preload BusyBox helper image from any source"
	echo "--- Diagnostics ---"
	ip link show dev eth0 | grep mtu || true
	cat /etc/resolv.conf || true
	curl -I --max-time 10 https://registry-1.docker.io/v2/ || true
	curl -I --max-time 10 "https://auth.docker.io/token?service=registry.docker.io" || true
	sudo k3s crictl images | grep busybox || true
	exit 1
fi

echo "BusyBox helper image ready: ${BUSYBOX_TARGET}"
INNEREOF
}

cleanup_stale_smoke_resources() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'INNEREOF'
set -euo pipefail
echo '-> Cleaning stale local-path smoke resources...'
sudo k3s kubectl delete pods -n local-path-storage -l app.kubernetes.io/name=lp-smoke --ignore-not-found --wait=false >/dev/null 2>&1 || true
sudo k3s kubectl delete pvc -n local-path-storage -l app.kubernetes.io/name=lp-smoke --ignore-not-found --wait=false >/dev/null 2>&1 || true
for pod in $(sudo k3s kubectl get pods -n local-path-storage -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | grep -E '^(lp-smoke-|ci-lp-smoke-)' || true); do
	sudo k3s kubectl delete pod "$pod" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
done
for pvc in $(sudo k3s kubectl get pvc -n local-path-storage -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | grep -E '^(lp-smoke-|ci-lp-smoke-)' || true); do
	sudo k3s kubectl delete pvc "$pvc" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
done
sleep 5
echo 'Stale smoke resources cleaned'
INNEREOF
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

echo '-> Cleaning stale smoke resources before gate...'
sudo k3s kubectl delete pods -n local-path-storage -l app.kubernetes.io/name=lp-smoke --ignore-not-found --wait=false >/dev/null 2>&1 || true
sudo k3s kubectl delete pvc -n local-path-storage -l app.kubernetes.io/name=lp-smoke --ignore-not-found --wait=false >/dev/null 2>&1 || true

for pod in $(sudo k3s kubectl get pods -n local-path-storage -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | grep -E '^(lp-smoke-|ci-lp-smoke-)' || true); do
	sudo k3s kubectl delete pod "$pod" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
done

for pvc in $(sudo k3s kubectl get pvc -n local-path-storage -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | grep -E '^(lp-smoke-|ci-lp-smoke-)' || true); do
	sudo k3s kubectl delete pvc "$pvc" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
done

sleep 5

echo '• smoke-testing PVC provisioning'
SMOKE_NAME="ci-lp-smoke-$(date +%s)"

cat <<MANIFEST | sudo k3s kubectl apply -f -
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
      image: docker.io/library/busybox:1.37.0
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

bound=false
for i in $(seq 1 60); do
	phase=$(sudo k3s kubectl get pvc "${SMOKE_NAME}" -n local-path-storage -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
	if [[ "$phase" == "Bound" ]]; then
		bound=true
		echo "  Smoke PVC became Bound after $((i * 3))s"
		break
	fi
	echo "  Attempt $i/60: smoke PVC phase=${phase:-NotFound}"
	sleep 3
done

if [[ "$bound" != true ]]; then
	echo '  Smoke PVC did not become Bound; local-path provisioner is not functional'
	sudo k3s kubectl get pvc,pods -n local-path-storage -o wide || true
	sudo k3s kubectl describe pvc "${SMOKE_NAME}" -n local-path-storage || true
	sudo k3s kubectl get events -n local-path-storage --sort-by=.lastTimestamp | tail -80 || true
	sudo k3s kubectl logs -n local-path-storage -l app=local-path-provisioner --tail=100 || true
	sudo k3s crictl images | grep busybox || echo "BusyBox image NOT found locally"
	overall_fail=1
else
	echo '  Smoke PVC bound; local-path provisioning works'
fi

cleanup_smoke
trap - EXIT

for ns in kube-system local-path-storage; do
	bad=$(
		sudo k3s kubectl get pods -n "$ns" --no-headers 2>/dev/null |
			awk '$3 != "Running" && $3 != "Completed" && $3 != "Terminating" && $1 !~ /^(lp-smoke-|ci-lp-smoke-)/ {print $1 " " $2 " " $3}' || true
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
		ensure_local_path_dir
		prepare_k3s_config
		restart_k3s_if_needed
		install_k3s
		preload_airgap_images
		preload_busybox_image
		cleanup_stale_smoke_resources
		start_k3s
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
	preload_busybox_image) preload_busybox_image ;;
	cleanup_stale_smoke_resources) cleanup_stale_smoke_resources ;;
	wait_for_critical_pods) wait_for_critical_pods ;;
	ensure_local_path_dir) ensure_local_path_dir ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
