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

# ==========================================
# 1. Ingestion Pipeline (Search Refresh)
# ==========================================

# DLQ for Ingestion
resource "google_pubsub_topic" "ingestion_dlq" {
  name    = "ingestion-jobs-dead-letter-${var.env_id}"
  project = var.project_id
}

resource "google_pubsub_subscription" "ingestion_dlq_sub" {
  name    = "ingestion-jobs-dead-letter-sub-${var.env_id}"
  topic   = google_pubsub_topic.ingestion_dlq.name
  project = var.project_id
}

# Main Topic: Batch Updates (Triggers Fan-Out)
resource "google_pubsub_topic" "batch_updates" {
  name    = "batch-updates-${var.env_id}"
  project = var.project_id
}

resource "google_pubsub_subscription" "batch_updates_sub" {
  name    = "batch-updates-sub-${var.env_id}"
  topic   = google_pubsub_topic.batch_updates.name
  project = var.project_id

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.ingestion_dlq.id
    max_delivery_attempts = 5
  }
}

# Main Topic: Ingestion Jobs (Single Search Refresh)
resource "google_pubsub_topic" "ingestion_jobs" {
  name    = "ingestion-jobs-${var.env_id}"
  project = var.project_id
}

resource "google_pubsub_subscription" "ingestion_jobs_sub" {
  name    = "ingestion-jobs-sub-${var.env_id}"
  topic   = google_pubsub_topic.ingestion_jobs.name
  project = var.project_id

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.ingestion_dlq.id
    max_delivery_attempts = 5
  }
}

# ==========================================
# 2. Notification Pipeline (Event Processing)
# ==========================================

# DLQ for Notifications
resource "google_pubsub_topic" "notification_dlq" {
  name    = "notification-events-dead-letter-${var.env_id}"
  project = var.project_id
}

resource "google_pubsub_subscription" "notification_dlq_sub" {
  name    = "notification-events-dead-letter-sub-${var.env_id}"
  topic   = google_pubsub_topic.notification_dlq.name
  project = var.project_id
}

# Main Topic: Notification Events (Fan-Out to Dispatchers)
resource "google_pubsub_topic" "notification_events" {
  name    = "notification-events-${var.env_id}"
  project = var.project_id
}

resource "google_pubsub_subscription" "notification_events_sub" {
  name    = "notification-events-sub-${var.env_id}"
  topic   = google_pubsub_topic.notification_events.name
  project = var.project_id

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.notification_dlq.id
    max_delivery_attempts = 5
  }
}

# ==========================================
# 3. Delivery Pipeline (Email Sending)
# ==========================================

# DLQ for Delivery
resource "google_pubsub_topic" "delivery_dlq" {
  name    = "delivery-dead-letter-${var.env_id}"
  project = var.project_id
}

resource "google_pubsub_subscription" "delivery_dlq_sub" {
  name    = "delivery-dead-letter-sub-${var.env_id}"
  topic   = google_pubsub_topic.delivery_dlq.name
  project = var.project_id
}

# Main Topic: Chime Delivery (Email)
resource "google_pubsub_topic" "chime_delivery" {
  name    = "chime-delivery-${var.env_id}"
  project = var.project_id
}

resource "google_pubsub_subscription" "chime_delivery_sub" {
  name    = "chime-delivery-sub-${var.env_id}"
  topic   = google_pubsub_topic.chime_delivery.name
  project = var.project_id

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.delivery_dlq.id
    max_delivery_attempts = 5
  }
}
