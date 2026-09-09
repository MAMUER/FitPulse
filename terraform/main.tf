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
  kubernetes {
    config_path = var.kubeconfig_path
  }
}

variable "kubeconfig_path" {
  description = "Path to kubeconfig file"
  type        = string
  default     = "~/.kube/config"
}

variable "vps_host" {
  description = "VPS IP address or hostname"
  type        = string
}

variable "vps_user" {
  description = "SSH user for VPS"
  type        = string
  default     = "root"
}

variable "ssh_private_key" {
  description = "SSH private key path"
  type        = string
  sensitive   = true
}

variable "domain" {
  description = "Primary domain for the application"
  type        = string
  default     = "fittpulse.duckdns.org"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "node_count" {
  description = "Number of worker nodes"
  type        = number
  default     = 1
}

variable "kubernetes_version" {
  description = "Kubernetes version for k3s"
  type        = string
  default     = "v1.36.0+k3s1"
}

variable "enable_monitoring" {
  description = "Enable monitoring stack (Prometheus, Grafana)"
  type        = bool
  default     = true
}

variable "enable_backup" {
  description = "Enable automated PostgreSQL backups"
  type        = bool
  default     = true
}

variable "grafana_admin_password" {
  description = "Grafana admin password"
  type        = string
  sensitive   = true
}

variable "jwt_private_key_pem" {
  description = "JWT private key PEM"
  type        = string
  sensitive   = true
}

variable "jwt_public_key_pem" {
  description = "JWT public key PEM"
  type        = string
  sensitive   = true
}

variable "rabbitmq_url" {
  description = "RabbitMQ URL"
  type        = string
  sensitive   = true
}

variable "redis_password" {
  description = "Redis/Valkey password"
  type        = string
  sensitive   = true
}

variable "postgres_password" {
  description = "PostgreSQL password"
  type        = string
  sensitive   = true
}

variable "google_client_id" {
  description = "Google OAuth client ID"
  type        = string
  sensitive   = true
}

variable "google_client_secret" {
  description = "Google OAuth client secret"
  type        = string
  sensitive   = true
}

variable "smtp_password" {
  description = "SMTP password"
  type        = string
  sensitive   = true
}

variable "totp_encryption_key" {
  description = "TOTP encryption key"
  type        = string
  sensitive   = true
}
