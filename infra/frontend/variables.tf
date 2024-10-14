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

variable "backend_api_host" {
  type = string
}

variable "google_analytics_id" {
  type = string
}

variable "custom_ssl_certificates" {
  type = list(string)
}

variable "docker_build_target" {
  type = string
}

variable "domains" {
  type = list(string)
}

variable "projects" {
  type = object({
    host     = string
    internal = string
    public   = string
  })
}

variable "deletion_protection" {
  type = bool
}

variable "firebase_settings" {
  type = object({
    auth_domain = string
    api_key     = string
    tenant_id   = string
  })
}
