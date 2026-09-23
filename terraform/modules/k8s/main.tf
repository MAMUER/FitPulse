variable "vps_id" {
  description = "VPS instance ID for k3s master"
  type        = string
}

variable "vps_external_ip" {
  description = "VPS external IP"
  type        = string
}

variable "ssh_user" {
  description = "SSH user"
  type        = string
  default     = "ubuntu"
}

variable "ssh_private_key" {
  description = "SSH private key path"
  type        = string
}

variable "k3s_version" {
  description = "K3s version"
  type        = string
  default     = "v1.31.0+k3s1"
}

variable "k3s_token" {
  description = "K3s cluster token"
  type        = string
  sensitive   = true
}

resource "null_resource" "k3s_install" {
  triggers = {
    vps_id         = var.vps_id
    k3s_version    = var.k3s_version
    k3s_token_hash = sha1(var.k3s_token)
  }

  connection {
    type        = "ssh"
    user        = var.ssh_user
    private_key = file(var.ssh_private_key)
    host        = var.vps_external_ip
  }

  provisioner "remote-exec" {
    inline = [
      "curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=${var.k3s_version} K3S_TOKEN=${var.k3s_token} sh -s - --write-kubeconfig-mode 644 --tls-san ${var.vps_external_ip}",
      "mkdir -p ~/.kube",
      "sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config",
      "sudo chown $(id -u):$(id -g) ~/.kube/config",
      "echo 'export KUBECONFIG=~/.kube/config' >> ~/.bashrc",
      "kubectl wait --for=condition=Ready nodes --all --timeout=300s",
    ]
  }
}

output "kubeconfig" {
  value       = null_resource.k3s_install.Triggers.vps_id
  description = "K3s is installed on the VPS"
}
