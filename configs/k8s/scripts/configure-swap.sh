#!/usr/bin/env bash
# configs/k8s/scripts/configure-swap.sh
# ============================================================================
# Configure 3GB Swap on VPS
# ----------------------------------------------------------------------------
# Creates and enables a swap file if it does not already exist, and
# configures sysctl + /etc/fstab for persistence.
# ============================================================================
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
	echo "This script must be run as root" >&2
	exit 1
fi

SWAPFILE="/swapfile"
SWAPSIZE="3G"

echo "-> Configuring ${SWAPSIZE} swap..."

if [[ -f "$SWAPFILE" ]]; then
	echo "-> Swap file already exists, reconfiguring..."
	sudo swapoff -a || true
	sudo rm -f "$SWAPFILE"
fi

sudo fallocate -l "$SWAPSIZE" "$SWAPFILE"
sudo chmod 600 "$SWAPFILE"
sudo mkswap "$SWAPFILE"
sudo swapon "$SWAPFILE"

if ! grep -q "$SWAPFILE none swap sw 0 0" /etc/fstab; then
	echo "$SWAPFILE none swap sw 0 0" | sudo tee -a /etc/fstab
	echo "-> Swap entry added to /etc/fstab"
else
	echo "-> Swap entry already exists in /etc/fstab, skipping"
fi

echo "vm.swappiness=10" | sudo tee -a /etc/sysctl.d/99-swappiness.conf
echo "vm.vfs_cache_pressure=50" | sudo tee -a /etc/sysctl.d/99-swappiness.conf
sudo sysctl -p /etc/sysctl.d/99-swappiness.conf

echo "-> Swap configuration:"
free -h
