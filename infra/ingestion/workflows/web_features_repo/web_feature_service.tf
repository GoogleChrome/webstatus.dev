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
  service_dir = "workflows/steps/services/web_feature_consumer"
}
resource "docker_image" "web_feature_consumer_image" {
  name = "${var.docker_repository_details.url}/web_feature_consumer_image"
  build {
    context = "${path.cwd}/.."
    build_args = {
      service_dir : local.service_dir
    }
    dockerfile = "images/go_service.Dockerfile"
  }
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
}
resource "docker_registry_image" "web_feature_consumer_remote_image" {
  name          = docker_image.web_feature_consumer_image.name
  keep_remotely = true
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
}

resource "google_storage_bucket_iam_member" "web_feature_consumer" {
  provider = google.internal_project
  bucket   = var.repo_bucket
  role     = "roles/storage.objectViewer"
  member   = google_service_account.web_feature_consumer_service_account.member
}

resource "google_service_account" "web_feature_consumer_service_account" {
  provider     = google.internal_project
  account_id   = "web-feature-consumer-${var.env_id}"
  display_name = "Web Feature Consumer service account for ${var.env_id}"
}

resource "google_project_iam_member" "gcp_datastore_user" {
  provider = google.internal_project
  role     = "roles/datastore.user"
  project  = var.datastore_info.project_id
  member   = google_service_account.web_feature_consumer_service_account.member
}

resource "google_spanner_database_iam_member" "gcp_spanner_user" {
  role     = "roles/spanner.databaseUser"
  provider = google.internal_project
  database = var.spanner_datails.database
  instance = var.spanner_datails.instance
  project  = var.spanner_datails.project_id
  member   = google_service_account.web_feature_consumer_service_account.member
}

resource "google_cloud_run_v2_service" "web_feature_service" {
  provider = google.internal_project
  count    = length(var.regions)
  name     = "${var.env_id}-${var.regions[count.index]}-web-feature-consumer-srv"
  location = var.regions[count.index]

  template {
    containers {
      image = "${docker_image.web_feature_consumer_image.name}@${docker_registry_image.web_feature_consumer_remote_image.sha256_digest}"
      env {
        name  = "BUCKET"
        value = var.repo_bucket
      }
      env {
        name  = "PROJECT_ID"
        value = var.datastore_info.project_id
      }
      env {
        name  = "DATASTORE_DATABASE"
        value = var.datastore_info.database_name
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
    service_account = google_service_account.web_feature_consumer_service_account.email
  }
  depends_on = [
    google_storage_bucket_iam_member.web_feature_consumer,
  ]
}

resource "google_cloud_run_v2_service_iam_member" "web_feature_step_invoker" {
  count    = length(var.regions)
  provider = google.internal_project
  location = var.regions[count.index]
  name     = google_cloud_run_v2_service.web_feature_service[count.index].name
  # name     = var.repo_downloader_step_region_to_step_info_map[var.regions[count.index]].name
  role   = "roles/run.invoker"
  member = google_service_account.service_account.member
}
