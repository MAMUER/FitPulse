variable "name" {
  description = "VPS instance name"
  type        = string
}

variable "region" {
  description = "Cloud region"
  type        = string
  default     = "ru-central1"
}

variable "cpu_cores" {
  description = "Number of CPU cores"
  type        = number
  default     = 4
}

variable "memory_gb" {
  description = "Memory in GB"
  type        = number
  default     = 8
}

variable "disk_gb" {
  description = "Disk size in GB"
  type        = number
  default     = 80
}

variable "disk_type" {
  description = "Disk type"
  type        = string
  default     = "ssd"
}

variable "ssh_keys" {
  description = "SSH public keys"
  type        = list(string)
  default     = []
}

variable "tags" {
  description = "Resource tags"
  type        = map(string)
  default     = {}
}

resource "yandex_compute_instance" "this" {
  name       = var.name
  platform   = "standard-v3"
  zone       = var.region
  hostname   = var.name
  tags       = concat(["fitpulse", "k8s-node"], keys(var.tags))

  resources {
    cores  = var.cpu_cores
    memory = var.memory_gb * 1024
  }

  boot_disk {
    initialize_params {
      image_id    = "fd87bj79l8q2ng6gba5g"  # Ubuntu 26.04 LTS
      size        = var.disk_gb
      type        = var.disk_type == "ssd" ? "network-ssd" : "network-hdd"
    }
  }

  network_interface {
    subnet_id = var.subnet_id
    nat       = true
  }

  metadata = {
    ssh-keys = join("\n", var.ssh_keys)
  }

  labels = var.tags
}

output "id" {
  value       = yandex_compute_instance.this.id
  description = "VPS instance ID"
}

output "external_ip" {
  value       = yandex_compute_instance.this.network_interface[0].nat_ip_address
  description = "External IP address"
}

output "internal_ip" {
  value       = yandex_compute_instance.this.network_interface[0].ip_address
  description = "Internal IP address"
}
