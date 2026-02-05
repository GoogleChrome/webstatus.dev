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

variable "project_id" { type = string }
variable "env_id" { type = string }
variable "image_url" { type = string }
variable "spanner_instance_id" { type = string }
variable "spanner_database_id" { type = string }
variable "email_subscription_id" { type = string }
variable "manual_instance_count" { type = number }
variable "service_account_email" { type = string }
variable "regions" { type = set(string) }
variable "deletion_protection" { type = bool }
variable "frontend_base_url" { type = string }
variable "chime_env" { type = string }
variable "chime_bcc_secret_ref" { type = string }
variable "from_address_secret_ref" { type = string }
