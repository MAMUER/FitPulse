variable "name" {
  description = "Database instance name"
  type        = string
}

variable "vps_id" {
  description = "VPS instance ID (for self-hosted)"
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

variable "postgres_version" {
  description = "PostgreSQL version"
  type        = string
  default     = "18"
}

variable "postgres_user" {
  description = "PostgreSQL admin user"
  type        = string
  sensitive   = true
}

variable "postgres_password" {
  description = "PostgreSQL admin password"
  type        = string
  sensitive   = true
}

variable "postgres_db" {
  description = "Default database name"
  type        = string
  default     = "fitpulse"
}

variable "backup_s3_bucket" {
  description = "S3 bucket for backups"
  type        = string
  default     = ""
}

variable "backup_schedule" {
  description = "Backup cron schedule"
  type        = string
  default     = "0 2 * * *"
}

resource "null_resource" "postgres_install" {
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
      "sudo apt-get update",
      "sudo apt-get install -y postgresql-${var.postgres_version} postgresql-${var.postgres_version}-pgsodium",
      "sudo -u postgres psql -c \"ALTER USER postgres PASSWORD '${var.postgres_password}';\"",
      "sudo -u postgres psql -c \"CREATE DATABASE ${var.postgres_db};\" || true",
    ]
  }
}

resource "null_resource" "postgres_config" {
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
      "sudo tee /etc/postgresql/${var.postgres_version}/main/conf.d/fitpulse.conf > /dev/null <<EOF",
      "shared_preload_libraries = 'pgsodium'",
      "wal_level = replica",
      "max_wal_senders = 10",
      "wal_keep_size = 1GB",
      "hot_standby = on",
      "EOF",
      "sudo systemctl restart postgresql",
    ]
  }
}

resource "null_resource" "postgres_backup" {
  count = var.backup_s3_bucket != "" ? 1 : 0

  triggers = {
    vps_id         = var.vps_id
    backup_schedule = var.backup_schedule
  }

  connection {
    type        = "ssh"
    user        = var.ssh_user
    private_key = file(var.ssh_private_key)
    host        = var.vps_external_ip
  }

  provisioner "remote-exec" {
    inline = [
      "sudo tee /etc/cron.d/postgres-backup > /dev/null <<EOF",
      "${var.backup_schedule} root /usr/local/bin/postgres-backup.sh",
      "EOF",
      "sudo chmod 644 /etc/cron.d/postgres-backup",
    ]
  }
}

output "connection_string" {
  value       = "postgresql://${var.postgres_user}:${var.postgres_password}@${var.vps_external_ip}:5432/${var.postgres_db}?sslmode=require"
  description = "PostgreSQL connection string"
  sensitive   = true
}
