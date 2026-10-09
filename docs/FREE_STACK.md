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

### MINIO_ROOT_PASSWORD

**Source:** You set this yourself when deploying MinIO.

```bash
# Add to GitHub Secrets:
# MINIO_SECRET_KEY → <generated-password>
```

### EXTERNAL_SECRETS_DB_PASSWORD

**Source:** You set this yourself when creating PostgreSQL user.

```bash
# Generate random password
openssl rand -base64 32

# Create PostgreSQL user (run once)
psql -h postgres-service -U postgres -c "CREATE USER external_secrets WITH PASSWORD '<generated-password>';"
psql -h postgres-service -U postgres -c "GRANT USAGE ON SCHEMA public TO external_secrets;"
psql -h postgres-service -U postgres -c "GRANT SELECT, INSERT, UPDATE, DELETE ON external_secrets TO external_secrets;"
psql -h postgres-service -U postgres -c "GRANT USAGE, SELECT ON SEQUENCE external_secrets_id_seq TO external_secrets;"

# Add to GitHub Secrets:
# EXTERNAL_SECRETS_DB_PASSWORD → <generated-password>
```

### ELASTICSEARCH_PASSWORD / ELASTICSEARCH_PLATFORM_PASSWORD / ELASTICSEARCH_AUDITOR_PASSWORD

**Source:** You set these yourself.

```bash
# Generate random passwords
openssl rand -base64 32  # ELASTICSEARCH_PASSWORD
openssl rand -base64 32  # ELASTICSEARCH_PLATFORM_PASSWORD
openssl rand -base64 32  # ELASTICSEARCH_AUDITOR_PASSWORD

# Add to GitHub Secrets:
# ELASTICSEARCH_PASSWORD → <elastic-user-password>
# ELASTICSEARCH_PLATFORM_PASSWORD → <platform-team-password>
# ELASTICSEARCH_AUDITOR_PASSWORD → <auditor-password>
```

### GRAFANA_ADMIN_PASSWORD

**Source:** You set this yourself.

```bash
# Generate random password
openssl rand -base64 24

# Add to GitHub Secrets:
# GRAFANA_ADMIN_PASSWORD → <generated-password>
```

### ARGOCD_ADMIN_PASSWORD (Optional)

**Source:** You set this yourself if deploying ArgoCD.

```bash
# Generate random password
openssl rand -base64 24

# Add to GitHub Secrets:
# ARGOCD_ADMIN_PASSWORD → <generated-password>
```

### TELEGRAM_BOT_TOKEN / TELEGRAM_CHAT_ID

```bash
# 1. Create bot via @BotFather (free, no card required)
# 2. Get chat ID via @chatid_echo_bot
# 3. Add to GitHub Secrets:
# TELEGRAM_BOT_TOKEN → <bot-token>
# TELEGRAM_CHAT_ID → <chat-id>
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

- ✅ Added: Vault + External Secrets Operator
- ✅ Added: cert-manager для TLS
- ✅ Added: MinIO для бэкапов
- ✅ Added: WAF (Ingress NGINX + ModSecurity CRS v4)

---

## GitHub Secrets — Full List

| Secret Name | Value | Source |
| --- | --- | --- | --- |
| `KUBECONFIG_DATA` | base64 kubeconfig | Existing k3s cluster |
| `MINIO_SECRET_KEY` | MinIO root password | You generate |
| `EXTERNAL_SECRETS_DB_PASSWORD` | PostgreSQL password | You generate |
| `ELASTICSEARCH_PASSWORD` | Elasticsearch elastic user | You generate |
| `ELASTICSEARCH_PLATFORM_PASSWORD` | Elasticsearch platform-team user | You generate |
| `ELASTICSEARCH_AUDITOR_PASSWORD` | Elasticsearch auditor user | You generate |
| `ARGOCD_ADMIN_PASSWORD` | ArgoCD admin password | You generate |
| `GRAFANA_ADMIN_PASSWORD` | Grafana admin password | You generate |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token | @BotFather |
| `TELEGRAM_CHAT_ID` | Telegram chat ID | @chatid_echo_bot |
| `CLOUDFLARE_API_TOKEN` | Cloudflare API token | cloudflare.com (free) |

---

## Implementation Notes

- Все секреты хранятся в Vault, синхронизируются через External Secrets Operator
- MinIO используется для бэкапов PostgreSQL (ежедневно через CronJob)
- cert-manager автоматически выписывает TLS-сертификаты через Let's Encrypt
- WAF (ModSecurity CRS v4) защищает от OWASP Top 10
- Valkey используется для сессий, rate limiting, TOTP cache
- RabbitMQ используется для очередей биометрических событий с DLQ
