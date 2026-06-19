#!/bin/bash
set -euo pipefail
DUCKDNS_TOKEN=$(cat /etc/duckdns/token)
DUCKDNS_DOMAIN='fittpulse'
CURRENT_IP=$(curl -sf --max-time 5 https://api.ipify.org 2>/dev/null ||
curl -sf --max-time 5 https://ifconfig.me 2>/dev/null ||
echo '')
if [ -z "`$CURRENT_IP" ]; then
	echo "$(date): Failed to determine public IP"
	exit 1
fi
RESPONSE=$(curl -sf --max-time 10 \
	"https://www.duckdns.org/update?domains=${DUCKDNS_DOMAIN}&token=${DUCKDNS_TOKEN}&ip=${CURRENT_IP}" \
	2>/dev/null || echo 'ERROR')
if [ "$RESPONSE" = "OK" ]; then
	echo "$(date): DuckDNS updated — ${DUCKDNS_DOMAIN}.duckdns.org → ${CURRENT_IP}"
else
	echo "$(date): DuckDNS update FAILED — response: ${RESPONSE}"
	exit 1
fi