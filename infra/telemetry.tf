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

# --- Shared OpenTelemetry Configuration & Constants ---

locals {
  # The official, public Google-built OpenTelemetry Collector image.
  # Hardcoding this here allows upgrading the collector version across all services,
  # jobs, and workers in a single place.
  otel_collector_image = "us-docker.pkg.dev/cloud-ops-agents-artifacts/google-cloud-opentelemetry-collector/otelcol-google:0.151.0"
}

# --- Shared OpenTelemetry Configuration Secret ---

# The secret is created in the internal project where all jobs, workers,
# and the backend API run, avoiding the need to enable Secret Manager API in the public project.
resource "google_secret_manager_secret" "otel_config" {
  provider  = google.internal_project
  secret_id = "otel-collector-config-${var.env_id}"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "otel_config_version" {
  provider    = google.internal_project
  secret      = google_secret_manager_secret.otel_config.id
  secret_data = file("${path.root}/../otel/otel-collector-config.yaml")
}

# --- Centralized IAM Access Control for Telemetry Secret ---

# Grant read access to the OTel config secret for all background worker service accounts.
resource "google_secret_manager_secret_iam_member" "worker_otel_config_secret_access" {
  for_each = toset([
    "serviceAccount:event-producer-${var.env_id}@${var.projects.internal}.iam.gserviceaccount.com",
    "serviceAccount:push-delivery-${var.env_id}@${var.projects.internal}.iam.gserviceaccount.com",
    "serviceAccount:webhook-worker-${var.env_id}@${var.projects.internal}.iam.gserviceaccount.com",
    "serviceAccount:${var.email_service_account_email}", # Email worker uses pre-existing SA
  ])
  provider  = google.internal_project
  secret_id = google_secret_manager_secret.otel_config.id
  role      = "roles/secretmanager.secretAccessor"
  member    = each.value
}

# Grant read access to the OTel config secret for all daily ingestion job service accounts.
resource "google_secret_manager_secret_iam_member" "job_otel_config_secret_access" {
  for_each = toset([
    "web-features",
    "wpt-consumer",
    "bcd-consumer",
    "chromium-enum",
    "uma-consumer",
    "dev-signals",
    "feature-map",
  ])
  provider  = google.internal_project
  secret_id = google_secret_manager_secret.otel_config.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${each.value}-job-${var.env_id}@${var.projects.internal}.iam.gserviceaccount.com"
}
