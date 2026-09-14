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
  description = "Grafana admin password (store in Secrets Manager)"
  type        = string
  sensitive   = true
}

variable "jwt_private_key_pem" {
  description = "JWT private key PEM (store in Secrets Manager)"
  type        = string
  sensitive   = true
}

variable "jwt_public_key_pem" {
  description = "JWT public key PEM (store in Secrets Manager)"
  type        = string
  sensitive   = true
}

variable "rabbitmq_url" {
  description = "RabbitMQ URL (store in Secrets Manager)"
  type        = string
  sensitive   = true
}

variable "valkey_password" {
  description = "Valkey password (store in Secrets Manager)"
  type        = string
  sensitive   = true
}

variable "postgres_password" {
  description = "PostgreSQL password (store in Secrets Manager)"
  type        = string
  sensitive   = true
}

variable "google_client_id" {
  description = "Google OAuth client ID (store in Secrets Manager)"
  type        = string
  sensitive   = true
}

variable "google_client_secret" {
  description = "Google OAuth client secret (store in Secrets Manager)"
  type        = string
  sensitive   = true
}

variable "smtp_password" {
  description = "SMTP password (store in Secrets Manager)"
  type        = string
  sensitive   = true
}

variable "totp_encryption_key" {
  description = "TOTP encryption key (store in Secrets Manager)"
  type        = string
  sensitive   = true
}
