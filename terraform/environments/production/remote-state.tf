terraform {
  backend "local" {
    path = "terraform.tfstate"
  }

  # For production, use S3 backend with DynamoDB locking:
  # backend "s3" {
  #   bucket         = "fitpulse-terraform-state-prod"
  #   key            = "infra/terraform.tfstate"
  #   region         = "us-east-1"
  #   dynamodb_table = "fitpulse-terraform-locks"
  #   encrypt        = true
  #   kms_key_id     = "arn:aws:kms:us-east-1:123456789:alias/terraform-state"
  # }
}
