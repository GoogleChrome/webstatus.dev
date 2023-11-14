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

module "services" {
  source = "./services"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }

  projects = var.projects
}

module "network" {
  source = "./network"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }
  env_id               = var.env_id
  host_project_id      = var.projects.host
  region_to_subnet_map = local.region_to_subnet_map
  depends_on           = [module.services]
}

module "storage" {
  source = "./storage"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }

  env_id              = var.env_id
  deletion_protection = false
  # `gcloud spanner instance-configs list --project=<PROJECT>` returns the available configs
  spanner_region_id        = local.spanner_repository_region
  datastore_region_id      = var.datastore_region_id
  spanner_processing_units = var.spanner_processing_units
  docker_repository_region = local.docker_repository_region
  projects                 = var.projects
  depends_on               = [module.services]
}

module "ingestion" {
  source = "./ingestion"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }

  env_id                    = var.env_id
  docker_repository_details = module.storage.docker_repository_details
  regions                   = keys(var.region_information)
  buckets                   = module.storage.buckets
  secret_ids                = var.secret_ids
  datastore_info            = module.storage.datastore_info
  projects                  = var.projects
  depends_on                = [module.services]
}

module "backend" {
  source = "./backend"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }

  region_to_subnet_info_map = module.network.region_to_subnet_info_map
  env_id                    = var.env_id
  spanner_datails           = module.storage.spanner_info
  docker_repository_details = module.storage.docker_repository_details
  datastore_info            = module.storage.datastore_info
  vpc_name                  = module.network.vpc_name
}

module "frontend" {
  source = "./frontend"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }

  env_id                    = var.env_id
  docker_repository_details = module.storage.docker_repository_details
  backend_api_host          = "TODO"
  region_to_subnet_info_map = module.network.region_to_subnet_info_map
  vpc_name                  = module.network.vpc_name
}