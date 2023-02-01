variable "stage" {
  type = string
}

variable "envSecrets" {
  type = map(string)
}

resource "google_cloud_run_v2_service" "default" {
  name     = "${var.stage}-plugin-registry"
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

      env {
        name  = "STAGE"
        value = var.stage
      }

      dynamic "env" {
        for_each = var.envSecrets
        content {
          name = env.key
          value_source {
            secret_key_ref {
              secret  = env.value
              version = "latest"
            }
          }
        }
      }
    }
  }

  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }
}

// allow public access
resource "google_cloud_run_service_iam_binding" "default" {
  location = google_cloud_run_v2_service.default.location
  service  = google_cloud_run_v2_service.default.name
  role     = "roles/run.invoker"
  members = [
    "allUsers"
  ]
}
