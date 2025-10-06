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

variable "project_name" {
  type        = string
  description = "The ID of the Google Cloud project"
}

variable "spanner_region_override" {
  type     = string
  nullable = true
  default  = null
}

variable "spanner_processing_units" {
  type = number
}

variable "deletion_protection" {
  type        = bool
  description = "Protect applicable resources from deletion."
}

variable "env_id" {
  description = "Environment ID. Commonly dervied from the branch name"
  type        = string
}

variable "datastore_region_id" {
  type = string
}

variable "docker_repository_region_override" {
  type     = string
  nullable = true
  default  = null
}

locals {
  docker_repository_region = coalesce(
    var.docker_repository_region_override,
  keys(var.region_information)[0])
  spanner_repository_region = coalesce(
    var.spanner_region_override,
  "regional-${keys(var.region_information)[0]}")
  region_to_subnet_map = { for region, info in var.region_information : region => info.networks }
}

variable "secret_ids" {
  type = object({
    github_token = string
  })
}

variable "projects" {
  type = object({
    host     = string
    internal = string
    public   = string
  })
}

variable "region_information" {
  type = map(object({
    networks = object({
      internal = object({
        cidr = string
      })
      public = object({
        cidr = string
      })
    })
  }))
}

variable "backend_api_url" {
  type = string
}

variable "google_analytics_id" {
  type = string
}

variable "frontend_docker_build_target" {
  type        = string
  description = "Dockerfile target for the frontend image"
}

variable "custom_ssl_certificates_for_frontend" {
  type        = list(string)
  description = "List of custom SSL certs for frontend that are not managed by GCP"
}

variable "custom_ssl_certificates_for_backend" {
  type        = list(string)
  description = "List of custom SSL certs for backend that are not managed by GCP"
}


variable "frontend_domains" {
  type        = list(string)
  description = "List of domains for the frontend"
}

variable "backend_domains" {
  type        = list(string)
  description = "List of domains for the backend"
}

variable "backend_cache_settings" {
  type = object({
    default_duration                  = string
    aggregated_feature_stats_duration = string
  })
  description = "Various cache settings for backend"
}

variable "backend_cors_allowed_origin" {
  type = string
}

variable "bcd_region_schedules" {
  type = map(string)
}

variable "developer_signals_region_schedules" {
  type = map(string)
}

variable "wpt_region_schedules" {
  type = map(string)
}

variable "uma_region_schedules" {
  type = map(string)
}

variable "chromium_region_schedules" {
  type = map(string)
}

variable "web_features_region_schedules" {
  type = map(string)
}


variable "firebase_api_key_location" {
  description = "Location of the firebase api key in secret manager"
  type        = string
}

variable "auth_github_config_locations" {
  description = "Location of the github configuration in secret manager"
  type = object({
    client_id     = string
    client_secret = string
  })
}

variable "backend_min_instance_count" {
  type        = number
  description = "Minimum instance count for backend instances"
}

variable "backend_max_instance_count" {
  type        = number
  description = "Maximum instance count for backend instances"
  # Use default max of 100.
  # https://cloud.google.com/run/docs/configuring/max-instances#setting
  default = 100
}


variable "frontend_min_instance_count" {
  type        = number
  description = "Minimum instance count for frontend instances"
}

variable "frontend_max_instance_count" {
  type        = number
  description = "Maximum instance count for frontend instances"
  # Use default max of 100.
  # https://cloud.google.com/run/docs/configuring/max-instances#setting
  default = 100
}

variable "notification_channel_ids" {
  description = "A list of notification channel ids to send alerts to."
  type        = list(string)
}
