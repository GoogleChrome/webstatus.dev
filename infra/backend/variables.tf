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

variable "spanner_datails" {
  type = object({
    instance = string
    database = string
  })
}

variable "datastore_info" {
  type = object({
    database_name = string
    project_id    = string
  })
}


variable "env_id" {
  type = string
}

variable "vpc_name" {
  type = string
}

variable "region_to_subnet_info_map" {
  type = map(object({
    internal = string
    public   = string
  }))
}

variable "docker_repository_details" {
  type = object({
    hostname = string
    url      = string
    location = string
    name     = string
  })
}

variable "ssl_certificates" {
  type = list(string)
}

variable "domains_for_gcp_managed_certificates" {
  type = list(string)
}

variable "projects" {
  type = object({
    host     = string
    internal = string
    public   = string
  })
}

variable "redis_env_vars" {
  type = map(object({
    host = string
    port = number
  }))
  description = "Map of Redis host and port per region"
}

variable "cache_duration" {
  type = string
}

variable "cors_allowed_origin" {
  type = string
}

variable "deletion_protection" {
  type = bool
}
