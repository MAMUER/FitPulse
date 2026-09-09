module "vps" {
  source = "../modules/vps"

  vps_host        = var.vps_host
  vps_user        = var.vps_user
  ssh_private_key = var.ssh_private_key
  domain          = var.domain
  environment     = var.environment
  kubernetes_version = var.kubernetes_version

  enable_monitoring = var.enable_monitoring
  enable_backup     = var.enable_backup

  tags = {
    Project     = "FitPulse"
    ManagedBy   = "Terraform"
    Environment = var.environment
  }
}

module "k3s" {
  source = "../modules/k3s"

  vps_host          = var.vps_host
  vps_user          = var.vps_user
  ssh_private_key   = var.ssh_private_key
  kubernetes_version = var.kubernetes_version
  node_count        = var.node_count
}

module "kubernetes_apps" {
  source = "../modules/kubernetes-apps"

  kubeconfig_path = var.kubeconfig_path
  domain          = var.domain
  environment     = var.environment

  enable_monitoring = var.enable_monitoring
  enable_backup     = var.enable_backup
}
