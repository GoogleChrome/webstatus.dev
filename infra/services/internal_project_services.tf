# Copyright 2023 Google LLC
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

resource "google_project_service" "internal_sercret" {
  provider = google.internal_project
  service  = "secretmanager.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "internal_artifact_registry" {
  provider = google.internal_project
  service  = "artifactregistry.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "internal_firestore" {
  provider = google.internal_project
  service  = "firestore.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "internal_spanner" {
  provider = google.internal_project
  service  = "spanner.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "internal_cloud_run" {
  provider = google.internal_project
  service  = "run.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "internal_workflows" {
  provider = google.internal_project
  service  = "workflows.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "internal_scheduler" {
  provider = google.internal_project
  service  = "cloudscheduler.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "internal_memorystore" {
  provider = google.internal_project
  service  = "memorystore.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "internal_private_service_access" {
  provider = google.internal_project
  service  = "servicenetworking.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "internal_logging" {
  provider = google.internal_project
  service  = "logging.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "internal_monitoring" {
  provider = google.internal_project
  service  = "monitoring.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "internal_trace" {
  provider = google.internal_project
  service  = "cloudtrace.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "host_private_service_access" {
  provider = google
  service  = "servicenetworking.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "host_networkconnectivity" {
  provider = google
  service  = "networkconnectivity.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}
