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

variable "docker_repository_details" {
  type = object({
    hostname = string
    url      = string
    location = string
    name     = string
  })
}

variable "secret_ids" {
  type = object({
    github_token = string
  })
}

variable "datastore_info" {
  type = object({
    database_name = string
    project_id    = string
  })
}

variable "projects" {
  type = object({
    host     = string
    internal = string
    public   = string
  })
}

variable "spanner_datails" {
  type = object({
    project_id = string
    instance   = string
    database   = string
  })
}

variable "bcd_region_schedules" {
  type = map(string)
}

variable "chromium_region_schedules" {
  type = map(string)
}

variable "developer_signals_region_schedules" {
  type = map(string)
}

variable "uma_region_schedules" {
  type = map(string)
}

variable "wpt_region_schedules" {
  type = map(string)
}

variable "web_features_region_schedules" {
  type = map(string)
}

variable "web_features_mapping_region_schedules" {
  type = map(string)
}

variable "deletion_protection" {
  type = bool
}

variable "notification_channel_ids" {
  description = "A list of notification channel ids to send alerts to."
  type        = list(string)
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

variable "otel_collector_endpoint" {
  type        = string
  description = "The endpoint for the application to export OTLP metrics/traces to the local collector"
}
