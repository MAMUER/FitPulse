output "endpoint" {
  description = "MinIO API endpoint"
  value       = "http://${kubernetes_service.minio.metadata[0].name}.${kubernetes_namespace.minio.metadata[0].name}.svc.cluster.local:9000"
}

output "console_endpoint" {
  description = "MinIO console endpoint"
  value       = "http://${kubernetes_service.minio.metadata[0].name}.${kubernetes_namespace.minio.metadata[0].name}.svc.cluster.local:9001"
}

output "terraform_state_bucket" {
  description = "Bucket name for Terraform state"
  value       = var.bucket_terraform
}
