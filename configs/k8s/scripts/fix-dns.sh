#!/usr/bin/env bash
# configs/k8s/scripts/fix-dns.sh
# Упрощенный и максимально надежный скрипт фиксации DNS
set -euo pipefail

K3S_CONFIG_DIR="/etc/rancher/k3s"
K3S_RESOLV_CONF="${K3S_CONFIG_DIR}/resolv.conf"
SYSTEMD_RESOLVED_DROPIN="/etc/systemd/resolved.conf.d/99-fitpulse-dns.conf"
TEMP_RESOLV_CONF=""

TEST_DOMAINS=("github.com" "ghcr.io" "registry-1.docker.io" "acme-v02.api.letsencrypt.org")

log_info() { echo "[INFO]  $*"; }
log_ok() { echo "[OK]    $*"; }

cleanup() {
	if [[ -n "$TEMP_RESOLV_CONF" ]]; then
		rm -f "$TEMP_RESOLV_CONF"
	fi
}
trap cleanup EXIT

echo "============================================"
echo "  VPS DNS Fix Script"
echo "============================================"

# CoreDNS forwards through the resolver file inherited by kubelet-managed pods.
# Keep systemd-resolved's loopback stub out of that file; CoreDNS sees 127.0.0.53
# as its own loopback and exits with plugin/loop.
if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet systemd-resolved; then
	mkdir -p "$(dirname "$SYSTEMD_RESOLVED_DROPIN")"
	cat >"$SYSTEMD_RESOLVED_DROPIN" <<'EOF'
[Resolve]
DNS=1.1.1.1 8.8.8.8
FallbackDNS=1.0.0.1 8.8.4.4
Domains=~.
EOF
	systemctl restart systemd-resolved
	log_ok "systemd-resolved configured to route DNS through external upstreams"
fi

mkdir -p "$K3S_CONFIG_DIR"
TEMP_RESOLV_CONF=$(mktemp "${K3S_CONFIG_DIR}/resolv.conf.XXXXXX")
cat >"$TEMP_RESOLV_CONF" <<'EOF'
nameserver 1.1.1.1
nameserver 8.8.8.8
nameserver 8.8.4.4
options timeout:2 attempts:3
EOF

chmod 644 "$TEMP_RESOLV_CONF"
mv "$TEMP_RESOLV_CONF" "$K3S_RESOLV_CONF"
TEMP_RESOLV_CONF=""
log_info "K3s resolver file:"
cat "$K3S_RESOLV_CONF"

log_info "Verifying host DNS resolution..."
failed=0
for domain in "${TEST_DOMAINS[@]}"; do
	if getent ahostsv4 "$domain" >/dev/null 2>&1; then
		log_ok "$domain"
	else
		echo "::error::DNS resolution failed for $domain" >&2
		failed=$((failed + 1))
	fi
done

if [[ "$failed" -ne 0 ]]; then
	echo "::error::DNS is still unavailable; refusing to continue cluster provisioning" >&2
	exit 1
fi

log_ok "K3s and host DNS resolver configuration is ready"
