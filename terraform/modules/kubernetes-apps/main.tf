resource "kubernetes_namespace" "monitoring" {
  count = var.enable_monitoring ? 1 : 0

  metadata {
    name = "monitoring"
    labels = {
      name = "monitoring"
    }
  }
}

resource "kubernetes_namespace" "cert_manager" {
  metadata {
    name = "cert-manager"

    labels = {
      name = "cert-manager"
    }
  }
}

resource "kubernetes_namespace" "ml_system" {
  metadata {
    name = "ml-system"

    labels = {
      name = "ml-system"
      zone  = "ml"
    }
  }
}

resource "kubernetes_namespace" "database" {
  metadata {
    name = "database"

    labels = {
      name = "database"
      zone  = "data"
    }
  }
}

resource "helm_release" "vpa" {
  name       = "vpa"
  repository = "https://kubernetes.github.io/autoscaler"
  chart      = "vertical-pod-autoscaler"
  version    = "2.4.2"
  namespace  = "kube-system"

  values = [
    yamlencode({
      installer = {
        crds = {
          keep = false
        }
      }
      verticalPodAutoscaler = {
        enabled = true
        updatePolicy = {
          updateMode = "Auto"
        }
      }
    })
  ]

  depends_on = [helm_release.ingress_nginx]
}

resource "helm_release" "argocd" {
  name       = "argocd"
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argo-cd"
  version    = "7.5.0"
  namespace  = "argocd"

  create_namespace = true

  values = [
    yamlencode({
      global = {
        domain = "${var.domain}"
      }
      configs = {
        params = {
          server = {
            rootpath = "/argocd"
          }
        }
      }
      server = {
        service = {
          type = "ClusterIP"
        }
        resources = {
          requests = {
            cpu    = "50m"
            memory = "128Mi"
          }
          limits = {
            cpu    = "500m"
            memory = "512Mi"
          }
        }
      }
      repoServer = {
        resources = {
          requests = {
            cpu    = "50m"
            memory = "128Mi"
          }
          limits = {
            cpu    = "500m"
            memory = "512Mi"
          }
        }
      }
      applicationSet = {
        enabled = true
      }
    })
  ]

  depends_on = [helm_release.ingress_nginx]
}

resource "helm_release" "external_secrets" {
  name       = "external-secrets"
  repository = "https://charts.external-secrets.io"
  chart      = "external-secrets"
  version    = "0.10.5"
  namespace  = "external-secrets"

  create_namespace = true

  values = [
    yamlencode({
      serviceAccount = {
        create = true
        name   = "external-secrets-sa"
      }
    })
  ]

  depends_on = [helm_release.ingress_nginx]
}

resource "helm_release" "cert_manager" {
  name       = "cert-manager"
  repository = "https://charts.jetstack.io"
  chart      = "cert-manager"
  version    = "v1.17.0"
  namespace  = kubernetes_namespace.cert_manager[0].metadata[0].name

  values = [
    yamlencode({
      installCRDs = true
      prometheus = {
        enabled = var.enable_monitoring
      }
    })
  ]

  depends_on = [kubernetes_namespace.cert_manager]
}

resource "helm_release" "prometheus" {
  count      = var.enable_monitoring ? 1 : 0
  name       = "prometheus"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "kube-prometheus-stack"
  version    = "68.0.0"
  namespace  = kubernetes_namespace.monitoring[0].metadata[0].name

  values = [
    yamlencode({
      prometheus = {
        prometheusSpec = {
          serviceMonitorSelectorNilUsesHelmValues = false
        }
      }
      grafana = {
        adminPassword = var.grafana_admin_password
      }
    })
  ]

  depends_on = [helm_release.cert_manager]
}

resource "helm_release" "ingress_nginx" {
  name       = "ingress-nginx"
  repository = "https://kubernetes.github.io/ingress-nginx"
  chart      = "ingress-nginx"
  version    = "4.12.0"
  namespace  = "ingress-nginx"

  create_namespace = true

  values = [
    yamlencode({
      controller = {
        service = {
          type      = "NodePort"
          nodePorts = {
            http  = 30080
            https = 30443
          }
        }
      }
    })
  ]
}

# Managed by External Secrets Operator (ESO) via configs/k8s/base/external-secrets/app-secrets.yaml
# resource "kubernetes_secret" "app_secrets" {
#   metadata {
#     name      = "app-secrets"
#     namespace = "fitness-platform-production"
#   }
#
#   data = {
#     JWT_PRIVATE_KEY_PEM        = var.jwt_private_key_pem
#     JWT_PUBLIC_KEY_PEM         = var.jwt_public_key_pem
#     RABBITMQ_URL               = var.rabbitmq_url
#     REDIS_PASSWORD             = var.redis_password
#     POSTGRES_PASSWORD          = var.postgres_password
#     GOOGLE_CLIENT_ID           = var.google_client_id
#     GOOGLE_CLIENT_SECRET       = var.google_client_secret
#     SMTP_PASSWORD              = var.smtp_password
#     TOTP_ENCRYPTION_KEY        = var.totp_encryption_key
#   }
#
#   type = "Opaque"
# }

variable "grafana_admin_password" {
  description = "Grafana admin password"
  type        = string
  sensitive   = true
}

variable "jwt_private_key_pem" {
  description = "JWT private key PEM"
  type        = string
  sensitive   = true
}

variable "jwt_public_key_pem" {
  description = "JWT public key PEM"
  type        = string
  sensitive   = true
}

variable "rabbitmq_url" {
  description = "RabbitMQ URL"
  type        = string
  sensitive   = true
}

variable "redis_password" {
  description = "Redis/Valkey password"
  type        = string
  sensitive   = true
}

variable "postgres_password" {
  description = "PostgreSQL password"
  type        = string
  sensitive   = true
}

variable "google_client_id" {
  description = "Google OAuth client ID"
  type        = string
  sensitive   = true
}

variable "google_client_secret" {
  description = "Google OAuth client secret"
  type        = string
  sensitive   = true
}

variable "smtp_password" {
  description = "SMTP password"
  type        = string
  sensitive   = true
}

variable "totp_encryption_key" {
  description = "TOTP encryption key"
  type        = string
  sensitive   = true
}
