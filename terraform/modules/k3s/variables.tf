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

variable "kubernetes_version" {
  description = "K3s version"
  type        = string
  default     = "v1.36.0+k3s1"
}

variable "node_count" {
  description = "Number of nodes"
  type        = number
  default     = 1
}

output "admin_token" {
  description = "K3s admin token"
  value       = "K3s token available after cluster bootstrap"
  sensitive   = true
}

output "kubeconfig_content" {
  description = "Kubeconfig content for accessing the cluster"
  value = yamlencode({
    apiVersion = "v1"
    kind       = "Config"
    clusters = [{
      name    = "k3s"
      cluster = {
        server                   = "https://${var.vps_host}:6443"
        "insecure-skip-tls-verify" = true
      }
    }]
    contexts = [{
      name    = "k3s-context"
      context = {
        cluster = "k3s"
        user    = "admin"
      }
    }]
    "current-context" = "k3s-context"
    users = [{
      name = "admin"
      user = {
        token = "K3S_ADMIN_TOKEN"
      }
    }]
  })
  sensitive = true
}

output "kubernetes_endpoint" {
  description = "Kubernetes API endpoint"
  value       = "https://${var.vps_host}:6443"
}
