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
  service_dir = "workflows/steps/services/common/repo_downloader"
}
resource "docker_image" "repo_downloader_image" {
  name = "${var.docker_repository_details.url}/repo_downloader_image"
  build {
    context = "${path.cwd}/.."
    build_args = {
      service_dir : local.service_dir
    }
    dockerfile = "${path.cwd}/../images/go_service.Dockerfile"
    # Use buildx default builder instead of legacy builder
    # Needed for the --mount args
    # Must also specify platform too.
    builder        = "default"
    platform       = "linux/amd64"
    build_log_file = "${path.cwd}/repo_downloader_image.log"
  }
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
}
resource "docker_registry_image" "repo_downloader_remote_image" {
  name          = docker_image.repo_downloader_image.name
  keep_remotely = true
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
}

resource "google_storage_bucket_iam_member" "iam_member" {
  provider = google.internal_project
  bucket   = var.repo_bucket
  role     = "roles/storage.objectUser"
  member   = google_service_account.service_account.member
}

resource "google_secret_manager_secret_iam_member" "iam_member" {
  provider  = google.internal_project
  secret_id = data.google_secret_manager_secret.github_token.id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.service_account.member
}

resource "google_service_account" "service_account" {
  provider     = google.internal_project
  account_id   = "repo-downloader-${var.env_id}"
  display_name = "Repo Downloader service account for ${var.env_id}"
}

data "google_secret_manager_secret" "github_token" {
  provider  = google.internal_project
  secret_id = var.github_token_secret_id
}

resource "google_cloud_run_v2_service" "service" {
  provider = google.internal_project
  count    = length(var.regions)
  name     = "${var.env_id}-${var.regions[count.index]}-repo-downloader-srv"
  location = var.regions[count.index]

  template {
    containers {
      image = "${docker_image.repo_downloader_image.name}@${docker_registry_image.repo_downloader_remote_image.sha256_digest}"
      env {
        name = "GITHUB_TOKEN"
        value_source {
          secret_key_ref {
            secret  = data.google_secret_manager_secret.github_token.id
            version = "latest"
          }
        }
      }
      env {
        name  = "BUCKET"
        value = var.repo_bucket
      }
    }
    service_account = google_service_account.service_account.email
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }

  depends_on = [
    google_storage_bucket_iam_member.iam_member,
    google_secret_manager_secret_iam_member.iam_member,
  ]

  deletion_protection = var.deletion_protection
}
