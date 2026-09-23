variable "name" {
  description = "Vault instance name"
  type        = string
}

variable "vps_id" {
  description = "VPS instance ID"
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

variable "vault_version" {
  description = "Vault version"
  type        = string
  default     = "1.15.0"
}

variable "vault_root_token" {
  description = "Vault root token"
  type        = string
  sensitive   = true
}

variable "vault_tls_cert" {
  description = "Vault TLS certificate (PEM)"
  type        = string
  default     = ""
}

variable "vault_tls_key" {
  description = "Vault TLS private key (PEM)"
  type        = string
  sensitive   = true
  default     = ""
}

resource "null_resource" "vault_install" {
  triggers = {
    vps_id        = var.vps_id
    vault_version = var.vault_version
  }

  connection {
    type        = "ssh"
    user        = var.ssh_user
    private_key = file(var.ssh_private_key)
    host        = var.vps_external_ip
  }

  provisioner "remote-exec" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y gnupg software-properties-common curl",
      "curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -",
      "sudo apt-add-repository -y 'deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main'",
      "sudo apt-get update",
      "sudo apt-get install -y vault=${var.vault_version}*",
      "sudo systemctl enable vault",
    ]
  }
}

resource "null_resource" "vault_config" {
  triggers = {
    vps_id = var.vps_id
  }

  connection {
    type        = "ssh"
    user        = var.ssh_user
    private_key = file(var.ssh_private_key)
    host        = var.vps_external_ip
  }

  provisioner "remote-exec" {
    inline = [
      "sudo mkdir -p /etc/vault.d /var/lib/vault",
      "sudo chown vault:vault /var/lib/vault",
    ]
  }

  provisioner "file" {
    content = templatefile("${path.module}/config.hcl.tftpl", {
      vault_tls_cert = var.vault_tls_cert != "" ? var.vault_tls_cert : "disabled"
      vault_tls_key  = var.vault_tls_key != "" ? var.vault_tls_key : "disabled"
    })
    destination = "/tmp/vault.hcl"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo mv /tmp/vault.hcl /etc/vault.d/vault.hcl",
      "sudo chown vault:vault /etc/vault.d/vault.hcl",
      "sudo chmod 640 /etc/vault.d/vault.hcl",
    ]
  }
}

resource "null_resource" "vault_start" {
  triggers = {
    vps_id = var.vps_id
  }

  depends_on = [null_resource.vault_config]

  connection {
    type        = "ssh"
    user        = var.ssh_user
    private_key = file(var.ssh_private_key)
    host        = var.vps_external_ip
  }

  provisioner "remote-exec" {
    inline = [
      "sudo systemctl start vault",
      "sleep 5",
      "export VAULT_ADDR=https://127.0.0.1:8200",
      "vault status",
    ]
  }
}

resource "null_resource" "vault_init" {
  triggers = {
    vps_id = var.vps_id
  }

  depends_on = [null_resource.vault_start]

  connection {
    type        = "ssh"
    user        = var.ssh_user
    private_key = file(var.ssh_private_key)
    host        = var.vps_external_ip
  }

  provisioner "remote-exec" {
    inline = [
      "export VAULT_ADDR=https://127.0.0.1:8200",
      "vault login ${var.vault_root_token}",
      "vault secrets enable -version=2 kv",
      "vault secrets enable database",
      "vault auth enable kubernetes",
      "vault write auth/kubernetes/config kubernetes_host=https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_SERVICE_PORT",
    ]
  }
}

output "vault_addr" {
  value       = "https://${var.vps_external_ip}:8200"
  description = "Vault address"
}

output "vault_root_token" {
  value       = var.vault_root_token
  description = "Vault root token"
  sensitive   = true
}
