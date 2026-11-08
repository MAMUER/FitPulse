#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/utils.sh
source "$SCRIPT_DIR/lib/utils.sh"
# shellcheck source=lib/github.sh
source "$SCRIPT_DIR/lib/github.sh"

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
		sudo touch /etc/sysctl.conf 2>/dev/null
	grep -q 'net.ipv4.ip_no_pmtu_disc=1' /etc/sysctl.conf || echo 'net.ipv4.ip_no_pmtu_disc=1' | sudo tee -a /etc/sysctl.conf > /dev/null
	sudo sysctl -p 2>/dev/null

	sudo systemctl restart k3s
	for i in \$(seq 1 90); do
		if sudo /usr/local/bin/k3s kubectl cluster-info &>/dev/null; then
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
	sudo /usr/local/bin/k3s kubectl get nodes -o wide
else
	echo '? Config unchanged, skipping k3s restart'
fi
"
}

configure_registries() {
	echo "-> Configuring containerd mirrors..."
	sudo mkdir -p /etc/rancher/k3s
	cat <<'EOF' | sudo tee /etc/rancher/k3s/registries.yaml >/dev/null
mirrors:
  "docker.io":
    endpoint:
      - "https://docker.1ms.run"
      - "https://dockerhub.icu"
      - "https://registry-1.docker.io"
  "quay.io":
    endpoint:
      - "https://quay.m.daocloud.io"
      - "https://quay.io"
EOF
	echo "registries.yaml configured"
}

start_k3s() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
set -euo pipefail
echo '-> Starting k3s...'
echo '-> Cleaning stuck pods from previous runs BEFORE restart...'
sudo /usr/local/bin/k3s kubectl delete pods --all -n kube-system --force --grace-period=0 2>/dev/null || true
sudo /usr/local/bin/k3s kubectl delete pods --all -n local-path-storage --force --grace-period=0 2>/dev/null || true

sudo touch /etc/sysctl.conf 2>/dev/null
grep -q 'net.ipv4.ip_no_pmtu_disc=1' /etc/sysctl.conf || echo 'net.ipv4.ip_no_pmtu_disc=1' | sudo tee -a /etc/sysctl.conf > /dev/null
sudo sysctl -p 2>/dev/null

echo '-> Restarting k3s to apply network fixes to containerd...'
sudo systemctl daemon-reload
sudo systemctl enable --now k3s

echo '-> Waiting for readiness...'
for i in \$(seq 1 120); do
	if sudo /usr/local/bin/k3s kubectl cluster-info &> /dev/null; then
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
sudo /usr/local/bin/k3s kubectl get nodes -o wide
sudo /usr/local/bin/k3s kubectl get pods -A
"
}

install_k3s() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
set -euo pipefail

# === 1. Ensure PMTUD is enabled (revert previous harm) ===
sudo sed -i '/ip_no_pmtu_disc/d' /etc/sysctl.conf 2>/dev/null || true

# === 2. MSS Clamping instead of fixed MTU (safe for WSL2) ===
sudo iptables -t mangle -F FORWARD 2>/dev/null || true
sudo iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || true

if ! grep -q 'TCPMSS' /etc/rc.local 2>/dev/null; then
	sudo mkdir -p /etc/network/if-pre-up.d
	echo '#!/bin/sh' | sudo tee /etc/network/if-pre-up.d/mss-clamp > /dev/null
	echo 'iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || true' | sudo tee -a /etc/network/if-pre-up.d/mss-clamp > /dev/null
	sudo chmod +x /etc/network/if-pre-up.d/mss-clamp
fi

# === 3. Skip install if k3s already present and working ===
K3S_EXISTING_VERSION=\$(/usr/local/bin/k3s --version 2>/dev/null | head -1 || true)
if [[ -x /usr/local/bin/k3s ]] && [[ -n \"\$K3S_EXISTING_VERSION\" ]]; then
	echo \"-> k3s already installed: \$K3S_EXISTING_VERSION\"
	echo '-> Skipping re-download/install'
	exit 0
fi
# Remove any broken/leftover binary so the installer can replace it cleanly
rm -f /usr/local/bin/k3s 2>/dev/null || true

# === 4. Install k3s ===
echo '-> Installing k3s from scratch...'
curl -sfL --retry 3 --retry-delay 5 https://get.k3s.io | sudo INSTALL_K3S_BIN_DIR=/usr/local/bin sh -s - server --write-kubeconfig-mode 644
echo 'k3s installed successfully'
"
}

preload_airgap_images() {
	local script_file="/tmp/k3s-preload-remote.sh"

	log_info "-> Detecting k3s version on VPS..."
	local k3s_ver
	k3s_ver=$(bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" \
		"/usr/local/bin/k3s --version 2>/dev/null | head -1 | awk '{print \$3}'" |
		tr -d '\r' | head -1) || true
	k3s_ver="${k3s_ver//[[:space:]]/}"

	# The tag must look like a real k3s version (e.g. v1.37.2+k3s1). Refuse
	# anything else so a corrupted capture can never build a bogus URL.
	if [[ ! "$k3s_ver" =~ ^v[0-9]+\.[0-9]+\.[0-9]+[A-Za-z0-9.+-]*$ ]]; then
		log_error "Could not detect a valid k3s version on VPS (got: '${k3s_ver:-<empty>}')"
		exit 1
	fi
	log_info "-> Detected k3s version: $k3s_ver"

	# URL-encode the '+' so the release URL stays valid.
	local encoded_tag="${k3s_ver//+/%2B}"

	# Build a small helper script locally and ship *that* to the VPS (scp of a
	# few KB is reliable, unlike pushing the 189 MB bundle). The VPS downloads
	# the bundle itself with resume support, verifies the checksum, imports it.
	cat >"$script_file" <<'REMOTE_SCRIPT'
#!/bin/bash
set -euo pipefail

ASSET="k3s-airgap-images-amd64.tar.gz"
SUMS="sha256sum-amd64.txt"
BUNDLE="/opt/k3s-airgap-images-amd64.tar.gz"
SUMS_FILE="/opt/sha256sum-amd64.txt"
BASE_URL="https://github.com/k3s-io/k3s/releases/download/__TAG__"

fetch() {
	if command -v curl >/dev/null 2>&1; then
		curl -fL --retry 5 --retry-delay 3 -C - -o "$1" "$2"
	elif command -v wget >/dev/null 2>&1; then
		wget -c -O "$1" "$2"
	else
		echo "Neither curl nor wget is installed on VPS" >&2
		exit 1
	fi
}

echo '-> Downloading airgap bundle on VPS (with resume)...'
rm -f "${BUNDLE}" "${SUMS_FILE}"
fetch "${BUNDLE}" "${BASE_URL}/${ASSET}"
echo '-> Downloading checksums file...'
fetch "${SUMS_FILE}" "${BASE_URL}/${SUMS}"

echo '-> Verifying SHA-256 checksum...'
cd /opt
awk -v name="${ASSET}" '$2 == name { print }' "${SUMS_FILE}" | sha256sum -c --status -

echo '-> Validating archive...'
tar -tzf "${BUNDLE}" >/dev/null

echo '-> Importing images into containerd...'
sudo k3s ctr -n k8s.io images import "${BUNDLE}"
sudo rm -f "${BUNDLE}" "${SUMS_FILE}"

for img in $(sudo k3s ctr -n k8s.io images ls -q | grep '^docker.io/' || true); do
	new_name="${img#docker.io/}"
	sudo k3s ctr -n k8s.io images tag "$img" "$new_name" 2>/dev/null || true
done

sudo systemctl restart k3s
echo '-> Verifying:'
sudo k3s crictl images | grep -E 'local-path-provisioner|pause' || true

# Clean up the helper script only on full success (leave it on failure so
# ssh-retry can re-run it without re-uploading).
rm -f /tmp/k3s-preload-remote.sh
REMOTE_SCRIPT

	sed -i "s|__TAG__|${encoded_tag}|g" "$script_file"

	log_info "-> Uploading helper script to VPS..."
	bash scripts/ssh-retry.sh scp "$script_file" "${VPS_USER}@${VPS_HOST}:/tmp/k3s-preload-remote.sh"

	log_info "-> Running airgap preload on VPS..."
	bash scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" \
		"sudo bash /tmp/k3s-preload-remote.sh"

	rm -f "$script_file"
	log_info "Airgap images imported successfully"
}

wait_for_critical_pods() {
	echo '-> Waiting for critical system pods, node, StorageClass and PVC provisioning readiness...'
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
set -euo pipefail
overall_fail=0

echo '• checking node Ready'
if ! sudo /usr/local/bin/k3s kubectl wait --for=condition=Ready node --all --timeout=120s; then
	echo '  ❌ node is not Ready'
	sudo /usr/local/bin/k3s kubectl get nodes -o wide || true
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
		if sudo /usr/local/bin/k3s kubectl get pods -n "$ns" -l "$sel" --no-headers 2>/dev/null | grep -q .; then
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

	if ! sudo /usr/local/bin/k3s kubectl wait --for=condition=Ready pod -l "$sel" -n "$ns" --timeout=300s; then
		echo "  ❌ $name: not Ready within 300s. Diagnostics:"
		sudo /usr/local/bin/k3s kubectl get pods -n "$ns" -l "$sel" -o wide || true
		sudo /usr/local/bin/k3s kubectl describe pods -n "$ns" -l "$sel" | tail -80 || true
		sudo /usr/local/bin/k3s kubectl logs -n "$ns" -l "$sel" --tail=50 || true
		overall_fail=1
	else
		echo "  ✅ $name is Ready"
	fi
done

echo '• checking StorageClass local-path'
if ! sudo /usr/local/bin/k3s kubectl get storageclass local-path >/dev/null 2>&1; then
	echo '  ❌ StorageClass local-path missing'
	overall_fail=1
else
	default_class=$(sudo /usr/local/bin/k3s kubectl get storageclass local-path -o jsonpath='{.metadata.annotations.storageclass\.kubernetes\.io/is-default-class}' 2>/dev/null || echo '')
	if [[ "$default_class" != "true" ]]; then
		echo '  ⚠️ StorageClass local-path is not default, patching...'
		if ! sudo /usr/local/bin/k3s kubectl patch storageclass local-path -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'; then
			echo '  ❌ failed to patch StorageClass local-path' >&2
			overall_fail=1
		fi
	fi

	mode=$(sudo /usr/local/bin/k3s kubectl get storageclass local-path -o jsonpath='{.volumeBindingMode}' 2>/dev/null || echo '')
	if [[ "$mode" != "WaitForFirstConsumer" ]]; then
		echo "  ❌ StorageClass local-path has wrong volumeBindingMode: ${mode:-<empty>}"
		overall_fail=1
	fi
fi

echo '-> Cleaning stale smoke resources before gate...'
sudo /usr/local/bin/k3s kubectl delete pods -n local-path-storage -l app.kubernetes.io/name=lp-smoke --ignore-not-found --wait=false >/dev/null 2>&1 || true
sudo /usr/local/bin/k3s kubectl delete pvc -n local-path-storage -l app.kubernetes.io/name=lp-smoke --ignore-not-found --wait=false >/dev/null 2>&1 || true
for pod in $(sudo /usr/local/bin/k3s kubectl get pods -n local-path-storage -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | grep -E '^(lp-smoke-|ci-lp-smoke-|helper-pod-)' || true); do
	sudo /usr/local/bin/k3s kubectl delete pod "$pod" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
done
for pvc in $(sudo /usr/local/bin/k3s kubectl get pvc -n local-path-storage -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | grep -E '^(lp-smoke-|ci-lp-smoke-)' || true); do
	sudo /usr/local/bin/k3s kubectl delete pvc "$pvc" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
done
sleep 5

echo '• smoke-testing PVC provisioning'
SMOKE_NAME="ci-lp-smoke-$(date +%s)"
cat <<MANIFEST | sudo /usr/local/bin/k3s kubectl apply -f -
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
MANIFEST

cleanup_smoke() {
	sudo /usr/local/bin/k3s kubectl delete pod "${SMOKE_NAME}-consumer" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
	sudo /usr/local/bin/k3s kubectl delete pvc "${SMOKE_NAME}" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
}
trap cleanup_smoke EXIT

BOUND=false
for i in $(seq 1 60); do
	phase=$(sudo /usr/local/bin/k3s kubectl get pvc "${SMOKE_NAME}" -n local-path-storage -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
	if [[ "$phase" == "Bound" ]]; then
		BOUND=true
		echo "  ✅ Smoke PVC became Bound after $((i * 3))s"
		break
	fi
	if ((i % 10 == 0)); then
		echo "  Still waiting... attempt $i/60, phase=${phase:-NotFound}"
	fi
	sleep 3
done

if [[ "$BOUND" != true ]]; then
	echo '  ❌ Smoke PVC did not become Bound; local-path provisioner is not functional'
	sudo /usr/local/bin/k3s kubectl get pvc,pods -n local-path-storage -o wide || true
	sudo /usr/local/bin/k3s kubectl describe pvc "${SMOKE_NAME}" -n local-path-storage || true
	sudo /usr/local/bin/k3s kubectl get events -n local-path-storage --sort-by=.lastTimestamp | tail -80 || true
	sudo /usr/local/bin/k3s kubectl logs -n local-path-storage -l app=local-path-provisioner --tail=100 || true
	overall_fail=1
else
	echo '  ✅ Smoke PVC bound; local-path provisioning works'
fi

cleanup_smoke
trap - EXIT

echo '-> Cleaning up stale non-ready pods in system namespaces...'
for ns in kube-system local-path-storage; do
	sudo /usr/local/bin/k3s kubectl get pods -n "$ns" --no-headers 2>/dev/null |
		awk '$3 != "Running" && $3 != "Completed" && $3 != "Terminating" && $1 !~ /^(lp-smoke-|ci-lp-smoke-|helper-pod-|create-pvc-|delete-pvc-)/ {print $1}' |
		while read -r pod; do
			sudo /usr/local/bin/k3s kubectl delete pod "$pod" -n "$ns" --ignore-not-found --wait=false >/dev/null 2>&1 || true
		done
done
sleep 15

for ns in kube-system local-path-storage; do
	bad=$(
		sudo /usr/local/bin/k3s kubectl get pods -n "$ns" --no-headers 2>/dev/null |
			awk '$3 != "Running" && $3 != "Completed" && $3 != "Terminating" && $1 !~ /^(lp-smoke-|ci-lp-smoke-|helper-pod-|create-pvc-|delete-pvc-)/ {print $1 " " $2 " " $3}' || true
	)
	if [[ -n "$bad" ]]; then
		echo "  ❌ non-ready pods in $ns:"
		echo "$bad"
		overall_fail=1
	fi
done

echo '=== Final pod snapshot ==='
sudo /usr/local/bin/k3s kubectl get pods -A || true

if [[ "$overall_fail" -ne 0 ]]; then
	echo '❌ CRITICAL PODS/NODE/STORAGE/PVC NOT READY — failing provision job' >&2
	exit 1
fi
echo '✅ All critical system components and PVC provisioning are Ready'
EOF
}

preload_busybox_image() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
set -euo pipefail
TARGET="docker.io/rancher/mirrored-library-busybox:1.37.0"
echo '-> Ensuring BusyBox helper image is available locally...'
if sudo /usr/local/bin/k3s crictl images 2>/dev/null | grep -E 'rancher/mirrored-library-busybox[[:space:]]+1\.37\.0' >/dev/null; then
	echo "BusyBox helper image ready: ${TARGET}"
	exit 0
fi
# Fallback: reuse any locally available busybox image by retagging it.
src=$(sudo /usr/local/bin/k3s ctr -n k8s.io images ls -q 2>/dev/null | grep -i busybox | head -1 || true)
if [[ -n "${src}" ]]; then
	echo "-> Retagging ${src} to ${TARGET}"
	sudo /usr/local/bin/k3s ctr -n k8s.io images rm "${TARGET}" >/dev/null 2>&1 || true
	sudo /usr/local/bin/k3s ctr -n k8s.io images tag "${src}" "${TARGET}" || true
	echo "BusyBox helper image ready: ${TARGET}"
	exit 0
fi
echo "Could not find any BusyBox image locally; airgap import may have failed" >&2
sudo /usr/local/bin/k3s crictl images | grep -i busybox || true
exit 1
EOF
}

cleanup_stale_smoke_resources() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
set -euo pipefail
echo '-> Cleaning stale local-path smoke resources...'
sudo /usr/local/bin/k3s kubectl delete pods -n local-path-storage -l app.kubernetes.io/name=lp-smoke --ignore-not-found --wait=false >/dev/null 2>&1 || true
sudo /usr/local/bin/k3s kubectl delete pvc -n local-path-storage -l app.kubernetes.io/name=lp-smoke --ignore-not-found --wait=false >/dev/null 2>&1 || true
for pod in $(sudo /usr/local/bin/k3s kubectl get pods -n local-path-storage -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | grep -E '^(lp-smoke-|ci-lp-smoke-)' || true); do
	sudo /usr/local/bin/k3s kubectl delete pod "$pod" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
done
for pvc in $(sudo /usr/local/bin/k3s kubectl get pvc -n local-path-storage -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | grep -E '^(lp-smoke-|ci-lp-smoke-)' || true); do
	sudo /usr/local/bin/k3s kubectl delete pvc "$pvc" -n local-path-storage --ignore-not-found --wait=false >/dev/null 2>&1 || true
done
sleep 5
echo 'Stale smoke resources cleaned'
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
		configure_registries
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
	configure_registries) configure_registries ;;
	install_k3s) install_k3s ;;
	preload_airgap_images) preload_airgap_images ;;
	preload_busybox_image) preload_busybox_image ;;
	ensure_local_path_dir) ensure_local_path_dir ;;
	cleanup_stale_smoke_resources) cleanup_stale_smoke_resources ;;
	wait_for_critical_pods) wait_for_critical_pods ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
