variable "domain" {
  type        = string
  description = "Primary domain (e.g. fittpulse.duckdns.org)"
}

variable "vps_ip" {
  type        = string
  description = "Public IPv4 address of the VPS"
}

variable "duckdns_token" {
  type        = string
  description = "DuckDNS token"
  sensitive   = true
}
