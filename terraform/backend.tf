# Вариант A: сначала локальный backend, чтобы Terraform мог запуститься
terraform {
  backend "local" {
    path = "terraform.tfstate"
  }
}
