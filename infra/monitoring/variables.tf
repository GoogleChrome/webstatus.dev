# Copyright 2026 Google LLC
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

variable "project_id" {
  description = "The project ID to deploy to"
  type        = string
}

variable "env_id" {
  description = "The deployment environment identifier (e.g. dev, staging, prod)"
  type        = string
}

variable "target_host" {
  description = "The public target hostname for synthetic uptime checks"
  type        = string
}

variable "notification_channel_ids" {
  description = "List of GCP Cloud Monitoring notification channel IDs for alerting"
  type        = list(string)
  default     = []
}
