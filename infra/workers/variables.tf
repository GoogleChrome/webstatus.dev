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

variable "internal_project_id" {
  description = "The internal project ID (where Spanner/PubSub live)"
  type        = string
}

variable "env_id" {
  description = "Environment ID"
  type        = string
}

variable "regions" {
  description = "List of region names to deploy workers into"
  type        = set(string)
}


variable "docker_repository_details" {
  description = "Docker repository details"
  type = object({
    url = string
  })
}

variable "spanner_details" {
  type = object({
    instance = string
    database = string
  })
}

variable "state_bucket_name" {
  type = string
}

variable "pubsub_details" {
  type = object({
    ingestion_subscription_id    = string
    ingestion_topic_id           = string
    batch_topic_id               = string
    batch_subscription_id        = string
    notification_topic_id        = string
    notification_subscription_id = string
    email_topic_id               = string
    email_subscription_id        = string
  })
}

variable "worker_instance_count" {
  description = "Number of instances for manual scaling"
  type = object({
    event_producer_count = number
    push_delivery_count  = number
    email_count          = number
  })
}

variable "email_service_account_email" {
  description = "Pre-existing Service Account email for the Email Worker"
  type        = string
}

variable "frontend_base_url" {
  description = "Frontend base URL"
  type        = string
}

variable "deletion_protection" {
  type = bool
}

variable "chime_details" {
  type = object({
    env                     = string
    bcc_secret_ref          = string
    from_address_secret_ref = string
  })
}
