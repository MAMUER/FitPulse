output "minio_endpoint" {
  description = "MinIO API endpoint"
  value       = module.minio.endpoint
}

output "minio_console_endpoint" {
  description = "MinIO console endpoint"
  value       = module.minio.console_endpoint
}

output "terraform_state_bucket" {
  description = "Bucket name for Terraform state"
  value       = module.minio.terraform_state_bucket
}

output "app_url" {
  description = "Main application URL"
  value       = module.dns.app_url
}
