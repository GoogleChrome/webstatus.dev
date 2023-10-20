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


module "storage" {
  source = "./storage"

  env_id              = var.env_id
  deletion_protection = false
  # `gcloud spanner instance-configs list --project=<PROJECT>` returns the available configs
  spanner_region_id        = coalesce(var.spanner_region_override, "regional-${var.regions[0]}")
  spanner_processing_units = var.spanner_processing_units
  docker_repository_region = coalesce(var.docker_repository_region_override, var.regions[0])
}

module "ingestion" {
  source = "./ingestion"

  env_id                    = var.env_id
  docker_repository_details = module.storage.docker_repository_details
  regions                   = var.regions
  buckets                   = module.storage.buckets
  secret_ids                = var.secret_ids
  firestore_info            = module.storage.firestore_info
}

module "backend" {
  source = "./backend"

  env_id                    = var.env_id
  spanner_datails           = module.storage.spanner_info
  docker_repository_details = module.storage.docker_repository_details
  regions                   = var.regions
  firestore_info            = module.storage.firestore_info
}

module "frontend" {
  source = "./frontend"

  env_id                    = var.env_id
  docker_repository_details = module.storage.docker_repository_details
  regions                   = var.regions
  backend_api_host          = "TODO"
}