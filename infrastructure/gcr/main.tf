variable "name" {
  type = string
}

resource "google_cloud_run_v2_service" "default" {
  name     = "${var.name}-plugin-registry"
  location = "europe-west1"
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    max_instance_request_concurrency = 100

    scaling {
      max_instance_count = 1
    }

    containers {
      image = "gcr.io/go-semantic-release/plugin-registry"

      startup_probe {
        http_get {
          path = "/"
        }
      }

      liveness_probe {
        http_get {
          path = "/"
        }
      }

      resources {
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
        cpu_idle = true
      }
    }
  }

  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }
}
