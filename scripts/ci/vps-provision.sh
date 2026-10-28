#!/bin/bash
set -euo pipefail

configure_swap() {
	echo "🔄 Проверка доступности VPS по SSH (попыток: 3)..."

	local max_attempts=3
	local attempt=1
	local connected=false

	while [ $attempt -le $max_attempts ]; do
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

	if [ "$connected" = false ]; then
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
if [ ! -f /etc/systemd/system/k3s.service ] && [ ! -f /usr/local/bin/k3s ]; then
	echo '? k3s is not installed yet, skipping restart (will be installed in next steps)'
	exit 0
fi
LOCAL_HASH=\$(sudo sha256sum /etc/rancher/k3s/config.yaml 2>/dev/null | awk '{print \$1}' || echo 'none')
SAVED_HASH=\$(sudo cat /etc/rancher/k3s/.config-hash 2>/dev/null || echo 'none')
if [ \"\$LOCAL_HASH\" != \"\$SAVED_HASH\" ]; then
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
		if [ \"\$i\" -eq 90 ]; then
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
	if [ \"\$i\" -eq 120 ]; then
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
echo '-> Installing k3s from local files...'

# 1. Проверяем, что файл действительно существует перед перемещением
if [ ! -f /tmp/k3s-install/k3s ]; then
	echo '❌ ERROR: /tmp/k3s-install/k3s not found! Download step may have failed.'
	exit 1
fi

# 2. Перемещаем бинарный файл
sudo mkdir -p /usr/local/bin
sudo mv /tmp/k3s-install/k3s /usr/local/bin/k3s
sudo chmod +x /usr/local/bin/k3s

# 3. Запускаем установочный скрипт, ЯВНО передавая переменную окружения через sudo env
sudo chmod +x /tmp/k3s-install/install.sh
sudo env INSTALL_K3S_SKIP_DOWNLOAD=true INSTALL_K3S_BIN_DIR=/usr/local/bin /tmp/k3s-install/install.sh server --write-kubeconfig-mode 644

echo '✅ k3s installed successfully'
# Очищаем временные файлы только в случае успеха
rm -rf /tmp/k3s-install
"
}

upload_k3s_files() {
	echo "-> Создаем директорию для файлов k3s..."
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "mkdir -p /tmp/k3s-install"

	echo "-> Uploading k3s binary via scp (with retries)..."
	./scripts/ssh-retry.sh scp /tmp/k3s "${VPS_USER}@${VPS_HOST}:/tmp/k3s-install/k3s"

	echo "-> Uploading install.sh via scp (with retries)..."
	./scripts/ssh-retry.sh scp /tmp/install.sh "${VPS_USER}@${VPS_HOST}:/tmp/k3s-install/install.sh"

	echo "-> Setting executable permissions..."
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "chmod +x /tmp/k3s-install/k3s /tmp/k3s-install/install.sh"
}
limit_k3s_memory() {
	echo "-> Cleaning up stuck pods from previous runs..."
	./scripts/ssh-retry.sh ssh "${VPS_USER}@${VPS_HOST}" "
		sudo k3s kubectl delete pods --all -n kube-system --force --grace-period=0 2>/dev/null || true
		sudo k3s kubectl delete pods --all -n local-path-storage --force --grace-period=0 2>/dev/null || true
		echo 'Stuck pods cleared in kube-system and local-path-storage'
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
		download_k3s
		install_k3s
		upload_k3s_files
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
	download_k3s) download_k3s ;;
	install_k3s) install_k3s ;;
	upload_k3s_files) upload_k3s_files ;;
	limit_k3s_memory) limit_k3s_memory ;;
	*)
		echo "Unknown function: $cmd"
		exit 1
		;;
	esac
}

main "$@"
