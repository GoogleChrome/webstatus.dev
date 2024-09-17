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

locals {
  service_dir = "workflows/steps/services/chromium_histogram_enums"
}
resource "docker_image" "chromium_histogram_enums_image" {
  name = "${var.docker_repository_details.url}/chromium_histogram_enums_image"
  build {
    context = "${path.cwd}/.."
    build_args = {
      service_dir : local.service_dir
      MAIN_BINARY : "job"
    }
    dockerfile = "images/go_service.Dockerfile"
  }
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
}
resource "docker_registry_image" "chromium_histogram_enums_remote_image" {
  name          = docker_image.chromium_histogram_enums_image.name
  keep_remotely = true
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
}

resource "google_service_account" "chromium_histogram_enums_service_account" {
  provider     = google.internal_project
  account_id   = "chromium-enums-job-${var.env_id}"
  display_name = "Chromium Histogram Enums Consumer service account for ${var.env_id}"
}

resource "google_project_iam_member" "gcp_spanner_user" {
  provider = google.internal_project
  role     = "roles/spanner.databaseUser"
  project  = var.spanner_datails.project_id
  member   = google_service_account.chromium_histogram_enums_service_account.member
}


resource "google_cloud_run_v2_job" "chromium_histogram_enums" {
  provider = google.internal_project
  count    = length(var.regions)
  name     = "${var.env_id}-${var.regions[count.index]}-chromium-enums-consumer"
  location = var.regions[count.index]

  template {
    template {
      timeout = format("%ds", var.timeout_seconds)
      containers {
        image = "${docker_image.chromium_histogram_enums_image.name}@${docker_registry_image.chromium_histogram_enums_remote_image.sha256_digest}"
        env {
          name  = "PROJECT_ID"
          value = var.spanner_datails.project_id
        }
        env {
          name  = "SPANNER_DATABASE"
          value = var.spanner_datails.database
        }
        env {
          name  = "SPANNER_INSTANCE"
          value = var.spanner_datails.instance
        }
      }
      service_account = google_service_account.chromium_histogram_enums_service_account.email
    }
  }

  deletion_protection = var.deletion_protection
}

resource "google_cloud_run_v2_job_iam_member" "chromium_histogram_enums_step_invoker" {
  count    = length(var.regions)
  provider = google.internal_project
  location = var.regions[count.index]
  name     = google_cloud_run_v2_job.chromium_histogram_enums[count.index].name
  role     = "roles/run.invoker"
  member   = google_service_account.service_account.member
}

resource "google_cloud_run_v2_job_iam_member" "chromium_histogram_enums_step_status" {
  count    = length(var.regions)
  provider = google.internal_project
  location = var.regions[count.index]
  name     = google_cloud_run_v2_job.chromium_histogram_enums[count.index].name
  role     = "roles/run.viewer"
  member   = google_service_account.service_account.member
}
