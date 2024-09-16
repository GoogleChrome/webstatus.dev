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

# Variables specific to this module.
variable "timeout_seconds" {
  description = "Timeout for the UMA Export job. Details here: https://cloud.google.com/run/docs/configuring/request-timeout#terraform"
  type        = number
  default     = "300" # 5 minutes
}

# Variables from parent modules.
# Refer to the parent modules for more details
variable "env_id" {
  type = string
}

variable "regions" {
  type = list(string)
}

variable "spanner_datails" {
  type = object({
    instance   = string
    database   = string
    project_id = string
  })
}

variable "docker_repository_details" {
  type = object({
    hostname = string
    url      = string
    location = string
    name     = string
  })
}

variable "region_schedules" {
  type = map(string)
}

variable "deletion_protection" {
  type = bool
}
