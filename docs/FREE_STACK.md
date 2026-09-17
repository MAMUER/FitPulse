# FitPulse — 100% Free/Open-Source Stack Architecture

> **Objective:** Run the entire platform on zero-license-cost software.
> **Cost:** Only infrastructure (VPS ~€4/mo). No SaaS, no vendor lock-in.
> **Key constraint:** No credit card required for any component.

---

## Stack Components (All Free, No Card Required)

| Layer | Component | License | Cost | Card Required? |
| ------- | ----------- | --------- | ------ | ---------------- |
| **IaC** | Terraform OSS | BSL-1.1 | Free | ❌ No |
| **State Backend** | MinIO (S3-compatible) | AGPL-3.0 | Free | ❌ No |
| **VPS** | Existing VPS (k3s already running) | — | Already paid | ❌ No |
| **Kubernetes** | k3s | Apache-2.0 | Free | ❌ No |
| **Secrets** | External Secrets Operator + PostgreSQL | Apache-2.0 | Free | ❌ No |
| **DNS** | DuckDNS / Cloudflare | — | Free | ❌ No |
| **CI/CD** | GitHub Actions | — | Free tier | ❌ No |
| **Monitoring** | Prometheus + Grafana + Alertmanager | Apache-2.0 | Free | ❌ No |
| **Logging** | Loki + Fluent Bit | AGPL-3.0 / MIT | Free | ❌ No |
| **Tracing** | Jaeger + OpenTelemetry | Apache-2.0 | Free | ❌ No |
| **Database** | PostgreSQL 18 + pgsodium | PostgreSQL License | Free | ❌ No |
| **Cache** | Valkey (Redis fork) | BSD-3-Clause | Free | ❌ No |
| **MQ** | RabbitMQ | MPL-2.0 | Free | ❌ No |
| **Container** | Docker / BuildKit | Apache-2.0 | Free | ❌ No |

---

## Architecture Diagram

```text
┌─────────────────────────────────────────────────────────────────┐
│                        INTERNET / USERS                         │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTPS
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  DuckDNS / Cloudflare (free DNS, no card required)              │
│  fittpulse.duckdns.org → <VPS_IP>                              │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Existing VPS (k3s already running)                             │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  k3s cluster (1 node, production)                         │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │  MinIO (S3-compatible)                              │  │  │
│  │  │  - Bucket: fitpulse-terraform-state                 │  │  │
│  │  │  - Bucket: fitpulse-ml-models                       │  │  │
│  │  │  - Bucket: fitpulse-db-backups                      │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │  PostgreSQL 18 + pgsodium                           │  │  │
│  │  │  - Table: external_secrets (ESO backend)            │  │  │
│  │  │  - Table: biometric_metrics, training_plans, etc.   │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  │                                                           │  │
│  │  Namespace: fitness-platform-production                   │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────────┐ │  │
│  │  │  Gateway     │  │ User Service │  │  Biometric Svc  │ │  │
│  │  │  (Chi + mTLS)│  │  (gRPC)      │  │  (gRPC)         │ │  │
│  │  └──────────────┘  └──────────────┘  └────────────────┘ │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────────┐ │  │
│  │  │ Training Svc │  │ Classifier   │  │ ML Generator   │ │  │
│  │  │  (gRPC)      │  │  (HTTP)      │  │  (Python/FastAPI)│ │  │
│  │  └──────────────┘  └──────────────┘  └────────────────┘ │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────────┐ │  │
│  │  │ Device Aggr  │  │ Open Wearables│  │ Valkey         │ │  │
│  │  │  (HTTP)      │  │  Frontend     │  │  (Cache/Session)│ │  │
│  │  └──────────────┘  └──────────────┘  └────────────────┘ │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────────┐ │  │
│  │  │ RabbitMQ     │  │ Prometheus   │  │ Grafana        │ │  │
│  │  │  (Queue)     │  │  (Metrics)   │  │  (Dashboards)  │ │  │
│  │  └──────────────┘  └──────────────┘  └────────────────┘ │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────────┐ │  │
│  │  │ Alertmanager │  │ Loki         │  │ Jaeger         │ │  │
│  │  │  (Alerts)    │  │  (Logs)      │  │  (Traces)      │ │  │
│  │  └──────────────┘  └──────────────┘  └────────────────┘ │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Secrets Management — Free Stack (No AWS, No Card)

### Current State

- External Secrets Operator already deployed
- AWS Secrets Manager is current backend (paywalled after first year)

### Target State (100% free)

```text
PostgreSQL table `external_secrets`
        ↓
External Secrets Operator (PostgreSQL provider)
        ↓
Kubernetes Secret `app-secrets`
        ↓
Приложения (env vars from mounted secret)
```

---

## How to Get Each Secret (No Credit Card Required)

### HCLOUD_TOKEN — NOT NEEDED ANYMORE

Terraform no longer provisions VPS. It manages only existing k3s resources.

### MINIO_ACCESS_KEY / MINIO_SECRET_KEY

**Source:** You set these yourself when deploying MinIO.

```bash
# Option 1: Generate random values
openssl rand -base64 12   # ACCESS_KEY
openssl rand -base64 24   # SECRET_KEY

# Option 2: Use fixed values (change in production!)
MINIO_ACCESS_KEY="fitpulse-admin"
MINIO_SECRET_KEY="REPLACE_WITH_REAL_MINIO_SECRET_KEY"

# Add to GitHub Secrets:
# MINIO_ACCESS_KEY → generated value
# MINIO_SECRET_KEY → generated value
```

### EXTERNAL_SECRETS_DB_PASSWORD

**Source:** You set this yourself when creating PostgreSQL user.

```bash
# Generate random password
openssl rand -base64 32

# Create PostgreSQL user (run once)
psql -h postgres-service -U postgres -f scripts/ci/setup-external-secrets-db.sh
# Replace REPLACE_WITH_REAL_PASSWORD in the script with generated value

# Add to GitHub Secrets:
# EXTERNAL_SECRETS_DB_PASSWORD → generated value
```

### KUBECONFIG_DATA

**Source:** Your existing k3s cluster.

```bash
# Copy kubeconfig from VPS
scp root@<VPS_IP>:/etc/rancher/k3s/k3s.yaml ~/.kube/config
# Update server IP
sed -i 's/127.0.0.1/<VPS_IP>/g' ~/.kube/config
# Base64 encode for GitHub Secrets
cat ~/.kube/config | base64 -w 0  # Linux/macOS
# Or on Windows PowerShell:
[Convert]::ToBase64String([IO.File]::ReadAllBytes("$HOME\.kube\config"))

# Add to GitHub Secrets:
# KUBECONFIG_DATA → base64-encoded kubeconfig content
```

### CLOUDFLARE_API_TOKEN (Optional, for DNS)

**Source:** Cloudflare API token (free, no card required for DNS-only access).

```bash
# 1. Create Cloudflare account (free, no card required for DNS)
# 2. Go to: https://dash.cloudflare.com/profile/api-tokens
# 3. Click "Create Token"
# 4. Use template "Edit zone DNS"
# 5. Select your domain
# 6. Create token
# 7. Add to GitHub Secrets:
#    CLOUDFLARE_API_TOKEN → token value
```

### DUCKDNS_TOKEN (Alternative to Cloudflare)

**Source:** DuckDNS account (free, no card required).

```bash
# 1. Go to https://www.duckdns.org/
# 2. Sign in with GitHub/Twitter/Reddit (no card)
# 3. Create subdomain (e.g. fittpulse)
# 4. Copy token from "install" page
# 5. Add to GitHub Secrets:
#    DUCKDNS_TOKEN → token value
```

---

## Terraform — New Architecture (No VPS Provisioning)

### What Changed

- ❌ Removed: Hetzner/DigitalOcean/Vultr VPS provisioning
- ❌ Removed: k3s bootstrap via remote-exec
- ✅ Added: Connect to existing k3s via kubeconfig
- ✅ Added: Deploy MinIO on existing k3s
- ✅ Added: Manage DNS records (Cloudflare/DuckDNS)
- ✅ Added: PostgreSQL External Secrets backend

### Directory Structure

```text
terraform/
├── backend.tf                 # MinIO S3 backend (configure after MinIO deployed)
├── main.tf                    # k8s provider + modules
├── variables.tf               # kubeconfig, MinIO, DNS vars
├── outputs.tf                 # MinIO endpoint, URLs
├── modules/
│   ├── minio/                 # Deploy MinIO on k3s
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   └── outputs.tf
│   └── dns/                   # DuckDNS / Cloudflare
│       ├── main.tf
│       ├── variables.tf
│       └── outputs.tf
└── environments/
    ├── staging/
    │   └── main.tf
    └── production/
        └── main.tf
```

---

## Secrets Migration from AWS to PostgreSQL

### Step 1: Apply migrations

```bash
kubectl apply -f db/migrations/V4__external_secrets_schema.sql
kubectl apply -f db/migrations/V5__external_secrets_seed.sql
```

### Step 2: Create PostgreSQL user

```bash
# Linux/macOS
psql -h postgres-service -U postgres -f scripts/ci/setup-external-secrets-db.sh

# Windows PowerShell
.\scripts\ci\setup-external-secrets-db.ps1 -PostgresHost postgres-service -PostgresUser postgres -ExternalSecretsPassword <generated_password>
```

### Step 3: Update External Secrets Operator

```bash
# Deploy new ClusterSecretStore (PostgreSQL backend)
kubectl apply -f configs/k8s/base/external-secrets/postgres-secretstore.yaml
kubectl apply -f configs/k8s/base/external-secrets/db-credentials-secret.yaml

# Verify sync
kubectl get externalsecret -n fitness-platform-production
kubectl get secret app-secrets -n fitness-platform-production
```

---

## GitHub Secrets — Full List (No Card Required)

| Secret Name | Value | Source | Card Required? |
| ------------- | ------- | -------- | ---------------- |
| `KUBECONFIG_DATA` | base64 kubeconfig | Existing k3s cluster | ❌ No |
| `MINIO_ACCESS_KEY` | MinIO root user | You generate | ❌ No |
| `MINIO_SECRET_KEY` | MinIO root password | You generate | ❌ No |
| `EXTERNAL_SECRETS_DB_PASSWORD` | PostgreSQL password | You generate | ❌ No |
| `CLOUDFLARE_API_TOKEN` | Cloudflare API token | cloudflare.com (free) | ❌ No |
| `DUCKDNS_TOKEN` | DuckDNS token | duckdns.org (free) | ❌ No |

---

## CI/CD Integration

```yaml
# .github/workflows/terraform.yml
name: Terraform
on:
  pull_request:
    paths: ['terraform/**']
  push:
    branches: [main]
    paths: ['terraform/**']

jobs:
  terraform:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: 1.9.0
      
      - name: Terraform Format
        run: terraform fmt -check -recursive
      
      - name: Terraform Init
        run: terraform init
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.MINIO_ACCESS_KEY }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.MINIO_SECRET_KEY }}
      
      - name: Terraform Plan
        run: terraform plan -var-file="environments/${{ github.ref == 'refs/heads/main' && 'production' || 'staging' }}.tfvars"
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.MINIO_ACCESS_KEY }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.MINIO_SECRET_KEY }}
      
      - name: Terraform Apply (main only)
        if: github.ref == 'refs/heads/main'
        run: terraform apply -auto-approve -var-file="environments/production.tfvars"
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.MINIO_ACCESS_KEY }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.MINIO_SECRET_KEY }}
          KUBECONFIG_DATA: ${{ secrets.KUBECONFIG_DATA }}
```

---

## Cost Breakdown (100% Free Stack)

| Component | Monthly Cost | Card Required? |
| ----------- | ------------- | ---------------- |
| Existing VPS (k3s) | €4 | Already paid |
| Terraform OSS | €0 | ❌ No |
| MinIO | €0 | ❌ No |
| PostgreSQL | €0 | ❌ No |
| DuckDNS | €0 | ❌ No |
| GitHub Actions | €0 | ❌ No |
| Cloudflare DNS | €0 | ❌ No |
| **Total additional** | **€0** | — |

---

## Implementation Priority

### Phase 1: Free Secrets Backend (1-2 days)

1. Apply V4 + V5 migrations
2. Create `external_secrets` PostgreSQL user
3. Deploy `postgres-secretstore.yaml`
4. Test secret sync
5. Decommission AWS SecretStore

### Phase 2: Terraform on Existing k3s (2-3 days)

1. Deploy MinIO on k3s via kubectl
2. Configure Terraform backend to use MinIO
3. Run `terraform init` + `terraform plan`
4. Test state locking

### Phase 3: DNS + Full IaC (1 week)

1. Add DNS module (Cloudflare or DuckDNS)
2. Migrate existing bash scripts to Terraform
3. Add CI/CD integration
