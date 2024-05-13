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

variable "gsi_client_id" {
  type = string
}

variable "google_analytics_id" {
  type = string
}

variable "frontend_docker_build_target" {
  type        = string
  description = "Dockerfile target for the frontend image"
}

variable "ssl_certificates" {
  type        = list(string)
  description = "List of custom SSL certs"
}


variable "frontend_domains_for_gcp_managed_certificates" {
  type        = list(string)
  description = "List of domains for the frontend that GCP should manage certs for."
}

variable "backend_domains_for_gcp_managed_certificates" {
  type        = list(string)
  description = "List of domains for the backend that GCP should manage certs for."
}

variable "cache_duration" {
  type        = string
  description = "TTL for entries that are cached"
}

variable "backend_cors_allowed_origin" {
  type = string
}
