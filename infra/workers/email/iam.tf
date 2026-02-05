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

data "google_service_account" "worker_sa" {
  account_id = var.service_account_email
  project    = var.project_id
  provider   = google.internal_project
}

# Grant permissions to the passed-in Service Account via the data source
resource "google_spanner_database_iam_member" "db_user" {
  instance = var.spanner_instance_id
  database = var.spanner_database_id
  role     = "roles/spanner.databaseUser"
  member   = "serviceAccount:${data.google_service_account.worker_sa.email}"
  provider = google.internal_project
}

resource "google_pubsub_subscription_iam_member" "email_sub" {
  subscription = var.email_subscription_id
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${data.google_service_account.worker_sa.email}"
  provider     = google.internal_project
}

resource "google_project_iam_member" "gcp_metric_permission" {
  role     = "roles/monitoring.metricWriter"
  provider = google.internal_project
  project  = var.project_id
  member   = "serviceAccount:${data.google_service_account.worker_sa.email}"
}

resource "google_project_iam_member" "gcp_log_permission" {
  role     = "roles/logging.logWriter"
  provider = google.internal_project
  project  = var.project_id
  member   = "serviceAccount:${data.google_service_account.worker_sa.email}"
}

resource "google_project_iam_member" "gcp_trace_permission" {
  role     = "roles/cloudtrace.agent"
  provider = google.internal_project
  project  = var.project_id
  member   = "serviceAccount:${data.google_service_account.worker_sa.email}"
}


resource "google_secret_manager_secret_iam_member" "worker_access_from_address" {
  provider   = google.internal_project
  secret_id  = data.google_secret_manager_secret.from_address_secret.id
  role       = "roles/secretmanager.secretAccessor"
  member     = "serviceAccount:${data.google_service_account.worker_sa.email}"
  depends_on = [data.google_secret_manager_secret.from_address_secret]
}

resource "google_secret_manager_secret_iam_member" "worker_access_bcc" {
  provider   = google.internal_project
  secret_id  = data.google_secret_manager_secret.bcc_secret.id
  role       = "roles/secretmanager.secretAccessor"
  member     = "serviceAccount:${data.google_service_account.worker_sa.email}"
  depends_on = [data.google_secret_manager_secret.bcc_secret]
}
