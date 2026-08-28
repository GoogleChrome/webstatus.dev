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

# --- 1. Event Producer ---

# Build Image
module "event_producer_image" {
  source                = "../modules/go_image"
  image_name            = "event_producer"
  go_module_path        = "workers/event_producer"
  binary_type           = "job"
  docker_repository_url = var.docker_repository_details.url
}

# Deploy Service (Multi-Region)
module "event_producer" {
  source = "./event_producer"
  providers = {
    google.internal_project = google.internal_project
  }

  project_id = var.internal_project_id
  env_id     = var.env_id
  image_url  = module.event_producer_image.remote_image

  spanner_instance_id = var.spanner_details.instance
  spanner_database_id = var.spanner_details.database
  state_bucket_name   = var.state_bucket_name

  ingestion_subscription_id = var.pubsub_details.ingestion_subscription_id
  ingestion_topic_id        = var.pubsub_details.ingestion_topic_id
  batch_subscription_id     = var.pubsub_details.batch_subscription_id
  notification_topic_id     = var.pubsub_details.notification_topic_id

  manual_instance_count = var.worker_instance_count.event_producer_count
  regions               = var.regions

  deletion_protection              = var.deletion_protection
  otel_config_secret_id            = var.otel_config_secret_id
  otel_project_id                  = var.otel_project_id
  otel_collector_image             = var.otel_collector_image
  otel_collector_config_mount_path = var.otel_collector_config_mount_path
  otel_collector_endpoint          = var.otel_collector_endpoint
}

# --- 2. Push Delivery ---

# Build Image
module "push_delivery_image" {
  source                = "../modules/go_image"
  image_name            = "push_delivery"
  go_module_path        = "workers/push_delivery"
  binary_type           = "job"
  docker_repository_url = var.docker_repository_details.url
}

# Deploy Service (Multi-Region)
module "push_delivery" {
  source = "./push_delivery"
  providers = {
    google.internal_project = google.internal_project
  }

  project_id = var.internal_project_id
  env_id     = var.env_id
  image_url  = module.push_delivery_image.remote_image

  spanner_instance_id = var.spanner_details.instance
  spanner_database_id = var.spanner_details.database

  notification_subscription_id = var.pubsub_details.notification_subscription_id
  email_topic_id               = var.pubsub_details.email_topic_id
  webhook_topic_id             = var.pubsub_details.webhook_topic_id

  manual_instance_count = var.worker_instance_count.push_delivery_count
  regions               = var.regions

  deletion_protection              = var.deletion_protection
  otel_config_secret_id            = var.otel_config_secret_id
  otel_project_id                  = var.otel_project_id
  otel_collector_image             = var.otel_collector_image
  otel_collector_config_mount_path = var.otel_collector_config_mount_path
  otel_collector_endpoint          = var.otel_collector_endpoint
}

# --- 3. Email Worker ---

# Build Image
module "email_image" {
  source                = "../modules/go_image"
  image_name            = "email"
  go_module_path        = "workers/email"
  binary_type           = "job"
  docker_repository_url = var.docker_repository_details.url
}

# Deploy Service (Multi-Region)
module "email" {
  source = "./email"
  providers = {
    google.internal_project = google.internal_project
  }

  project_id = var.internal_project_id
  env_id     = var.env_id
  image_url  = module.email_image.remote_image

  spanner_instance_id = var.spanner_details.instance
  spanner_database_id = var.spanner_details.database

  email_subscription_id = var.pubsub_details.email_subscription_id

  manual_instance_count = var.worker_instance_count.email_count

  service_account_email = var.email_service_account_email
  regions               = var.regions

  frontend_base_url   = var.frontend_base_url
  deletion_protection = var.deletion_protection

  chime_env                        = var.chime_details.env
  chime_bcc_secret_ref             = var.chime_details.bcc_secret_ref
  from_address_secret_ref          = var.chime_details.from_address_secret_ref
  otel_config_secret_id            = var.otel_config_secret_id
  otel_project_id                  = var.otel_project_id
  otel_collector_image             = var.otel_collector_image
  otel_collector_config_mount_path = var.otel_collector_config_mount_path
  otel_collector_endpoint          = var.otel_collector_endpoint
}

# --- 4. Webhook Worker ---

# Build Image
module "webhook_image" {
  source                = "../modules/go_image"
  image_name            = "webhook"
  go_module_path        = "workers/webhook"
  binary_type           = "job"
  docker_repository_url = var.docker_repository_details.url
}

# Deploy Service (Multi-Region)
module "webhook" {
  source = "./webhook"
  providers = {
    google.internal_project = google.internal_project
  }

  project_id = var.internal_project_id
  env_id     = var.env_id
  image_url  = module.webhook_image.remote_image

  spanner_instance_id = var.spanner_details.instance
  spanner_database_id = var.spanner_details.database

  webhook_subscription_id = var.pubsub_details.webhook_subscription_id

  manual_instance_count = var.worker_instance_count.webhook_count
  regions               = var.regions

  frontend_base_url = var.frontend_base_url

  deletion_protection              = var.deletion_protection
  otel_config_secret_id            = var.otel_config_secret_id
  otel_project_id                  = var.otel_project_id
  otel_collector_image             = var.otel_collector_image
  otel_collector_config_mount_path = var.otel_collector_config_mount_path
  otel_collector_endpoint          = var.otel_collector_endpoint
}

# --- 5. VCS Scanner Worker ---

# Build Image
module "vcs_scanner_image" {
  source                = "../modules/go_image"
  image_name            = "vcs_scanner"
  go_module_path        = "workers/vcs_scanner"
  binary_type           = "job"
  docker_repository_url = var.docker_repository_details.url
}

# Deploy Service (Multi-Region)
module "vcs_scanner" {
  source = "./vcs_scanner"
  providers = {
    google.internal_project = google.internal_project
  }

  project_id = var.internal_project_id
  env_id     = var.env_id
  image_url  = module.vcs_scanner_image.remote_image

  spanner_instance_id = var.spanner_details.instance
  spanner_database_id = var.spanner_details.database

  vcs_scan_tasks_subscription_id   = var.pubsub_details.vcs_scan_tasks_subscription_id
  github_app_id                    = var.github_app_id
  github_app_private_key_secret_id = var.github_app_private_key_secret_id

  manual_instance_count = var.worker_instance_count.vcs_scanner_count
  regions               = var.regions

  deletion_protection              = var.deletion_protection
  otel_config_secret_id            = var.otel_config_secret_id
  otel_project_id                  = var.otel_project_id
  otel_collector_image             = var.otel_collector_image
  otel_collector_config_mount_path = var.otel_collector_config_mount_path
  otel_collector_endpoint          = var.otel_collector_endpoint
}

# --- 6. GitHub Issue Delivery Worker ---

# Build Image
module "github_issue_delivery_image" {
  source                = "../modules/go_image"
  image_name            = "github_issue_delivery"
  go_module_path        = "workers/github_issue_delivery"
  binary_type           = "job"
  docker_repository_url = var.docker_repository_details.url
}

# Deploy Service (Multi-Region)
module "github_issue_delivery" {
  source = "./github_issue_delivery"
  providers = {
    google.internal_project = google.internal_project
  }

  project_id = var.internal_project_id
  env_id     = var.env_id
  image_url  = module.github_issue_delivery_image.remote_image

  spanner_instance_id = var.spanner_details.instance
  spanner_database_id = var.spanner_details.database

  github_issue_delivery_subscription_id = var.pubsub_details.github_issue_delivery_subscription_id
  github_app_id                         = var.github_app_id
  github_app_private_key_secret_id      = var.github_app_private_key_secret_id

  manual_instance_count = var.worker_instance_count.github_issue_delivery_count
  regions               = var.regions

  deletion_protection              = var.deletion_protection
  frontend_base_url                = var.frontend_base_url
  otel_config_secret_id            = var.otel_config_secret_id
  otel_project_id                  = var.otel_project_id
  otel_collector_image             = var.otel_collector_image
  otel_collector_config_mount_path = var.otel_collector_config_mount_path
  otel_collector_endpoint          = var.otel_collector_endpoint
}
