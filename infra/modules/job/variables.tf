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

variable "env_id" {
  type = string
}

variable "regions" {
  type = list(string)
}

variable "image" {
  type = string
}

variable "timeout_seconds" {
  type = number
}

variable "short_name" {
  type = string
}

variable "full_name" {
  type = string
}

variable "deletion_protection" {
  type = bool
}

variable "env_vars" {
  type = list(object({
    name  = string
    value = string
  }))
}

variable "spanner_project_id" {
  type = string
}

# Refer to this document for the resource limit rules.
# By default, we use the minimum for the second generation cloud run processes.
variable "resource_limits" {
  type = object({
    cpu    = string
    memory = string
  })
  default = {
    cpu    = "1"
    memory = "512Mi"
  }

}

variable "does_process_write_to_spanner" {
  type = bool
}

variable "does_process_write_to_datastore" {
  type = bool
}

variable "otel_config_secret_id" {
  type        = string
  description = "The Secret Manager secret ID containing the OTel collector configuration"
}

variable "otel_project_id" {
  type        = string
  description = "The GCP project ID where telemetry traces/metrics will be exported"
}

variable "otel_collector_image" {
  type        = string
  description = "The container image to use for the OTel collector sidecar"
}

variable "otel_collector_config_mount_path" {
  type        = string
  description = "The volume mount path for the OTel collector configuration"
}
