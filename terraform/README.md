# Terraform — Proof of Concept (Roadmap Phase 2)

> **Status:** Concept only. No provider credentials or real resources.
> **Purpose:** Manage Kubernetes resources on your existing k3s cluster.
> **Constraint:** 100% free/open-source stack, no paid providers required.

## Architecture

```text
terraform/
├── backend.tf                 # MinIO S3-compatible backend (after MinIO is deployed)
├── main.tf                    # Root config: k8s provider + optional Cloudflare DNS
├── variables.tf               # Global variables
├── outputs.tf                 # Global outputs
├── .terraform.lock.hcl        # Provider lock file
├── .gitignore                 # Ignore .terraform, .kubeconfig, *.tfstate
└── modules/
    ├── minio/                 # MinIO deployment on existing k3s
    │   ├── main.tf
    │   ├── variables.tf
    │   └── outputs.tf
    └── dns/                   # DuckDNS / Cloudflare DNS management
        ├── main.tf
        ├── variables.tf
        └── outputs.tf
```

## Free/Open-Source Toolchain

| Component | Tool | License | Cost |
| ----------- | ------ | --------- | ------ |
| IaC engine | Terraform OSS | BSL-1.1 | Free |
| Kubernetes provider | hashicorp/kubernetes | MPL-2.0 | Free |
| State backend | MinIO (S3-compatible) | AGPL-3.0 | Free (on same VPS) |
| DNS | DuckDNS / Cloudflare | — | Free |
| Secrets backend | PostgreSQL + External Secrets Operator | Apache-2.0 | Free |

## Prerequisites

1. **Existing k3s cluster** running on your VPS
2. **kubectl configured** locally or in CI with access to the cluster
3. **MinIO deployed** on k3s (can be done via `kubectl apply` before first `terraform init`)
4. **GitHub Secrets** for CI/CD:
   - `KUBECONFIG_DATA` — base64-encoded kubeconfig content
   - `MINIO_ACCESS_KEY` — MinIO root user
   - `MINIO_SECRET_KEY` — MinIO root password
   - `CLOUDFLARE_API_TOKEN` — optional, for DNS management

## How to Use

### 1. Get kubeconfig from your existing VPS

```bash
# Copy kubeconfig from VPS to local machine
scp root@<VPS_IP>:/etc/rancher/k3s/k3s.yaml ~/.kube/config
# Update server IP in kubeconfig
sed -i 's/127.0.0.1/<VPS_IP>/g' ~/.kube/config
# Test connection
kubectl get nodes
```

### 2. Configure MinIO endpoint for Terraform backend

```hcl
# terraform/backend.tf
terraform {
  backend "s3" {
    endpoint                    = "http://minio.minio.svc.cluster.local:9000"
    bucket                      = "fitpulse-terraform-state"
    key                         = "k3s/production/terraform.tfstate"
    region                      = "us-east-1"
    skip_credentials_validation = true
    skip_metadata_api_check     = true
    use_lockfile                = true
  }
}
```

### 3. Initialize Terraform

```bash
cd terraform
terraform init
```

### 4. Plan and Apply

```bash
# Plan
terraform plan -var-file="production.tfvars"

# Apply
terraform apply -var-file="production.tfvars"
```

## Module: MinIO

Deploys MinIO on existing k3s for:

- Terraform state backend (S3-compatible)
- ML model storage
- Database backups

## Module: DNS

Manages DNS records via:

- Cloudflare API (preferred, if token provided)
- DuckDNS (free dynamic DNS alternative)

## Cost Estimate

| Resource | Cost | Notes |
| ---------- | ------ | ------- |
| Terraform OSS | €0 | Open-source |
| MinIO | €0 | Runs on existing VPS |
| DuckDNS | €0 | Free dynamic DNS |
| Cloudflare | €0 | Free tier |
| **Total** | **€0** | No new infrastructure needed |

## Phase 2 Roadmap

1. Deploy MinIO on existing k3s cluster
2. Configure Terraform backend to use MinIO
3. Add PostgreSQL External Secrets backend module
4. Add DNS management module
5. Migrate existing bash scripts to Terraform modules
6. Add CI/CD integration with GitHub Actions
