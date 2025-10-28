# Copyright 2024 Google LLC
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

variable "go_module_path" {
  type = string
}

variable "env_id" {
  type = string
}

variable "regions" {
  type = list(string)
}

variable "region_schedules" {
  type = map(string)
}

variable "deletion_protection" {
  type = bool
}

variable "docker_repository_url" {
  type = string
}

variable "spanner_details" {
  type = object({
    instance   = string
    database   = string
    project_id = string
  })
}

variable "timeout_seconds" {
  type = number
}


variable "short_name" {
  type        = string
  description = "short name to describe resources."
  validation {
    condition     = length(var.short_name) <= 13
    error_message = "Short name must be 13 characters or less."
  }
}

variable "full_name" {
  type        = string
  description = "Full name of Workflow"
}

variable "image_name" {
  type = string
}

variable "project_id" {
  type = string
}

variable "env_vars" {
  type = list(object({
    name  = string
    value = string
  }))
}

variable "does_process_write_to_spanner" {
  type    = bool
  default = false
}

variable "does_process_write_to_datastore" {
  type    = bool
  default = false
}

# Refer to this document for the resource limit rules.
# By default, we use the minimum for the second generation cloud run processes.
variable "resource_job_limits" {
  type = object({
    cpu    = string
    memory = string
  })
  default = {
    cpu    = "1"
    memory = "512Mi"
  }

}

variable "notification_channel_ids" {
  description = "A list of notification channel ids to send alerts to."
  type        = list(string)
}
