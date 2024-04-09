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

module "web_features_repo_workflow" {
  source = "./workflows/web_features_repo"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }

  regions                                      = var.regions
  env_id                                       = var.env_id
  repo_downloader_step_region_to_step_info_map = module.repo_downloader_step.region_to_step_info_map
  datastore_info                               = var.datastore_info
  spanner_datails                              = var.spanner_datails
  repo_bucket                                  = var.buckets.repo_download_bucket
  docker_repository_details                    = var.docker_repository_details
}

module "wpt_workflow" {
  source = "./workflows/wpt"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }

  regions                   = var.regions
  env_id                    = var.env_id
  datastore_info            = var.datastore_info
  spanner_datails           = var.spanner_datails
  docker_repository_details = var.docker_repository_details
}

module "bcd_workflow" {
  source = "./workflows/bcd"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }

  regions                   = var.regions
  env_id                    = var.env_id
  spanner_datails           = var.spanner_datails
  docker_repository_details = var.docker_repository_details
}
