terraform {
  backend "s3" {
    bucket         = "fitpulse-terraform-state-prod"
    key            = "infra/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "fitpulse-terraform-locks"
    encrypt        = true
  }
}
