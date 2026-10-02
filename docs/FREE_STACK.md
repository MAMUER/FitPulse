# FitPulse — 100% Free/Open-Source Stack Architecture

> **Objective:** Run the entire platform on zero-license-cost software.
> **Cost:** Only infrastructure (VPS ~€4/mo). No SaaS, no vendor lock-in.
> **Key constraint:** No credit card required for any component.

---

## Stack Components (All Free, No Card Required)

| Layer | Component | License | Cost | Card Required? |
| --- | --- | --- | --- | --- |
| **IaC** | Kubernetes manifests | — | Free | ❌ No |
| **VPS** | Existing VPS (k3s already running) | — | Already paid | ❌ No |
| **Kubernetes** | k3s | Apache-2.0 | Free | ❌ No |
| **Secrets** | Vault + External Secrets Operator | Apache-2.0 / BUSL-2.0 | Free | ❌ No |
| **DNS** | DuckDNS / Cloudflare | — | Free | ❌ No |
| **CI/CD** | GitHub Actions | — | Free tier | ❌ No |
| **Monitoring** | Prometheus + Grafana + Alertmanager | Apache-2.0 | Free | ❌ No |
| **Tracing** | Jaeger + OpenTelemetry | Apache-2.0 | Free | ❌ No |
| **Database** | PostgreSQL 18 + pgsodium | PostgreSQL License | Free | ❌ No |
| **Cache** | Valkey (Redis fork) | BSD-3-Clause | Free | ❌ No |
| **MQ** | RabbitMQ | MPL-2.0 | Free | ❌ No |
| **Container** | Docker / BuildKit | Apache-2.0 | Free | ❌ No |
| **TLS** | cert-manager + Let's Encrypt | Apache-2.0 | Free | ❌ No |
| **WAF** | Ingress NGINX + ModSecurity CRS v4 | Apache-2.0 | Free | ❌ No |
| **Backup** | MinIO (S3-compatible) | AGPL-3.0 | Free | ❌ No |

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
│  fittpulse.duckdns.org → <VPS_IP>                                       │
└────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Existing VPS (k3s already running)                             │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  k3s cluster (1 node, production)                         │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │  MinIO (S3-compatible)                              │  │  │
│  │  │  - Bucket: fitpulse-db-backups                      │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │  Vault                                              │  │  │
│  │  │  - SecretStore для External Secrets Operator        │  │  │
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
│  │  │ Device Aggr  │  │ Admin CLI    │  │ Valkey         │ │  │
│  │  │  (HTTP)      │  │  (CLI)       │  │  (Cache/Session)│ │  │
│  │  └──────────────┘  └──────────────┘  └────────────────┘ │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────────┐ │  │
│  │  │ RabbitMQ     │  │ Prometheus   │  │ Grafana        │ │  │
│  │  │  (Queue)     │  │  (Metrics)   │  │  (Dashboards)  │ │  │
│  │  └──────────────┘  └──────────────┘  └────────────────┘ │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────────┐ │  │
│  │  │ Alertmanager │  │ Jaeger       │  │ cert-manager   │ │  │
│  │  │  (Alerts)    │  │  (Traces)    │  │  (TLS)         │ │  │
│  │  └──────────────┘  └──────────────┘  └────────────────┘ │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Secrets Management — Free Stack (No AWS, No Card)

### Current State

- Vault deployed в `configs/k8s/base/vault/`
- External Secrets Operator deployed, использует PostgreSQL backend (таблица `external_secrets`)
- Секреты синхронизируются из Vault → External Secrets → Kubernetes Secret `app-secrets`

### Flow

```text
Vault (SecretStore)
         ↓
External Secrets Operator (PostgreSQL provider)
         ↓
Kubernetes Secret `app-secrets`
         ↓
Приложения (env vars from mounted secret)
```

---

## How to Get Each Secret (No Credit Card Required)

### Vault Root Token / Unseal Key

**Source:** Генерируется при первом запуске Vault.

```bash
# Vault уже развёрнут. Храните unseal key и root token в безопасном месте.
# Не храните в репозитории.
```

### MINIO_ACCESS_KEY / MINIO_SECRET_KEY

**Source:** You set these yourself when deploying MinIO.

```bash
# Option 1: Generate random values
openssl rand -base64 12   # ACCESS_KEY
openssl rand -base64 24   # SECRET_KEY

# Option 2: Use fixed values (change in production!)
MINIO_ACCESS_KEY="fitpulse-admin"
MINIO_SECRET_KEY="REPLACE_WITH_REAL_MINIO_SECRET_KEY"

# Add to Vault:
vault kv put secret/minio \
  access_key="$MINIO_ACCESS_KEY" \
  secret_key="$MINIO_SECRET_KEY"
```

### EXTERNAL_SECRETS_DB_PASSWORD

**Source:** You set this yourself when creating PostgreSQL user.

```bash
# Generate random password
openssl rand -base64 32

# Create PostgreSQL user (run once)
psql -h postgres-service -U postgres -f scripts/ci/setup-external-secrets-db.sh
# Replace REPLACE_WITH_REAL_PASSWORD in the script with generated value

# Add to Vault:
vault kv put secret/external-secrets \
  db_password="<generated-password>"
```

### KUBECONFIG_DATA

**Source:** Your existing k3s cluster.

```bash
# Copy kubeconfig from VPS
scp root@<VPS_IP>:/etc/rancher/k3s/k3s.yaml ~/.kube/config
# Update server IP
sed -i 's/127.0.0.1/<VPS_IP>/g' ~/.kube/config
# Base64 encode for GitHub Secrets
cat ~/.kube/config | base64 -w 0

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
# 7. Add to Vault:
vault kv put secret/cloudflare api_token="<token>"
```

---

## Kubernetes Manifests — Direct Deployment (No Terraform)

### What Changed

- ❌ Removed: Terraform-based VPS provisioning
- ❌ Removed: AWS Secrets Manager
- ✅ Added: Vault + External Secrets Operator
- ✅ Added: cert-manager для TLS
- ✅ Added: MinIO для бэкапов
- ✅ Added: WAF (Ingress NGINX + ModSecurity CRS v4)

---

## GitHub Secrets — Full List (No Card Required)

| Secret Name | Value | Source | Card Required? |
| --- | --- | --- | --- |
| `KUBECONFIG_DATA` | base64 kubeconfig | Existing k3s cluster | ❌ No |
| `MINIO_ACCESS_KEY` | MinIO root user | You generate | ❌ No |
| `MINIO_SECRET_KEY` | MinIO root password | You generate | ❌ No |
| `EXTERNAL_SECRETS_DB_PASSWORD` | PostgreSQL password | You generate | ❌ No |
| `CLOUDFLARE_API_TOKEN` | Cloudflare API token | cloudflare.com (free) | ❌ No |

---

## Cost Breakdown (100% Free Stack)

| Component | Monthly Cost | Card Required? |
| --- | --- | --- |
| Existing VPS (k3s) | €4 | Already paid |
| MinIO | €0 | ❌ No |
| PostgreSQL | €0 | ❌ No |
| DuckDNS | €0 | ❌ No |
| GitHub Actions | €0 | ❌ No |
| Cloudflare DNS | €0 | ❌ No |
| **Total additional** | **€0** | — |

---

## Implementation Notes

- Все секреты хранятся в Vault, синхронизируются через External Secrets Operator
- MinIO используется для бэкапов PostgreSQL (ежедневно через CronJob)
- cert-manager автоматически выписывает TLS-сертификаты через Let's Encrypt
- WAF (ModSecurity CRS v4) защищает от OWASP Top 10
- Valkey используется для сессий, rate limiting, TOTP cache
- RabbitMQ используется для очередей биометрических событий с DLQ
