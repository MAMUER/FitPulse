# Terraform Modules for FitPulse

## Overview

These modules provision the core infrastructure for FitPulse Phase 2.

## Modules

### vps

Provisions a VPS instance (Yandex Cloud / Selectel / Timeweb).

```hcl
module "vps_master" {
  source     = "./modules/vps"
  name       = "fitpulse-master"
  region     = "ru-central1"
  cpu_cores  = 4
  memory_gb  = 8
  disk_gb    = 80
  disk_type  = "ssd"
  ssh_keys   = ["ssh-rsa AAAA..."]
  tags       = { env = "production" }
}
```

### k8s

Installs k3s on a VPS instance.

```hcl
module "k3s_master" {
  source          = "./modules/k8s"
  vps_id          = module.vps_master.id
  vps_external_ip = module.vps_master.external_ip
  ssh_user        = "ubuntu"
  ssh_private_key = "~/.ssh/id_rsa"
  k3s_version     = "v1.31.0+k3s1"
  k3s_token       = var.k3s_token
}
```

### db

Provisions PostgreSQL on a VPS instance.

```hcl
module "postgres" {
  source             = "./modules/db"
  name               = "fitpulse-postgres"
  vps_id             = module.vps_db.id
  vps_external_ip    = module.vps_db.external_ip
  ssh_user           = "ubuntu"
  ssh_private_key    = "~/.ssh/id_rsa"
  postgres_version   = "18"
  postgres_user      = var.postgres_user
  postgres_password  = var.postgres_password
  postgres_db        = "fitpulse"
  backup_s3_bucket   = module.minio.bucket_backups
  backup_schedule    = "0 2 * * *"
}
```

### vault

Installs HashiCorp Vault on a VPS instance.

```hcl
module "vault" {
  source             = "./modules/vault"
  name               = "fitpulse-vault"
  vps_id             = module.vps_vault.id
  vps_external_ip    = module.vps_vault.external_ip
  ssh_user           = "ubuntu"
  ssh_private_key    = "~/.ssh/id_rsa"
  vault_version      = "1.15.0"
  vault_root_token   = var.vault_root_token
  vault_tls_cert     = tls_self_signed_cert.vault.cert_pem
  vault_tls_key      = tls_self_signed_cert.vault.private_key_pem
}
```

## Prerequisites

- Terraform >= 1.5.0
- Cloud provider CLI configured (yc, aws, etc.)
- SSH key pair for VPS access
- S3-compatible storage for Terraform state

## State

Terraform state is stored in S3-compatible backend (MinIO).

```hcl
terraform {
  backend "s3" {
    endpoint     = "https://minio.fittpulse.ru"
    bucket       = "fitpulse-terraform-state"
    key          = "terraform.tfstate"
    region       = "us-east-1"
    skip_credentials_validation = true
    skip_metadata_api_check     = true
    force_path_style            = true
  }
}
```

## Usage

```bash
cd terraform
terraform init
terraform plan
terraform apply
```

## Notes

- These modules are **not** production-ready. They require:
  - Separate VPS for each component (Vault, PostgreSQL, k3s)
  - TLS certificates for Vault
  - Proper network segmentation
  - Monitoring and alerting
- For production, consider using managed services (Yandex Managed PostgreSQL, etc.)
