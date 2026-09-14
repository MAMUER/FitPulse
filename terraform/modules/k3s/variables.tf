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
