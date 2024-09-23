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

# TODO. Once this workflow is changed from a web services to a job, use the same
# single stage workflow as the others.
module "web_features_repo_workflow" {
  source = "./workflows/web_features_repo"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }

  regions                                      = var.regions
  deletion_protection                          = var.deletion_protection
  env_id                                       = var.env_id
  repo_downloader_step_region_to_step_info_map = module.repo_downloader_step.region_to_step_info_map
  datastore_info                               = var.datastore_info
  spanner_datails                              = var.spanner_datails
  repo_bucket                                  = var.buckets.repo_download_bucket
  docker_repository_details                    = var.docker_repository_details
  region_schedules                             = var.web_features_region_schedules
}

module "wpt_workflow" {
  source = "../modules/single_stage_go_workflow"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }
  regions                         = var.regions
  short_name                      = "wpt-consumer"
  full_name                       = "WPT Workflow"
  deletion_protection             = var.deletion_protection
  project_id                      = var.spanner_datails.project_id
  timeout_seconds                 = 86400 # 24 hours
  image_name                      = "wpt_consumer_image"
  spanner_details                 = var.spanner_datails
  env_id                          = var.env_id
  region_schedules                = var.wpt_region_schedules
  docker_repository_url           = var.docker_repository_details.url
  go_module_path                  = "workflows/steps/services/wpt_consumer"
  does_process_write_to_spanner   = true
  does_process_write_to_datastore = true
  resource_job_limits = {
    cpu    = "2"
    memory = "1024Mi"
  }
  env_vars = [
    {
      name  = "PROJECT_ID"
      value = var.spanner_datails.project_id
    },
    {
      name  = "SPANNER_DATABASE"
      value = var.spanner_datails.database
    },
    {
      name  = "SPANNER_INSTANCE"
      value = var.spanner_datails.instance
    },
    {
      name  = "DATASTORE_DATABASE"
      value = var.datastore_info.database_name
    },
    {
      name  = "DATA_WINDOW_DURATION"
      value = "17520h" # 2 years
    }
  ]
}

module "bcd_workflow" {
  source = "../modules/single_stage_go_workflow"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }
  regions                       = var.regions
  short_name                    = "bcd-consumer"
  full_name                     = "BCD Workflow"
  deletion_protection           = var.deletion_protection
  project_id                    = var.spanner_datails.project_id
  timeout_seconds               = 60 * 5 # 5 minutes
  image_name                    = "bcd_consumer_image"
  spanner_details               = var.spanner_datails
  env_id                        = var.env_id
  region_schedules              = var.bcd_region_schedules
  docker_repository_url         = var.docker_repository_details.url
  go_module_path                = "workflows/steps/services/bcd_consumer"
  does_process_write_to_spanner = true
  env_vars = [
    {
      name  = "PROJECT_ID"
      value = var.spanner_datails.project_id
    },
    {
      name  = "SPANNER_DATABASE"
      value = var.spanner_datails.database
    },
    {
      name  = "SPANNER_INSTANCE"
      value = var.spanner_datails.instance
    }
  ]
}

