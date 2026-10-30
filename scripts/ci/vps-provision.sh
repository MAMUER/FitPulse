#!/bin/bash
set -euo pipefail

configure_swap() {
	echo "🔄 Проверка доступности VPS по SSH (попыток: 3)..."

	local max_attempts=3
	local attempt=1
	local connected=false

	while [[ $attempt -le $max_attempts ]]; do
		echo "➡️ Попытка $attempt из $max_attempts..."
		# Увеличили ConnectTimeout до 15 и добавили -v для отладки, если понадобится
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

	# Если swap уже настроен — пропускаем
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

# 1. Останавливаем и принудительно убиваем все процессы k3s
echo '-> Stopping and killing k3s processes...'
systemctl stop k3s k3s-agent 2>/dev/null || true
pkill -9 -f 'k3s' 2>/dev/null || true

# 2. Запускаем официальный скрипт очистки сетей (удаляет cni0, flannel.1 и т.д.)
if [ -f /usr/local/bin/k3s-killall.sh ]; then
    echo '-> Running k3s-killall.sh to clean network interfaces...'
    /usr/local/bin/k3s-killall.sh || true
fi

# 3. Полностью удаляем все директории состояния k3s, CNI и kubelet
echo '-> Removing k3s, CNI, and kubelet state directories...'
rm -rf /var/lib/rancher/k3s
rm -rf /etc/rancher/k3s
rm -rf /var/lib/cni
rm -rf /var/lib/kubelet
rm -rf /run/k3s
rm -rf /run/flannel

# 4. Удаляем бинарные файлы и systemd-юниты
echo '-> Removing k3s binaries and systemd units...'
rm -f /usr/local/bin/k3s
rm -f /usr/local/bin/kubectl
rm -f /usr/local/bin/crictl
rm -f /usr/local/bin/ctr
rm -f /etc/systemd/system/k3s.service
rm -f /etc/systemd/system/k3s.service.env
rm -f /etc/systemd/system/k3s-agent.service
rm -f /etc/systemd/system/multi-user.target.wants/k3s.service
rm -f /etc/systemd/system/multi-user.target.wants/k3s-agent.service

# 5. Перезагружаем демон systemd и сбрасываем ошибки
echo '-> Reloading systemd daemon and resetting failed states...'
systemctl daemon-reload
systemctl reset-failed

# 6. Очищаем директорию local-path-provisioner (гарантирует чистые PVC при новой установке)
echo '-> Cleaning up local-path-provisioner directory...'
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

	# 1. Создаем директорию с правами root
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo mkdir -p /etc/rancher/k3s"

	# 2. Копируем файл во временную папку пользователя (где есть права на запись)
	./scripts/ssh-retry.sh scp /tmp/k3s-config-remote.yaml "${VPS_USER}@${VPS_HOST}:/tmp/k3s-config-remote.yaml"

	# 3. Перемещаем файл в системную директорию с правами root и ставим правильные права доступа
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "sudo cp /tmp/k3s-config-remote.yaml /etc/rancher/k3s/config.yaml && sudo chmod 600 /etc/rancher/k3s/config.yaml"

	# 4. Проверяем, что файл записался
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
		# Применяем глобальный фикс MTU для containerd
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
echo '=== Cluster status ==='
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
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
