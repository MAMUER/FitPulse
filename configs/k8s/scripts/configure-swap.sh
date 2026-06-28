#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
	echo "This script must be run as root" >&2
	exit 1
fi

SWAPFILE="/swapfile"
SWAPSIZE="3G"
SYSCTL_CONF="/etc/sysctl.d/99-swappiness.conf"

echo "-> Configuring ${SWAPSIZE} swap..."

if [[ -f "$SWAPFILE" ]]; then
	echo "-> Swap file already exists, reconfiguring..."
	swapoff -a || true
	rm -f "$SWAPFILE"
fi

fallocate -l "$SWAPSIZE" "$SWAPFILE"
chmod 600 "$SWAPFILE"
mkswap "$SWAPFILE"
swapon "$SWAPFILE"

if ! grep -q "$SWAPFILE none swap sw 0 0" /etc/fstab; then
	echo "$SWAPFILE none swap sw 0 0" | tee -a /etc/fstab
	echo "-> Swap entry added to /etc/fstab"
else
	echo "-> Swap entry already exists in /etc/fstab, skipping"
fi

# Перезаписываем файл с параметрами sysctl (вместо добавления)
cat >"$SYSCTL_CONF" <<EOF
vm.swappiness=10
vm.vfs_cache_pressure=50
EOF

sysctl -p "$SYSCTL_CONF"

echo "-> Swap configuration:"
free -h
