variable "namespace" {
  type        = string
  default     = "minio"
  description = "Kubernetes namespace for MinIO"
}

variable "domain" {
  type        = string
  description = "Primary domain for MinIO console redirect"
}

variable "root_user" {
  type        = string
  description = "MinIO root username"
  sensitive   = true
}

variable "root_password" {
  type        = string
  description = "MinIO root password"
  sensitive   = true
}

variable "storage_class" {
  type        = string
  default     = "local-path"
  description = "Kubernetes StorageClass for MinIO PVC"
}

variable "bucket_terraform" {
  type        = string
  description = "Bucket name for Terraform state"
}

variable "bucket_models" {
  type        = string
  default     = "fitpulse-ml-models"
  description = "Bucket name for ML models"
}

variable "bucket_backups" {
  type        = string
  default     = "fitpulse-db-backups"
  description = "Bucket name for database backups"
}
