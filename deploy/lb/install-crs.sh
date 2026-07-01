#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
	echo "This script must be run as root or via sudo" >&2
	exit 1
fi

CRS_DIR="/opt/modsecurity-crs"
CRS_VERSION="v4.0.0"

echo "-> Installing OWASP CRS ${CRS_VERSION}..."

if ! command -v git &>/dev/null; then
	echo "-> git not found, installing..."
	apt-get update -qq
	apt-get install -y -qq git
fi

if [[ -d "${CRS_DIR}/.git" ]]; then
	echo "-> CRS directory exists, updating..."
	git -C "${CRS_DIR}" fetch --depth 1 origin "${CRS_VERSION}" || true
	git -C "${CRS_DIR}" checkout -f "${CRS_VERSION}" || true
else
	mkdir -p "${CRS_DIR}"
	git clone --depth 1 --branch "${CRS_VERSION}" \
		https://github.com/coreruleset/coreruleset.git "${CRS_DIR}"
fi

# Ensure proper permissions
chown -R root:root "${CRS_DIR}"
chmod -R 755 "${CRS_DIR}"

# Deploy ModSecurity runtime config
echo "-> Deploying ModSecurity configuration..."
mkdir -p /etc/nginx
cat >/etc/nginx/modsecurity.conf <<'MODSEC'
# ModSecurity runtime configuration (OWASP CRS v4)
<IfModule mod_security2.c>
    SecRuleEngine On
    SecRequestBodyAccess On
    SecResponseBodyAccess Off
    SecRequestBodyLimit 13107200
    SecRequestBodyNoFilesLimit 131072
    SecAuditEngine RelevantOnly
    SecAuditLogRelevantStatus "^(?:5|4(?!04))"
    SecAuditLogParts ABIJDEFHZ
    SecAuditLogType Serial
    SecAuditLog /var/log/nginx/modsecurity_audit.log
    SecDebugLog /var/log/nginx/modsecurity_debug.log
    SecDebugLogLevel 0

    SecRule REQUEST_URI "@beginsWith /health" \
        "id:1000001,phase:1,allow,ctl:ruleEngine=Off"

    <IfFile /opt/modsecurity-crs/crs-setup.conf>
        Include /opt/modsecurity-crs/crs-setup.conf
    </IfFile>
    <IfDirectory /opt/modsecurity-crs/rules>
        Include /opt/modsecurity-crs/rules/*.conf
    </IfDirectory>
</IfModule>
MODSEC
chmod 644 /etc/nginx/modsecurity.conf
mkdir -p /var/log/nginx
touch /var/log/nginx/modsecurity_audit.log
touch /var/log/nginx/modsecurity_debug.log
chmod 640 /var/log/nginx/modsecurity_audit.log
chmod 640 /var/log/nginx/modsecurity_debug.log

echo "✅ OWASP CRS ${CRS_VERSION} installed successfully"

