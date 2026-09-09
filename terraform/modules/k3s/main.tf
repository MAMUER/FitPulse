resource "null_resource" "k3s_install" {
  triggers = {
    vps_host    = var.vps_host
    k3s_version = var.kubernetes_version
  }

  connection {
    type        = "ssh"
    host        = var.vps_host
    user        = var.vps_user
    private_key = file(var.ssh_private_key)
    timeout     = "30m"
  }

  provisioner "remote-exec" {
    inline = [
      "curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION='${var.kubernetes_version}' K3S_URL=https://${var.vps_host}:6443 sh -s - server --write-kubeconfig /etc/rancher/k3s/k3s.yaml --write-kubeconfig-mode 600",
      "mkdir -p ~/.kube",
      "sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config",
      "sudo chown $(id -u):$(id -g) ~/.kube/config",
    ]
  }
}

output "admin_token" {
  description = "K3s admin token"
  value       = "K3S_TOKEN"
  sensitive   = true
}

output "kubeconfig_content" {
  description = "Kubeconfig content"
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
        token = "K3S_TOKEN"
      }
    }]
  })
  sensitive = true
}

output "kubernetes_endpoint" {
  description = "Kubernetes API endpoint"
  value       = "https://${var.vps_host}:6443"
}
