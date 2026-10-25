#!/bin/bash
set -euo pipefail

configure_swap() {
	# Проверка, что VPS доступен по SSH
	if ! timeout 10 ssh -o BatchMode=yes -o ConnectTimeout=10 "${VPS_USER}@${VPS_HOST}" "echo ok" >/dev/null 2>&1; then
		echo "⚠️ VPS is not reachable via SSH — skipping swap configuration"
		return 0
	fi
	# Если swap уже настроен — пропускаем
	if timeout 10 ssh -o BatchMode=yes -o ConnectTimeout=10 "${VPS_USER}@${VPS_HOST}" "test -f /tmp/.swap-configured" >/dev/null 2>&1; then
		echo "✅ Swap already configured, skipping"
		return 0
	fi
	./scripts/ssh-retry.sh scp configs/k8s/scripts/configure-swap.sh "${VPS_USER}@${VPS_HOST}:/tmp/configure-swap.sh"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo chmod +x /tmp/configure-swap.sh && sudo /tmp/configure-swap.sh && touch /tmp/.swap-configured"
}

check_disk_space() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
DISK_USAGE=$(df / | tail -1 | awk '{print $5}' | sed 's/%//')
echo "Current disk usage: ${DISK_USAGE}%"
if [ "$DISK_USAGE" -gt 90 ]; then
	echo "⚠️ WARNING: Disk usage is over 90%!"
	curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
		-H "Content-Type: application/json" \
		-d "{\"chat_id\": \"${TELEGRAM_CHAT_ID}\", \"text\": \"⚠️ VPS Disk usage is ${DISK_USAGE}%!\"}" || true
fi
EOF
}

cleanup_old_k3s() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
set -x
echo \"-> Stopping k3s...\"
systemctl stop k3s 2>/dev/null || true
pkill -9 -f \"k3s server\" 2>/dev/null || true
echo \"-> Removing k3s files...\"
rm -rf /var/lib/rancher/k3s /etc/rancher/k3s
rm -f /usr/local/bin/k3s /etc/systemd/system/k3s.service /etc/systemd/system/k3s*.timer
systemctl daemon-reload
echo \"Cleanup done\"
"
}

configure_cpu_governor() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
set -euo pipefail
echo "-> Configuring CPU governor..."
if [ -d /sys/devices/system/cpu/cpufreq ]; then
	for gov in /sys/devices/system/cpu/cpufreq/policy*/scaling_governor; do
		[ -f "$gov" ] && echo "performance" | sudo tee "$gov" >/dev/null || true
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

configure_docker_logs() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" <<'EOF'
echo "-> Setting up Docker log rotation..."
sudo mkdir -p /etc/docker
cat <<EOT | sudo tee /etc/docker/daemon.json
{
	"log-driver": "json-file",
	"log-opts": {
		"max-size": "10m",
		"max-file": "3"
	}
}
EOT
echo "-> Setting up journald size limit..."
sudo mkdir -p /etc/systemd/journald.conf.d
cat <<EOT | sudo tee /etc/systemd/journald.conf.d/99-size-limit.conf
[Journal]
SystemMaxUse=500M
SystemMaxFileSize=50M
MaxRetentionSec=1week
EOT
sudo systemctl daemon-reload
sudo systemctl restart systemd-journald
echo "-> Logs rotation configured"
EOF
}

configure_disk_encryption() {
	./scripts/ssh-retry.sh scp configs/k8s/scripts/configure-storage-encryption.sh "${VPS_USER}@${VPS_HOST}:/tmp/configure-storage-encryption.sh"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo chmod +x /tmp/configure-storage-encryption.sh && sudo /tmp/configure-storage-encryption.sh || true"
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
limit_k3s_memory() {
	./scripts/ssh-retry.sh scp configs/k8s/scripts/k3s-memory-watchdog.sh "${VPS_USER}@${VPS_HOST}:/tmp/k3s-memory-watchdog.sh"
	./scripts/ssh-retry.sh scp configs/k8s/scripts/setup-k3s-memory.sh "${VPS_USER}@${VPS_HOST}:/tmp/setup-k3s-memory.sh"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo chmod +x /tmp/setup-k3s-memory.sh && sudo /tmp/setup-k3s-memory.sh"
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
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
mkdir -p /etc/rancher/k3s
echo 'Directory /etc/rancher/k3s created'
"
	./scripts/ssh-retry.sh scp /tmp/k3s-config-remote.yaml "${VPS_USER}@${VPS_HOST}:/etc/rancher/k3s/config.yaml"
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
echo '=== Config written to /etc/rancher/k3s/config.yaml ==='
cat /etc/rancher/k3s/config.yaml
"
}

restart_k3s_if_needed() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
set -euo pipefail
if [ ! -f /etc/systemd/system/k3s.service ] && [ ! -f /usr/local/bin/k3s ]; then
	echo '✅ k3s is not installed yet, skipping restart (will be installed in next steps)'
	exit 0
fi
LOCAL_HASH=\$(sudo sha256sum /etc/rancher/k3s/config.yaml 2>/dev/null | awk '{print \$1}' || echo 'none')
SAVED_HASH=\$(sudo cat /etc/rancher/k3s/.config-hash 2>/dev/null || echo 'none')
if [ \"\$LOCAL_HASH\" != \"\$SAVED_HASH\" ]; then
	echo '-> Config changed (or first run), restarting k3s to apply kubelet-arg...'
	sudo systemctl restart k3s
	for i in \$(seq 1 90); do
		if k3s kubectl cluster-info &>/dev/null; then
			echo \"✅ k3s ready after \${i}s\"
			break
		fi
		if [ \"\$i\" -eq 90 ]; then
			echo '❌ Timeout waiting for k3s restart'
			journalctl -u k3s -n 30 --no-pager
			exit 1
		fi
		sleep 2
	done
	echo '-> Saving config hash for future comparisons...'
	echo \"\$LOCAL_HASH\" | sudo tee /etc/rancher/k3s/.config-hash > /dev/null
	echo '✅ k3s restarted with new config'
	k3s kubectl get nodes -o wide
else
	echo '✅ Config unchanged, skipping k3s restart'
fi
"
}

start_k3s() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
echo \"-> Starting k3s...\"
systemctl daemon-reload
systemctl enable --now k3s
echo \"-> Waiting for readiness...\"
for i in \$(seq 1 120); do
	if k3s kubectl cluster-info &> /dev/null; then
		echo \"k3s ready after \${i}s\"
		break
	fi
	[ \$i -eq 120 ] && { echo \"Timeout\"; journalctl -u k3s -n 20 --no-pager; exit 1; }
	sleep 2
done
echo \"=== Cluster status ===\"
k3s kubectl get nodes -o wide
k3s kubectl get pods -A
"
}

download_k3s() {
	echo "-> Downloading k3s binary and install script..."
	curl --proto =https -sfL -o /tmp/k3s https://github.com/k3s-io/k3s/releases/download/v1.36.2%2Bk3s1/k3s
	chmod +x /tmp/k3s
	curl --proto =https -sfL -o /tmp/install.sh https://get.k3s.io
	chmod +x /tmp/install.sh
	echo "Download complete."
}

install_k3s() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
set -euo pipefail
echo \"-> Moving k3s binary to /usr/local/bin...\"
sudo mv /tmp/k3s-install/k3s /usr/local/bin/k3s || mv /tmp/k3s-install/k3s /usr/local/bin/k3s
sudo chmod +x /usr/local/bin/k3s || chmod +x /usr/local/bin/k3s
echo \"-> Running k3s install script with Docker runtime...\"
export INSTALL_K3S_SKIP_DOWNLOAD=true
chmod +x /tmp/k3s-install/install.sh
sudo /tmp/k3s-install/install.sh server --write-kubeconfig-mode 644 --docker || /tmp/k3s-install/install.sh server --write-kubeconfig-mode 644 --docker
echo \"k3s installed successfully\"
rm -rf /tmp/k3s-install
"
}

upload_k3s_files() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "mkdir -p /tmp/k3s-install"
	./scripts/ssh-retry.sh scp /tmp/k3s "${VPS_USER}@${VPS_HOST}:/tmp/k3s-install/k3s"
	./scripts/ssh-retry.sh scp /tmp/install.sh "${VPS_USER}@${VPS_HOST}:/tmp/k3s-install/install.sh"
}

pre_pull_system_images() {
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
echo \"-> Pre-pulling system images via Docker...\"
docker pull rancher/mirrored-pause:3.6 || echo \"pause image pull failed, continuing\"
docker pull rancher/mirrored-coredns-coredns:1.10.1 || echo \"coredns pull failed\"
docker pull registry.k8s.io/ingress-nginx/controller:v1.11.3 || echo \"ingress-nginx pull failed\"
docker pull rancher/local-path-provisioner:v0.0.26 || echo \"local-path pull failed\"
docker pull busybox:latest || echo \"busybox pull failed\"
echo \"Docker images:\"
docker images | head -10
"
}

main() {
	local cmd="${1:-all}"
	case "$cmd" in
	all)
		configure_swap
		check_disk_space
		cleanup_old_k3s
		configure_cpu_governor
		configure_docker_logs
		configure_disk_encryption
		fix_dns
		limit_k3s_memory
		prepare_k3s_config
		restart_k3s_if_needed
		start_k3s
		download_k3s
		install_k3s
		upload_k3s_files
		pre_pull_system_images
		;;
	configure_swap) configure_swap ;;
	check_disk_space) check_disk_space ;;
	cleanup_old_k3s) cleanup_old_k3s ;;
	configure_cpu_governor) configure_cpu_governor ;;
	configure_docker_logs) configure_docker_logs ;;
	configure_disk_encryption) configure_disk_encryption ;;
	fix_dns) fix_dns ;;
	limit_k3s_memory) limit_k3s_memory ;;
	prepare_k3s_config) prepare_k3s_config ;;
	restart_k3s_if_needed) restart_k3s_if_needed ;;
	start_k3s) start_k3s ;;
	download_k3s) download_k3s ;;
	install_k3s) install_k3s ;;
	upload_k3s_files) upload_k3s_files ;;
	pre_pull_system_images) pre_pull_system_images ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
