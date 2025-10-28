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

module "web_features_workflow" {
  source = "../modules/single_stage_go_workflow"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }
  regions                         = var.regions
  short_name                      = "web-features"
  full_name                       = "Web Features Workflow"
  deletion_protection             = var.deletion_protection
  project_id                      = var.spanner_datails.project_id
  timeout_seconds                 = 7200 # 2 hours
  image_name                      = "web_features_consumer_image"
  spanner_details                 = var.spanner_datails
  notification_channel_ids        = var.notification_channel_ids
  env_id                          = var.env_id
  region_schedules                = var.web_features_region_schedules
  docker_repository_url           = var.docker_repository_details.url
  go_module_path                  = "workflows/steps/services/web_feature_consumer"
  does_process_write_to_spanner   = true
  does_process_write_to_datastore = true
  resource_job_limits = {
    cpu    = "4"
    memory = "2Gi"
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
      value = "262980h" # 30 years in hours (365.25*30*24)
    }
  ]
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
  notification_channel_ids        = var.notification_channel_ids
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
  notification_channel_ids      = var.notification_channel_ids
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

module "chromium_enum_workflow" {
  source = "../modules/single_stage_go_workflow"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }
  regions                       = var.regions
  short_name                    = "chromium-enum"
  full_name                     = "Chromium Enum Workflow"
  deletion_protection           = var.deletion_protection
  project_id                    = var.spanner_datails.project_id
  timeout_seconds               = 60 * 5 # 5 minutes
  image_name                    = "chromium_enum_consumer_image"
  spanner_details               = var.spanner_datails
  notification_channel_ids      = var.notification_channel_ids
  env_id                        = var.env_id
  region_schedules              = var.chromium_region_schedules
  docker_repository_url         = var.docker_repository_details.url
  go_module_path                = "workflows/steps/services/chromium_histogram_enums"
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

module "uma_export_workflow" {
  source = "../modules/single_stage_go_workflow"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }
  regions                       = var.regions
  short_name                    = "uma-consumer"
  full_name                     = "UMA Export Workflow"
  deletion_protection           = var.deletion_protection
  project_id                    = var.spanner_datails.project_id
  timeout_seconds               = 60 * 5 # 5 minutes
  image_name                    = "uma_export_consumer_image"
  spanner_details               = var.spanner_datails
  notification_channel_ids      = var.notification_channel_ids
  env_id                        = var.env_id
  region_schedules              = var.uma_region_schedules
  docker_repository_url         = var.docker_repository_details.url
  go_module_path                = "workflows/steps/services/uma_export"
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

module "developer_signals_workflow" {
  source = "../modules/single_stage_go_workflow"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }
  regions                       = var.regions
  short_name                    = "dev-signals" # Needs to be short enough to create a service account
  full_name                     = "Developer Signals Workflow"
  deletion_protection           = var.deletion_protection
  project_id                    = var.spanner_datails.project_id
  timeout_seconds               = 60 * 10 # 5 minutes
  image_name                    = "developer_signals_consumer_image"
  spanner_details               = var.spanner_datails
  notification_channel_ids      = var.notification_channel_ids
  env_id                        = var.env_id
  region_schedules              = var.developer_signals_region_schedules
  docker_repository_url         = var.docker_repository_details.url
  go_module_path                = "workflows/steps/services/developer_signals_consumer"
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

module "web_features_mapping_workflow" {
  source = "../modules/single_stage_go_workflow"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }
  regions                       = var.regions
  short_name                    = "feature-map"
  full_name                     = "Web Features Mapping Workflow"
  deletion_protection           = var.deletion_protection
  project_id                    = var.spanner_datails.project_id
  timeout_seconds               = 60 * 10 # 10 minutes
  image_name                    = "web_features_mapping_consumer_image"
  spanner_details               = var.spanner_datails
  notification_channel_ids      = var.notification_channel_ids
  env_id                        = var.env_id
  region_schedules              = var.web_features_mapping_region_schedules
  docker_repository_url         = var.docker_repository_details.url
  go_module_path                = "workflows/steps/services/web_features_mapping_consumer"
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
