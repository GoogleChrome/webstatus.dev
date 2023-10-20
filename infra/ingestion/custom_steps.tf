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

module "sample_custom_step" {
  source = "./common_workflow_steps/sample_step"

  env_id                    = var.env_id
  regions                   = var.regions
  docker_repository_details = var.docker_repository_details
}

module "repo_downloader_step" {
  source = "./common_workflow_steps/repo_downloader"

  env_id                    = var.env_id
  regions                   = var.regions
  docker_repository_details = var.docker_repository_details
  repo_bucket               = var.buckets.repo_download_bucket
  github_token_secret_id    = var.secret_ids.github_token
}