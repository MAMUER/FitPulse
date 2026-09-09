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
  description = "Primary domain"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "kubernetes_version" {
  description = "Kubernetes version for k3s"
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

variable "tags" {
  description = "Resource tags"
  type        = map(string)
  default     = {}
}

output "vps_ip" {
  description = "VPS IP address"
  value       = var.vps_host
}
