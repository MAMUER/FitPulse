# Root Terraform configuration — 100% free/open-source stack
#
# This Terraform configuration manages:
# - MinIO deployment on existing k3s
# - Optional DNS records via Cloudflare
# - Terraform state stored in MinIO S3-compatible backend
#
# Prerequisites:
# - Existing k3s cluster (already running on your VPS)
# - kubectl configured locally or in CI
# - MinIO deployed (can be done via kubectl before first terraform init)

terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.30"
    }
  }
}

provider "kubernetes" {
  config_path = var.kubeconfig_path
}

module "minio" {
  source           = "./modules/minio"
  namespace        = "minio"
  domain           = var.domain
  root_user        = var.minio_root_user
  root_password    = var.minio_root_password
  storage_class    = var.storage_class
  bucket_terraform = "fitpulse-terraform-state"
  bucket_models    = "fitpulse-ml-models"
  bucket_backups   = "fitpulse-db-backups"
}
