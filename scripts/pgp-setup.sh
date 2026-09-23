#!/usr/bin/env bash
set -euo pipefail

# PGP Key Setup Script for FitPulse
# Generates an Ed25519 signing key + Curve25519 encryption subkey
# Publishes to WKD and keys.openpgp.org

KEY_NAME="${1:-FitPulse Security}"
KEY_EMAIL="${2:-mihnikolaenko12@yandex.ru}"
KEY_EXPIRE="${3:-1y}"
KEY_DIR="${4:-./pgp-keys}"

mkdir -p "$KEY_DIR"

echo "=== FitPulse PGP Key Setup ==="
echo "Name:  $KEY_NAME"
echo "Email: $KEY_EMAIL"
echo "Expire: $KEY_EXPIRE"
echo ""

# Generate key
gpg --batch --gen-key <<EOF
%no-protection
Key-Type: ECC
Key-Curve: curve25519
Subkey-Type: ECC
Subkey-Curve: curve25519
Name-Real: $KEY_NAME
Name-Email: $KEY_EMAIL
Expire-Date: $KEY_EXPIRE
%commit
EOF

KEY_ID=$(gpg --list-keys --with-colons "$KEY_EMAIL" | grep '^fpr:' | head -1 | cut -d: -f10)
FINGERPRINT=$(gpg --fingerprint "$KEY_EMAIL" | grep 'Fingerprint' | sed 's/.*= //' | tr -d ' ')

echo ""
echo "=== Key Generated ==="
echo "Key ID:       $KEY_ID"
echo "Fingerprint:  $FINGERPRINT"
echo ""

# Export public key
gpg --armor --export "$KEY_ID" > "$KEY_DIR/public-key.asc"
gpg --armor --export-secret-keys "$KEY_ID" > "$KEY_DIR/private-key.asc"

# Export for WKD
gpg --export "$KEY_ID" > "$KEY_DIR/openpgpkey_${KEY_ID:0:40}.asc"

echo "=== Keys exported to $KEY_DIR ==="
echo ""
echo "=== Publishing to keys.openpgp.org ==="
gpg --keyserver hkps://keys.openpgp.org --send-keys "$KEY_ID"

echo ""
echo "=== Setting up WKD ==="
echo "Upload $KEY_DIR/openpgpkey_${KEY_ID:0:40}.asc to:"
echo "  https://fittpulse.ru/.well-known/openpgpkey/hu/${KEY_ID:0:40}"
echo ""
echo "=== Next steps ==="
echo "1. Update SECURITY.md and BUG_BOUNTY_SCOPE.md with fingerprint: $FINGERPRINT"
echo "2. Upload WKD file to your web server"
echo "3. Verify: gpg --auto-key-locate clear,wkd --locate-keys $KEY_EMAIL"
echo ""
echo "=== Security reminder ==="
echo "- Store private key securely (hardware token recommended)"
echo "- Backup private key to encrypted storage"
echo "- Rotate keys annually"
echo "- Never commit private key to git"
