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
  account_id   = "webhook-worker-${var.env_id}"
  provider     = google.internal_project
  display_name = "Webhook Worker Service Account (${var.env_id})"
  project      = var.project_id
}

resource "google_cloud_run_v2_worker_pool" "worker" {
  for_each            = var.regions
  name                = "webhook-worker-${var.env_id}-${each.key}"
  provider            = google.internal_project
  location            = each.key
  project             = var.project_id
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
        name  = "WEBHOOK_SUBSCRIPTION_ID"
        value = var.webhook_subscription_id
      }
      env {
        name  = "FRONTEND_BASE_URL"
        value = var.frontend_base_url
      }
    }
  }
}

# Grant permissions to the passed-in Service Account via the data source
resource "google_spanner_database_iam_member" "db_user" {
  instance = var.spanner_instance_id
  database = var.spanner_database_id
  role     = "roles/spanner.databaseUser"
  member   = "serviceAccount:${google_service_account.worker_sa.email}"
  provider = google.internal_project
}

resource "google_pubsub_subscription_iam_member" "webhook_sub" {
  subscription = var.webhook_subscription_id
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
