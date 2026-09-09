#!/bin/bash
set -euo pipefail

VPS_HOST="${1:?Usage: $0 <vps_host> <vps_user> <ssh_private_key>}"
VPS_USER="${2:?Usage: $0 <vps_host> <vps_user> <ssh_private_key>}"
SSH_KEY="${3:?Usage: $0 <vps_host> <vps_user> <ssh_private_key>}"

ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i "$SSH_KEY" "$VPS_USER@$VPS_HOST" "sudo cat /var/lib/rancher/k3s/server/node-token" 2>/dev/null || echo ""
