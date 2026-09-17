# DNS module — manage DNS records via DuckDNS
#
# Free option:
# - DuckDNS: completely free dynamic DNS
#
# This module updates A record for:
# - fittpulse.duckdns.org (main app)

locals {
  subdomain = trimsuffix(trimsuffix(trimprefix(var.domain, "https://"), "/"), "*.")
}

# DuckDNS record (update via API call)
resource "null_resource" "duckdns_update" {
  triggers = {
    domain = local.subdomain
    ip     = var.vps_ip
  }

  provisioner "local-exec" {
    command = "curl -s 'https://www.duckdns.org/update?domains=${local.subdomain}&token=${var.duckdns_token}&ip=${var.vps_ip}'"
  }
}
