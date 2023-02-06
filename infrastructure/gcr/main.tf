variable "stage" {
  type = string
}

variable "envSecrets" {
  type = map(string)
}

variable "domain" {
  type = string
}

resource "google_cloud_run_v2_service" "default" {
  name     = "${var.stage}-plugin-registry"
  location = "europe-west1"
  ingress  = "INGRESS_TRAFFIC_ALL"

#  annotations = {
#    "run.googleapis.com/launch-stage" = "BETA"
#  }

  template {
    max_instance_request_concurrency = 100
    timeout = "300s"

    scaling {
      max_instance_count = 3
    }

#    annotations = {
#      "run.googleapis.com/startup-cpu-boost" = "true"
#    }

    containers {
      image = "gcr.io/go-semantic-release/plugin-registry:1.20.0"

      startup_probe {
        period_seconds = 1
        http_get {
          path = "/ping"
        }
      }

      liveness_probe {
        http_get {
          path = "/ping"
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


data "google_project" "project" {}

resource "google_cloud_run_domain_mapping" "default" {
  name     = var.domain
  location = google_cloud_run_v2_service.default.location
  metadata {
    namespace = data.google_project.project.project_id
  }
  spec {
    route_name = google_cloud_run_v2_service.default.name
  }
}
