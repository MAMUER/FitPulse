#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
	echo "This script must be run as root" >&2
	exit 1
fi

K3S_STORAGE="/var/lib/rancher/k3s/storage"
ENCRYPTED_VOLUME="/dev/mapper/crypt-k3s"
MOUNT_POINT="/var/lib/rancher/k3s"

echo "-> Configuring host-level encrypted storage for k3s..."

# Check if we have an available block device for encryption (excluding root and swap)
AVAILABLE_DEVICE=""
for dev in /dev/vdb /dev/sdb /dev/nvme1n1; do
	if [[ -b "${dev}" ]] && ! mountpoint -q "${dev}"; then
		AVAILABLE_DEVICE="${dev}"
		break
	fi
done

if [[ -z "${AVAILABLE_DEVICE}" ]]; then
	echo "⚠️  No available block device found for dm-crypt."
	echo "   To enable fscrypt/dm-crypt:"
	echo "   1. Attach an additional volume to your VPS (e.g., /dev/vdb)"
	echo "   2. Run this script again"
	echo "   3. Or manually run:"
	echo "      cryptsetup luksFormat ${AVAILABLE_DEVICE:-/dev/vdb}"
	echo "      cryptsetup open ${AVAILABLE_DEVICE:-/dev/vdb} crypt-k3s"
	echo "      mkfs.ext4 ${ENCRYPTED_VOLUME}"
	echo "      mount ${ENCRYPTED_VOLUME} ${MOUNT_POINT}"
	echo ""
	echo "-> Creating encrypted storage directory without device..."
	# Ensure the base directory exists for k3s local-path-provisioner
	mkdir -p "${K3S_STORAGE}"
	chmod 755 "${K3S_STORAGE}"
	echo "✅ Storage directory prepared (unencrypted - use dedicated volume for encryption)"
	exit 0
fi

echo "-> Found available device: ${AVAILABLE_DEVICE}"

# Ensure cryptsetup is installed
if ! command -v cryptsetup &>/dev/null; then
	echo "-> Installing cryptsetup..."
	apt-get update -qq && apt-get install -y -qq cryptsetup >/dev/null 2>&1 || true
fi
if ! command -v cryptsetup &>/dev/null; then
	echo "⚠️  cryptsetup still not available, skipping encryption"
	mkdir -p "${K3S_STORAGE}"
	chmod 755 "${K3S_STORAGE}"
	exit 0
fi
# If already set up, skip
if cryptsetup status crypt-k3s &>/dev/null; then
	echo "Encrypted volume crypt-k3s already exists, skipping setup"
	exit 0
fi

# If device already has a filesystem (not LUKS), skip destructive format
if cryptsetup isLuks "${AVAILABLE_DEVICE}" 2>/dev/null; then
	: # LUKS device, proceed
elif blkid -o value -s TYPE "${AVAILABLE_DEVICE}" 2>/dev/null | grep -q .; then
	echo "Device ${AVAILABLE_DEVICE} already contains a filesystem, skipping LUKS format"
	mkdir -p "${K3S_STORAGE}"
	chmod 755 "${K3S_STORAGE}"
	exit 0
fi
echo "-> WARNING: This will format ${AVAILABLE_DEVICE} with LUKS encryption."
echo "   All data on ${AVAILABLE_DEVICE} will be destroyed!"
# В CI-режиме подтверждаем автоматически, иначе спрашиваем пользователя
if [[ "${CI:-}" == "true" ]] || [[ "${1:-}" == "--yes" ]]; then
	echo "-> Auto-confirming format (CI mode / --yes flag)"
else
	read -r -p "   Continue? (yes/no): " CONFIRM
	if [[ "${CONFIRM}" != "yes" ]]; then
		echo "-> Aborted by user"
		exit 1
	fi
fi

# Setup LUKS encryption
echo "-> Setting up LUKS encryption on ${AVAILABLE_DEVICE}..."
cryptsetup luksFormat "${AVAILABLE_DEVICE}"
cryptsetup open "${AVAILABLE_DEVICE}" crypt-k3s

# Create filesystem
echo "-> Creating ext4 filesystem..."
mkfs.ext4 -F "${ENCRYPTED_VOLUME}"

# Move existing k3s data if present
if [[ -d "${MOUNT_POINT}" ]] && [[ "$(ls -A "${MOUNT_POINT}")" ]]; then
	echo "-> Backing up existing k3s data..."
	mv "${MOUNT_POINT}" "${MOUNT_POINT}.bak"
fi

# Mount encrypted volume
mkdir -p "${MOUNT_POINT}"
mount "${ENCRYPTED_VOLUME}" "${MOUNT_POINT}"

# Restore data if backed up
if [[ -d "${MOUNT_POINT}.bak" ]]; then
	echo "-> Restoring k3s data..."
	mv "${MOUNT_POINT}.bak"/* "${MOUNT_POINT}/" 2>/dev/null || true
	rmdir "${MOUNT_POINT}.bak" 2>/dev/null || true
fi

# Add to fstab for persistence
echo "${ENCRYPTED_VOLUME} ${MOUNT_POINT} ext4 defaults 0 2" >>/etc/fstab

echo "✅ Encrypted storage configured at ${MOUNT_POINT}"
echo "   To unlock on boot, add to /etc/crypttab:"
echo "   crypt-k3s ${AVAILABLE_DEVICE} none luks"

# If device already has a filesystem (not LUKS), skip destructive format
if cryptsetup isLuks "${AVAILABLE_DEVICE}" 2>/dev/null; then
	: # LUKS device, proceed
elif blkid -o value -s TYPE "${AVAILABLE_DEVICE}" 2>/dev/null | grep -q .; then
	echo "Device ${AVAILABLE_DEVICE} already contains a filesystem, skipping LUKS format"
	mkdir -p "${K3S_STORAGE}"
	chmod 755 "${K3S_STORAGE}"
	exit 0
fi

echo "-> WARNING: This will format ${AVAILABLE_DEVICE} with LUKS encryption."
echo "   All data on ${AVAILABLE_DEVICE} will be destroyed!"
# В CI-режиме подтверждаем автоматически, иначе спрашиваем пользователя
if [[ "${CI:-}" == "true" ]] || [[ "${1:-}" == "--yes" ]]; then
	echo "-> Auto-confirming format (CI mode / --yes flag)"
else
	read -r -p "   Continue? (yes/no): " CONFIRM
	if [[ "${CONFIRM}" != "yes" ]]; then
		echo "-> Aborted by user"
		exit 1
	fi
fi

# Setup LUKS encryption
echo "-> Setting up LUKS encryption on ${AVAILABLE_DEVICE}..."
cryptsetup luksFormat "${AVAILABLE_DEVICE}"
cryptsetup open "${AVAILABLE_DEVICE}" crypt-k3s

# Create filesystem
echo "-> Creating ext4 filesystem..."
mkfs.ext4 -F "${ENCRYPTED_VOLUME}"

# Move existing k3s data if present
if [[ -d "${MOUNT_POINT}" ]] && [[ "$(ls -A "${MOUNT_POINT}")" ]]; then
	echo "-> Backing up existing k3s data..."
	mv "${MOUNT_POINT}" "${MOUNT_POINT}.bak"
fi

# Mount encrypted volume
mkdir -p "${MOUNT_POINT}"
mount "${ENCRYPTED_VOLUME}" "${MOUNT_POINT}"

# Restore data if backed up
if [[ -d "${MOUNT_POINT}.bak" ]]; then
	echo "-> Restoring k3s data..."
	mv "${MOUNT_POINT}.bak"/* "${MOUNT_POINT}/" 2>/dev/null || true
	rmdir "${MOUNT_POINT}.bak" 2>/dev/null || true
fi

# Add to fstab for persistence
echo "${ENCRYPTED_VOLUME} ${MOUNT_POINT} ext4 defaults 0 2" >>/etc/fstab

echo "✅ Encrypted storage configured at ${MOUNT_POINT}"
echo "   To unlock on boot, add to /etc/crypttab:"
echo "   crypt-k3s ${AVAILABLE_DEVICE} none luks"
