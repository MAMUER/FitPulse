data "tls_public_key" "vps_ssh" {
  private_key_pem = file(var.ssh_private_key)
}

resource "cloudinit_config" "vps_init" {
  gzip          = false
  base64_encode = false

  part {
    content_type = "text/cloud-config"
    content = yamlencode({
      packages = {
        update = true
        upgrade = true
      }
      package_update = true
      package_upgrade = true

      runcmd = [
        "apt-get install -y curl wget git jq sudo ufw htop iotop sysstat",
        "ufw allow 22/tcp",
        "ufw allow 80/tcp",
        "ufw allow 443/tcp",
        "ufw allow 6443/tcp",
        "ufw enable",
        "systemctl enable systemd-timesyncd",
        "timedatectl set-timezone UTC",
        "mkdir -p /etc/docker",
        "cat > /etc/docker/daemon.json <<'EOF'\n{\n  \\\"log-driver\\\": \\\"json-file\\\",\n  \\\"log-opts\\\": {\n    \\\"max-size\\\": \\\"10m\\\",\n    \\\"max-file\\\": \\\"3\\\"\n  }\n}\nEOF",
        "mkdir -p /etc/systemd/journald.conf.d",
        "cat > /etc/systemd/journald.conf.d/99-size-limit.conf <<'EOF'\n[Journal]\nSystemMaxUse=500M\nSystemMaxFileSize=50M\nMaxRetentionSec=1week\nEOF",
        "systemctl daemon-reload",
        "systemctl restart systemd-journald",
        "curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION='${var.kubernetes_version}' sh -s - server --write-kubeconfig /etc/rancher/k3s/k3s.yaml --write-kubeconfig-mode 600",
        "mkdir -p /etc/rancher/k3s",
        "curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION='${var.kubernetes_version}' K3S_URL=https://${var.vps_host}:6443 sh -s - server --write-kubeconfig /etc/rancher/k3s/k3s.yaml --write-kubeconfig-mode 600",
      ]
    })
  }
}

output "ssh_public_key" {
  description = "SSH public key for VPS access"
  value       = data.tls_public_key.vps_ssh.public_key_openssh
  sensitive   = true
}

output "vps_ip" {
  description = "VPS IP address"
  value       = var.vps_host
}
