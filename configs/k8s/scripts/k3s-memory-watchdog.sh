#!/bin/bash
set -euo pipefail
K3S_PID=$(pgrep -f 'k3s server' || echo '')
if [ -z "$K3S_PID" ]; then
    echo "$(date): k3s not running, skipping"
    exit 0
fi
K3S_RSS=$(ps -o rss= -p "$K3S_PID" 2>/dev/null || echo 0)
LIMIT_KB=600000
if [ "$K3S_RSS" -gt "$LIMIT_KB" ]; then
    echo "$(date): k3s memory ${K3S_RSS}KB exceeds limit ${LIMIT_KB}KB, restarting..." | logger -t k3s-watchdog
    systemctl restart k3s
    sleep 10
    NEW_RSS=$(ps -o rss= -p $(pgrep -f 'k3s server' || echo 0) 2>/dev/null || echo 0)
    echo "$(date): k3s restarted, new memory: ${NEW_RSS}KB" | logger -t k3s-watchdog
fi
