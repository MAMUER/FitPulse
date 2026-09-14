terraform {
  required_version = ">= 1.9.0"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.35"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.16"
    }
    cloudinit = {
      source  = "hashicorp/cloudinit"
      version = "~> 2.3"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5"
    }
  }
}

provider "kubernetes" {
  config_path = var.kubeconfig_path
}

provider "helm" {
}

module "vps" {
  source = "../../modules/vps"

  vps_host          = var.vps_host
  vps_user          = var.vps_user
  ssh_private_key   = var.ssh_private_key
  domain            = var.domain
  environment       = var.environment
  kubernetes_version = var.kubernetes_version

  enable_monitoring = var.enable_monitoring
  enable_backup     = var.enable_backup

  tags = {
    Project     = "FitPulse"
    ManagedBy   = "Terraform"
    Environment = var.environment
  }
}

module "k3s" {
  source = "../../modules/k3s"

  vps_host          = var.vps_host
  vps_user          = var.vps_user
  ssh_private_key   = var.ssh_private_key
  kubernetes_version = var.kubernetes_version
  node_count        = var.node_count
}

module "kubernetes_apps" {
  source = "../../modules/kubernetes-apps"

  kubeconfig_path      = var.kubeconfig_path
  domain               = var.domain
  environment          = var.environment

  enable_monitoring    = var.enable_monitoring
  enable_backup        = var.enable_backup

  grafana_admin_password = var.grafana_admin_password
  jwt_private_key_pem    = var.jwt_private_key_pem
  jwt_public_key_pem     = var.jwt_public_key_pem
  rabbitmq_url           = var.rabbitmq_url
  valkey_password         = var.valkey_password
  postgres_password       = var.postgres_password
  google_client_id        = var.google_client_id
  google_client_secret    = var.google_client_secret
  smtp_password           = var.smtp_password
  totp_encryption_key     = var.totp_encryption_key
}
