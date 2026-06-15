#!/bin/bash
# DuckDNS IP updater — run via cron every 5 minutes
# Usage: ./duckdns-update.sh
# Requires: DUCKDNS_TOKEN environment variable or .env file

set -euo pipefail

# Load token from environment or file
DUCKDNS_TOKEN="${DUCKDNS_TOKEN:-}"
DUCKDNS_DOMAIN="${DUCKDNS_DOMAIN:-fittpulse}"

if [ -z "$DUCKDNS_TOKEN" ]; then
	if [ -f /etc/duckdns/token ]; then
		DUCKDNS_TOKEN=$(cat /etc/duckdns/token)
	else
		echo "ERROR: DUCKDNS_TOKEN not set and /etc/duckdns/token not found"
		exit 1
	fi
fi

# Get current public IP
CURRENT_IP=$(curl -sf --max-time 5 https://api.ipify.org 2>/dev/null ||
	curl -sf --max-time 5 https://ifconfig.me 2>/dev/null ||
	curl -sf --max-time 5 https://icanhazip.com 2>/dev/null ||
	echo "")

if [ -z "$CURRENT_IP" ]; then
	echo "$(date): Failed to determine public IP"
	exit 1
fi

# Update DuckDNS
RESPONSE=$(curl -sf --max-time 10 \
	"https://www.duckdns.org/update?domains=${DUCKDNS_DOMAIN}&token=${DUCKDNS_TOKEN}&ip=${CURRENT_IP}" \
	2>/dev/null || echo "ERROR")

if [ "$RESPONSE" = "OK" ]; then
	echo "$(date): DuckDNS updated — ${DUCKDNS_DOMAIN}.duckdns.org → ${CURRENT_IP}"
else
	echo "$(date): DuckDNS update FAILED — response: ${RESPONSE}"
	exit 1
fi
