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
  source = "../modules/vps"

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
  source = "../modules/k3s"

  vps_host          = var.vps_host
  vps_user          = var.vps_user
  ssh_private_key   = var.ssh_private_key
  kubernetes_version = var.kubernetes_version
  node_count        = var.node_count
}

module "kubernetes_apps" {
  source = "../modules/kubernetes-apps"

  kubeconfig_path = var.kubeconfig_path
  domain          = var.domain
  environment     = var.environment

  enable_monitoring = var.enable_monitoring
  enable_backup     = var.enable_backup
}
