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

resource "google_project_service" "public_cloud_run" {
  provider = google.public_project
  service  = "run.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

resource "google_project_service" "public_certificate_manager" {
  provider = google.public_project
  service  = "certificatemanager.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}

# https://stackoverflow.com/questions/68154414/google-cloud-spanner-requests-enabling-on-a-different-project
# Public project does not deploy actual instances but it is needed.
resource "google_project_service" "public_spanner" {
  provider = google.public_project
  service  = "spanner.googleapis.com"

  disable_dependent_services = true
  disable_on_destroy         = false
}
