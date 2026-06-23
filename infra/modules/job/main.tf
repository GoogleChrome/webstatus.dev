# Copyright 2024 Google LLC
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

resource "google_cloud_run_v2_job" "job" {
  provider = google.internal_project
  count    = length(var.regions)
  name     = "${var.env_id}-${var.regions[count.index]}-${var.short_name}"
  location = var.regions[count.index]

  template {
    template {
      timeout = format("%ds", var.timeout_seconds)
      containers {
        image = var.image
        resources {
          limits = {
            cpu    = var.resource_limits.cpu
            memory = var.resource_limits.memory
          }
        }
        dynamic "env" {
          for_each = var.env_vars
          content {
            name  = env.value.name
            value = env.value.value
          }
        }
        env {
          name  = "OTEL_SERVICE_NAME"
          value = var.short_name
        }
        env {
          name  = "OTEL_GCP_PROJECT_ID"
          value = var.otel_project_id
        }
      }
      containers {
        name  = "otel"
        image = var.otel_collector_image
        volume_mounts {
          name       = "otel-config"
          mount_path = var.otel_collector_config_mount_path
        }
        # No probes for Cloud Run Jobs as they run to completion
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
      service_account = google_service_account.job_service_account.email
    }
  }

  deletion_protection = var.deletion_protection
}


resource "google_service_account" "job_service_account" {
  provider     = google.internal_project
  account_id   = "${var.short_name}-job-${var.env_id}"
  display_name = "${var.full_name} Job service account for ${var.env_id}"
}

resource "google_project_iam_member" "spanner_user" {
  count    = var.does_process_write_to_spanner ? 1 : 0
  provider = google.internal_project
  role     = "roles/spanner.databaseUser"
  project  = var.spanner_project_id
  member   = google_service_account.job_service_account.member
}

resource "google_project_iam_member" "datastore_user" {
  count    = var.does_process_write_to_datastore ? 1 : 0
  provider = google.internal_project
  role     = "roles/datastore.user"
  # For now assume the spanner project also contains the datastore project.
  project = var.spanner_project_id
  member  = google_service_account.job_service_account.member
}

# --- Telemetry IAM Permissions for Ingestion Job Service Accounts ---

# Grant Cloud Trace Agent role to allow exporting traces.
resource "google_project_iam_member" "job_trace_agent" {
  provider = google.internal_project
  project  = var.spanner_project_id
  role     = "roles/cloudtrace.agent"
  member   = google_service_account.job_service_account.member
}

# Grant Monitoring Metric Writer role to allow exporting metrics.
resource "google_project_iam_member" "job_metric_writer" {
  provider = google.internal_project
  project  = var.spanner_project_id
  role     = "roles/monitoring.metricWriter"
  member   = google_service_account.job_service_account.member
}

# Grant Logging Log Writer role to allow exporting structured logs.
resource "google_project_iam_member" "job_log_writer" {
  provider = google.internal_project
  project  = var.spanner_project_id
  role     = "roles/logging.logWriter"
  member   = google_service_account.job_service_account.member
}
