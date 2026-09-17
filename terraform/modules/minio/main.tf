# MinIO module — deploy MinIO on k3s for S3-compatible state backend
#
# This module creates:
# - MinIO deployment + service in namespace `minio`
# - ServiceMonitor for Prometheus (optional)
# - Secret for credentials (managed by CI/CD or Terraform)

resource "kubernetes_namespace" "minio" {
  metadata {
    name = var.namespace
    labels = {
      app = "minio"
    }
  }
}

resource "kubernetes_secret" "minio_credentials" {
  metadata {
    name      = "minio-credentials"
    namespace = kubernetes_namespace.minio.metadata[0].name
  }

  data = {
    root-user     = var.root_user
    root-password = var.root_password
  }
}

resource "kubernetes_deployment" "minio" {
  metadata {
    name      = "minio"
    namespace = kubernetes_namespace.minio.metadata[0].name
    labels = {
      app = "minio"
    }
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        app = "minio"
      }
    }

    template {
      metadata {
        labels = {
          app = "minio"
        }
      }

      spec {
        security_context {
          run_as_non_root = true
          run_as_user     = 1001
          fs_group        = 1001
        }

        container {
          name  = "minio"
          image = "minio/minio:RELEASE.2025-10-15T17-29-55Z"

          port {
            container_port = 9000
            name           = "api"
          }
          port {
            container_port = 9001
            name           = "console"
          }

          env {
            name  = "MINIO_ROOT_USER"
            value_from {
              secret_key_ref {
                name = kubernetes_secret.minio_credentials.metadata[0].name
                key  = "root-user"
              }
            }
          }
          env {
            name  = "MINIO_ROOT_PASSWORD"
            value_from {
              secret_key_ref {
                name = kubernetes_secret.minio_credentials.metadata[0].name
                key  = "root-password"
              }
            }
          }
          env {
            name  = "MINIO_BROWSER_REDIRECT_URL"
            value = "https://minio.${var.domain}"
          }

          args = ["server", "/data", "--console-address", ":9001"]

          volume_mount {
            name       = "minio-storage"
            mount_path = "/data"
          }

          security_context {
            run_as_user                = 1001
            run_as_group               = 1001
            read_only_root_filesystem  = true
            allow_privilege_escalation = false
            capabilities {
              drop = ["ALL"]
            }
          }

          liveness_probe {
            http_get {
              path = "/minio/health/live"
              port = "9000"
            }
            initial_delay_seconds = 30
            period_seconds        = 30
          }

          readiness_probe {
            http_get {
              path = "/minio/health/ready"
              port = "9000"
            }
            initial_delay_seconds = 10
            period_seconds        = 10
          }
        }

        volume {
          name = "minio-storage"
          persistent_volume_claim {
            claim_name = kubernetes_persistent_volume_claim.minio.metadata[0].name
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "minio" {
  metadata {
    name      = "minio"
    namespace = kubernetes_namespace.minio.metadata[0].name
    labels = {
      app = "minio"
    }
    annotations = {
      "prometheus.io/scrape" = "true"
      "prometheus.io/port"   = "9000"
      "prometheus.io/path"   = "/minio/v2/metrics/cluster"
    }
  }

  spec {
    selector = {
      app = "minio"
    }

    port {
      name        = "api"
      port        = 9000
      target_port = 9000
    }

    port {
      name        = "console"
      port        = 9001
      target_port = 9001
    }

    type = "ClusterIP"
  }
}

resource "kubernetes_persistent_volume_claim" "minio" {
  metadata {
    name      = "minio-storage"
    namespace = kubernetes_namespace.minio.metadata[0].name
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "10Gi"
      }
    }

    storage_class_name = var.storage_class
  }
}
