# PGP Web Key Directory (WKD) Setup

## Overview

WKD (Web Key Directory) allows email clients (GnuPG, Thunderbird, etc.) to automatically discover PGP public keys via HTTPS.

## Directory Structure

```
/var/www/fittpulse.ru/.well-known/openpgpkey/
└── hu/
    └── <KEY_ID_PREFIX>
```

Where `<KEY_ID_PREFIX>` is the first 40 characters of the key ID (without the `0x` prefix).

## Setup Steps

### 1. Generate key

```bash
bash scripts/pgp-setup.sh
```

### 2. Export for WKD

```bash
gpg --export "$KEY_ID" > openpgpkey_${KEY_ID:0:40}.asc
```

### 3. Upload to web server

```bash
# Create directory
mkdir -p /var/www/fittpulse.ru/.well-known/openpgpkey/hu

# Upload file
scp openpgpkey_${KEY_ID:0:40}.asc user@server:/var/www/fittpulse.ru/.well-known/openpgpkey/hu/

# Set permissions
chmod 644 /var/www/fittpulse.ru/.well-known/openpgpkey/hu/openpgpkey_${KEY_ID:0:40}.asc
```

### 4. Configure nginx

```nginx
location ^~ /.well-known/openpgpkey/ {
    root /var/www/fittpulse.ru;
    default_type application/octet-stream;
    add_header Content-Disposition "attachment";
}
```

### 5. Verify

```bash
gpg --auto-key-locate clear,wkd --locate-keys mihnikolaenko12@yandex.ru
```

## Automation (GitHub Actions)

See `.github/workflows/pgp-wkd.yml` for automated publishing.

## Security Considerations

- WKD serves **public keys only** — no sensitive data
- HTTPS is mandatory (WKD over HTTP is deprecated)
- Key rotation: update WKD file when key expires
- Backup: store private key in encrypted backup (not in WKD)
