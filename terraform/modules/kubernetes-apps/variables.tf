variable "kubeconfig_path" {
  description = "Path to kubeconfig"
  type        = string
}

variable "domain" {
  description = "Primary domain"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "enable_monitoring" {
  description = "Enable monitoring"
  type        = bool
  default     = true
}

variable "enable_backup" {
  description = "Enable backups"
  type        = bool
  default     = true
}

variable "grafana_admin_password" {
  description = "Grafana admin password"
  type        = string
  sensitive   = true
}

variable "jwt_private_key_pem" {
  description = "JWT private key"
  type        = string
  sensitive   = true
}

variable "jwt_public_key_pem" {
  description = "JWT public key"
  type        = string
  sensitive   = true
}

variable "rabbitmq_url" {
  description = "RabbitMQ URL"
  type        = string
  sensitive   = true
}

variable "redis_password" {
  description = "Redis password"
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
