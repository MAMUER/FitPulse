#!/bin/bash
set -euo pipefail
sudo mkdir -p /etc/systemd/system/k3s.service.d
cat > /tmp/memory-limit.conf << 'EOFCONF'
[Service]
MemoryMax=1G
MemoryHigh=800M
EOFCONF
sudo mv /tmp/memory-limit.conf /etc/systemd/system/k3s.service.d/memory-limit.conf
sudo mv /tmp/k3s-memory-watchdog.sh /usr/local/bin/k3s-memory-watchdog.sh
sudo chmod +x /usr/local/bin/k3s-memory-watchdog.sh
(sudo crontab -l 2>/dev/null | grep -v k3s-memory-watchdog; echo '*/5 * * * * /usr/local/bin/k3s-memory-watchdog.sh >> /var/log/k3s-watchdog.log 2>&1') | sudo crontab -
sudo systemctl daemon-reload
sudo systemctl restart k3s
echo 'k3s memory watchdog configured'
