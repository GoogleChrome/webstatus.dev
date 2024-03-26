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

variable "regions" {
  type = list(string)
}


variable "datastore_info" {
  type = object({
    database_name = string
    project_id    = string
  })
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

variable "data_window_duration" {
  description = "How far back in time do we want to go. Units come from https://pkg.go.dev/time#ParseDuration"
  default     = "17520h" # 2 years
}

variable "timeout_seconds" {
  description = "Timeout for the WPT step."
  type        = number
  default     = "3600" # An hour
}
