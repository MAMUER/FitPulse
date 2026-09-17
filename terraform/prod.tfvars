variable "hcloud_token" {
  type        = string
  description = "Hetzner Cloud API token"
  sensitive   = true
}

variable "hcloud_ssh_key_id" {
  type        = string
  default     = ""
  description = "Hetzner SSH key ID (optional)"
}
