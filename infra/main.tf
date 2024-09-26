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
  firebase_api_key = sensitive(try(data.google_secret_manager_secret_version_access.firebase_api_key[0].secret_data, ""))
}

data "google_secret_manager_secret_version_access" "firebase_api_key" {
  count    = var.firebase_api_key_location != "" ? 1 : 0
  provider = google.internal_project
  secret   = var.firebase_api_key_location
}

module "auth" {
  source = "./auth"
  providers = {
    google.internal_project = google.internal_project
  }
  env_id                  = var.env_id
  github_config_locations = var.auth_github_config_locations
}

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
  deletion_protection = var.deletion_protection
  # `gcloud spanner instance-configs list --project=<PROJECT>` returns the available configs
  spanner_region_id         = local.spanner_repository_region
  datastore_region_id       = var.datastore_region_id
  spanner_processing_units  = var.spanner_processing_units
  docker_repository_region  = local.docker_repository_region
  projects                  = var.projects
  depends_on                = [module.services]
  vpc_id                    = module.network.vpc_id
  region_to_subnet_info_map = module.network.region_to_subnet_info_map
}

module "ingestion" {
  source = "./ingestion"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }

  env_id                        = var.env_id
  docker_repository_details     = module.storage.docker_repository_details
  deletion_protection           = var.deletion_protection
  regions                       = keys(var.region_information)
  buckets                       = module.storage.buckets
  secret_ids                    = var.secret_ids
  datastore_info                = module.storage.datastore_info
  spanner_datails               = module.storage.spanner_info
  projects                      = var.projects
  depends_on                    = [module.services]
  wpt_region_schedules          = var.wpt_region_schedules
  bcd_region_schedules          = var.bcd_region_schedules
  uma_region_schedules          = var.uma_region_schedules
  chromium_region_schedules     = var.chromium_region_schedules
  web_features_region_schedules = var.web_features_region_schedules
}

module "backend" {
  source = "./backend"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }

  region_to_subnet_info_map            = module.network.region_to_subnet_info_map
  deletion_protection                  = var.deletion_protection
  env_id                               = var.env_id
  spanner_datails                      = module.storage.spanner_info
  docker_repository_details            = module.storage.docker_repository_details
  datastore_info                       = module.storage.datastore_info
  vpc_name                             = module.network.vpc_name
  ssl_certificates                     = var.ssl_certificates
  domains_for_gcp_managed_certificates = var.backend_domains_for_gcp_managed_certificates
  projects                             = var.projects
  cache_duration                       = var.cache_duration
  redis_env_vars                       = module.storage.redis_env_vars
  cors_allowed_origin                  = var.backend_cors_allowed_origin
}

module "frontend" {
  source = "./frontend"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }

  env_id                               = var.env_id
  deletion_protection                  = var.deletion_protection
  docker_repository_details            = module.storage.docker_repository_details
  backend_api_host                     = var.backend_api_url
  google_analytics_id                  = var.google_analytics_id
  region_to_subnet_info_map            = module.network.region_to_subnet_info_map
  vpc_name                             = module.network.vpc_name
  docker_build_target                  = var.frontend_docker_build_target
  ssl_certificates                     = var.ssl_certificates
  domains_for_gcp_managed_certificates = var.frontend_domains_for_gcp_managed_certificates
  projects                             = var.projects
  firebase_settings = {
    api_key     = local.firebase_api_key
    auth_domain = "${var.projects.internal}.firebaseapp.com"
    tenant_id   = module.auth.tenant_id
  }
}
