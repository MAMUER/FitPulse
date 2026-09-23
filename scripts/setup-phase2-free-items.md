# Setup Instructions for Phase 2 Free Items

## 1. PGP Key + WKD Setup

### Step 1: Generate PGP key locally

```bash
# Install GnuPG if not present
# macOS: brew install gnupg
# Ubuntu/Debian: sudo apt-get install gnupg
# Windows: https://gnupg.org/download/

# Generate Ed25519 key
gpg --batch --gen-key <<EOF
%no-protection
Key-Type: ECC
Key-Curve: curve25519
Subkey-Type: ECC
Subkey-Curve: curve25519
Name-Real: FitPulse Security
Name-Email: mihnikolaenko12@yandex.ru
Expire-Date: 1y
%commit
EOF
```

### Step 2: Extract fingerprint and key ID

```bash
# Get fingerprint
gpg --fingerprint mihnikolaenko12@yandex.ru

# Output example:
# Fingerprint: ABCD 1234 EF56 7890 ABCD 1234 EF56 7890 ABCD 1234

# Get key ID (last 16 chars of fingerprint without spaces)
gpg --list-keys --with-colons mihnikolaenko12@yandex.ru | grep '^pub:' | head -1 | cut -d: -f5
```

### Step 3: Update documentation

Replace `[FILL: YOUR_PGP_FINGERPRINT]` and `[FILL: YOUR_PGP_KEY_ID]` in:
- `SECURITY.md`
- `BUG_BOUNTY_SCOPE.md`

With the real values from Step 2.

### Step 4: Publish public key

```bash
# Export public key
gpg --armor --export mihnikolaenko12@yandex.ru > public-key.asc

# Upload to keys.openpgp.org
gpg --keyserver hkps://keys.openpgp.org --send-keys <KEY_ID>

# Verify
gpg --auto-key-locate clear,wkd --locate-keys mihnikolaenko12@yandex.ru
```

### Step 5: Set up WKD

Upload `public-key.asc` to your web server at:
```
https://fittpulse.ru/.well-known/openpgpkey/hu/<KEY_ID_PREFIX>
```

Where `<KEY_ID_PREFIX>` is the first 40 characters of the key ID.

### Step 6: Store private key securely

**DO NOT commit the private key to git.**

Store it in:
- Hardware token (YubiKey, Nitrokey) — recommended
- Encrypted backup (Vault, 1Password, etc.)
- GitHub Secrets (for CI) as `PGP_PRIVATE_KEY` and `PGP_PASSPHRASE`

### Step 7: GitHub Secrets

Add to GitHub Secrets (Settings → Secrets and variables → Actions):
- `PGP_PRIVATE_KEY`: ASCII-armored private key
- `PGP_PASSPHRASE`: passphrase for the private key (if any)

## 2. Cloudflare Turnstile Setup

### Step 1: Create Turnstile site

1. Go to https://dash.cloudflare.com/profile/turnstile
2. Create new site:
   - Site name: `FitPulse`
   - Domain: `fittpulse.ru`
   - Mode: `Managed` (recommended for 152-ФЗ compliance)
3. Copy **Site Key** and **Secret Key**

### Step 2: Update ConfigMap

Replace `[REPLACE_WITH_YOUR_SITE_KEY]` in `configs/k8s/base/captcha/configmap.yaml` with the real Site Key.

**The Secret Key must NOT be in the ConfigMap.** It will be stored in Vault and injected via External Secrets.

### Step 3: Store Secret Key in Vault

```bash
# Store in Vault
vault kv put secret/data/captcha \
  CLOUDFLARE_TURNSTILE_SECRET_KEY=<your-secret-key>

# Verify
vault kv get secret/data/captcha
```

### Step 4: Deploy

```bash
kubectl apply -f configs/k8s/base/captcha/
kubectl rollout restart deployment/gateway -n fitness-platform-production
```

## 3. DVC Initialization

### Step 1: Install DVC

```bash
pip install dvc
```

### Step 2: Run setup script

```bash
bash scripts/dvc-setup.sh
```

### Step 3: Configure remote storage (production)

```bash
# For MinIO/S3-compatible storage
dvc remote add -d production s3://fitpulse-ml-models
dvc remote modify production endpointurl http://minio:9000
dvc remote modify production access_key_id <your-access-key>
dvc remote modify production secret_access_key <your-secret-key>
```

### Step 4: Run pipeline

```bash
# Test pipeline
dvc repro

# Push to remote
dvc push
```

**DO NOT commit DVC remote credentials to git.** Use environment variables or Vault.

## 4. Terraform Setup (Preparation Only)

### Prerequisites

- Terraform >= 1.5.0
- Cloud provider CLI (yc, aws, etc.)
- SSH key pair
- S3-compatible backend (MinIO)

### Step 1: Configure backend

```bash
cd terraform
terraform init
```

### Step 2: Set variables

```bash
# Create terraform.tfvars (DO NOT commit)
cat > terraform.tfvars <<EOF
yandex_cloud_token = "your-yandex-token"
ssh_public_key     = "ssh-rsa AAAA..."
k3s_token          = "your-k3s-token"
postgres_user      = "fitpulse"
postgres_password  = "your-password"
vault_root_token   = "your-vault-token"
EOF
```

**DO NOT commit `terraform.tfvars` to git.** Add to `.gitignore`.

### Step 3: Plan and apply (when ready)

```bash
terraform plan
terraform apply
```

## Security Checklist

- [ ] PGP private key stored in hardware token or encrypted backup
- [ ] Cloudflare Turnstile Secret Key in Vault, NOT in git
- [ ] Terraform variables in `.gitignore`, not in git
- [ ] DVC remote credentials in environment variables or Vault
- [ ] No secrets in ConfigMaps
- [ ] GitHub Secrets used for CI/CD

## GitHub Secrets Required

Add these to GitHub Secrets (Settings → Secrets and variables → Actions):

| Secret | Description |
| --- | --- |
| `PGP_PRIVATE_KEY` | ASCII-armored PGP private key |
| `PGP_PASSPHRASE` | PGP key passphrase (if any) |
| `CLOUDFLARE_TURNSTILE_SECRET_KEY` | Cloudflare Turnstile secret key |
| `VAULT_ADDR` | Vault address |
| `VAULT_TOKEN` | Vault token (for CI) |
