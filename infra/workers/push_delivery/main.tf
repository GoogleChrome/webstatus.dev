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

resource "google_service_account" "worker_sa" {
  account_id   = "push-delivery-${var.env_id}"
  provider     = google.internal_project
  display_name = "Push Delivery Service Account (${var.env_id})"
  project      = var.project_id
}

resource "google_cloud_run_v2_worker_pool" "worker" {
  for_each            = var.regions
  name                = "push-delivery-${var.env_id}-${each.key}"
  location            = each.key
  project             = var.project_id
  provider            = google.internal_project
  launch_stage        = "BETA"
  deletion_protection = var.deletion_protection

  scaling {
    manual_instance_count = var.manual_instance_count
  }
  template {
    service_account = google_service_account.worker_sa.email

    containers {
      image = var.image_url

      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
      env {
        name  = "SPANNER_INSTANCE"
        value = var.spanner_instance_id
      }
      env {
        name  = "SPANNER_DATABASE"
        value = var.spanner_database_id
      }
      env {
        name  = "NOTIFICATION_SUBSCRIPTION_ID"
        value = var.notification_subscription_id
      }
      env {
        name  = "EMAIL_TOPIC_ID"
        value = var.email_topic_id
      }
      env {
        name  = "WEBHOOK_TOPIC_ID"
        value = var.webhook_topic_id
      }
      env {
        name  = "OTEL_SERVICE_NAME"
        value = "push-delivery"
      }
      env {
        name  = "OTEL_GCP_PROJECT_ID"
        value = var.otel_project_id
      }
      env {
        name  = "OTEL_EXPORTER_OTLP_ENDPOINT"
        value = var.otel_collector_endpoint
      }
    }
    containers {
      name  = "otel"
      image = var.otel_collector_image
      args  = ["--config=${var.otel_collector_config_mount_path}/config.yaml"]
      env {
        name  = "OTEL_COLLECTOR_REGION"
        value = each.key
      }
      volume_mounts {
        name       = "otel-config"
        mount_path = var.otel_collector_config_mount_path
      }
    }
    volumes {
      name = "otel-config"
      secret {
        secret = var.otel_config_secret_id
        items {
          version = "latest"
          path    = "config.yaml"
        }
      }
    }
  }
}
