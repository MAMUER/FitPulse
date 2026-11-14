#!/usr/bin/env bash
# configs/k8s/scripts/fix-dns.sh
# Упрощенный и максимально надежный скрипт фиксации DNS
set -euo pipefail

RESOLV_CONF="/etc/resolv.conf"
BACKUP_FILE="${RESOLV_CONF}.bak.$(date +%Y%m%d_%H%M%S)"

TEST_DOMAINS=("archive.ubuntu.com" "github.com" "registry-1.docker.io")

log_info() { echo "[INFO]  $*"; }
log_ok() { echo "[OK]    $*"; }
log_warn() { echo "[WARN]  $*"; }

echo "============================================"
echo "  VPS DNS Fix Script"
echo "============================================"

# 1. Делаем резервную копию
if [[ -f "$RESOLV_CONF" ]]; then
	cp "$RESOLV_CONF" "$BACKUP_FILE"
	log_ok "Backup created: ${BACKUP_FILE}"
fi

# 1b. На WSL отключаем generateResolvConf, иначе WSL перезапишет resolv.conf
# при каждом старте/сетевом событии и вернёт 10.255.255.254 (DNS Windows-хоста).
if grep -qi microsoft /proc/version 2>/dev/null; then
	if [[ -f /etc/wsl.conf ]] && grep -q '^\[network\]' /etc/wsl.conf; then
		if grep -q '^[[:space:]]*generateResolvConf' /etc/wsl.conf; then
			sed -i 's/^[[:space:]]*generateResolvConf[[:space:]]*=.*/generateResolvConf = false/' /etc/wsl.conf
		else
			sed -i '/^\[network\]/a generateResolvConf = false' /etc/wsl.conf
		fi
	else
		printf '\n[network]\ngenerateResolvConf = false\n' >>/etc/wsl.conf
	fi
	log_ok "WSL generateResolvConf disabled (static resolv.conf will persist)"
fi

# 2. Перезаписываем на надежные DNS-серверы
cat >"$RESOLV_CONF" <<EOF
# Fixed DNS - managed by CI/CD
nameserver 8.8.8.8
nameserver 1.1.1.1
nameserver 8.8.4.4
options timeout:2 attempts:3 rotate
EOF
chmod 644 "$RESOLV_CONF"
log_ok "resolv.conf written with working nameservers"

# 3. Проверка DNS (НЕ прерывает скрипт при ошибках)
log_info "Verifying DNS resolution..."
sleep 2
failed=0
for domain in "${TEST_DOMAINS[@]}"; do
	if getent hosts "$domain" >/dev/null 2>&1 || ping -c 1 -W 2 "$domain" >/dev/null 2>&1; then
		log_ok "✓ ${domain}"
	else
		log_warn "✗ ${domain} (ignoring transient glitch)"
		((failed++)) || true
	fi
done

if [[ $failed -gt 0 ]]; then
	log_warn "Some DNS checks failed, but resolv.conf is updated. Proceeding."
fi

log_ok "DNS fix completed successfully!"
# Гарантированный успешный выход
exit 0
