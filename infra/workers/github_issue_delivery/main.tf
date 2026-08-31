# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

resource "google_service_account" "worker_sa" {
  account_id   = "gh-issue-deliv-${var.env_id}"
  provider     = google.internal_project
  display_name = "GitHub Issue Delivery Worker Service Account (${var.env_id})"
  project      = var.project_id
}

resource "google_cloud_run_v2_worker_pool" "worker" {
  for_each            = var.regions
  name                = "gh-issue-deliv-${var.env_id}-${each.key}"
  location            = each.key
  project             = var.project_id
  provider            = google.internal_project
  launch_stage        = "BETA"
  deletion_protection = var.deletion_protection

  scaling {
    manual_instance_count = var.manual_instance_count
  }
  template {
    service_account = google_service_account.worker_sa.email

    containers {
      image = var.image_url

      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
      env {
        name  = "SPANNER_INSTANCE"
        value = var.spanner_instance_id
      }
      env {
        name  = "SPANNER_DATABASE"
        value = var.spanner_database_id
      }
      env {
        name  = "GITHUB_ISSUE_DELIVERY_SUBSCRIPTION_ID"
        value = var.github_issue_delivery_subscription_id
      }
      env {
        name  = "GITHUB_APP_ID"
        value = var.github_app_id
      }
      env {
        name  = "GITHUB_APP_PRIVATE_KEY_PATH"
        value = var.github_app_private_key_secret_id != "" ? "/etc/secrets/github-app/private-key.pem" : ""
      }
      env {
        name  = "FRONTEND_BASE_URL"
        value = var.frontend_base_url
      }
      env {
        name  = "OTEL_SERVICE_NAME"
        value = "github-issue-delivery"
      }
      env {
        name  = "OTEL_GCP_PROJECT_ID"
        value = var.otel_project_id
      }
      env {
        name  = "OTEL_EXPORTER_OTLP_ENDPOINT"
        value = var.otel_collector_endpoint
      }
      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }
      dynamic "volume_mounts" {
        for_each = var.github_app_private_key_secret_id != "" ? [1] : []
        content {
          name       = "github-app-key"
          mount_path = "/etc/secrets/github-app"
        }
      }
    }
    containers {
      name  = "otel"
      image = var.otel_collector_image
      args  = ["--config=${var.otel_collector_config_mount_path}/config.yaml"]
      env {
        name  = "OTEL_COLLECTOR_REGION"
        value = each.key
      }
      volume_mounts {
        name       = "otel-config"
        mount_path = var.otel_collector_config_mount_path
      }
    }
    volumes {
      name = "otel-config"
      secret {
        secret = var.otel_config_secret_id
        items {
          version = "latest"
          path    = "config.yaml"
        }
      }
    }
    dynamic "volumes" {
      for_each = var.github_app_private_key_secret_id != "" ? [1] : []
      content {
        name = "github-app-key"
        secret {
          secret = var.github_app_private_key_secret_id
          items {
            version = "latest"
            path    = "private-key.pem"
          }
        }
      }
    }
  }

  depends_on = [
    google_spanner_database_iam_member.db_user,
    google_pubsub_subscription_iam_member.github_issue_delivery_sub,
    google_secret_manager_secret_iam_member.worker_github_app_key_secret_access,
  ]
}

resource "google_secret_manager_secret_iam_member" "worker_github_app_key_secret_access" {
  count     = var.github_app_private_key_secret_id != "" ? 1 : 0
  provider  = google.internal_project
  secret_id = var.github_app_private_key_secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.worker_sa.email}"
}

resource "google_spanner_database_iam_member" "db_user" {
  instance = var.spanner_instance_id
  database = var.spanner_database_id
  role     = "roles/spanner.databaseUser"
  member   = "serviceAccount:${google_service_account.worker_sa.email}"
  provider = google.internal_project
}

resource "google_pubsub_subscription_iam_member" "github_issue_delivery_sub" {
  subscription = var.github_issue_delivery_subscription_id
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${google_service_account.worker_sa.email}"
  provider     = google.internal_project
}

resource "google_project_iam_member" "gcp_metric_permission" {
  role     = "roles/monitoring.metricWriter"
  provider = google.internal_project
  project  = var.project_id
  member   = "serviceAccount:${google_service_account.worker_sa.email}"
}

resource "google_project_iam_member" "gcp_log_permission" {
  role     = "roles/logging.logWriter"
  provider = google.internal_project
  project  = var.project_id
  member   = "serviceAccount:${google_service_account.worker_sa.email}"
}

resource "google_project_iam_member" "gcp_trace_permission" {
  role     = "roles/cloudtrace.agent"
  provider = google.internal_project
  project  = var.project_id
  member   = "serviceAccount:${google_service_account.worker_sa.email}"
}
