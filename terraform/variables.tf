variable "kubeconfig_path" {
  type        = string
  description = "Path to kubeconfig for existing k3s cluster"
}

variable "vps_ip" {
  type        = string
  description = "Public IPv4 address of the existing VPS"
}

variable "domain" {
  type        = string
  description = "Primary domain (e.g. fittpulse.duckdns.org)"
}

variable "duckdns_token" {
  type        = string
  description = "DuckDNS token"
  sensitive   = true
}

variable "minio_root_user" {
  type        = string
  description = "MinIO root username"
  sensitive   = true
}

variable "minio_root_password" {
  type        = string
  description = "MinIO root password"
  sensitive   = true
}

variable "storage_class" {
  type        = string
  default     = "local-path"
  description = "Kubernetes StorageClass for MinIO PVC"
}

variable "external_secrets_db_password" {
  type        = string
  description = "PostgreSQL password for External Secrets user"
  sensitive   = true
}
