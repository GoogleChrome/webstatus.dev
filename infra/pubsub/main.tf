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

# Redelivery backoff for NACKed (or expired) messages on every consumer subscription.
# Without a retry policy Pub/Sub redelivers immediately, so a message that keeps
# failing (e.g. a permanent error misclassified as transient) is retried in a hot
# loop. With exponential backoff a failing message is retried at most roughly every
# 10 minutes, but it still stays unacknowledged. For the subscriptions listed in the
# "Pub/Sub High Latency" alert (queue_latency in alerts.tf: ingestion-jobs,
# notification-events, chime-delivery, vcs-scan-tasks, github-issue-delivery), that
# alert (oldest_unacked_message_age > 600s) still fires for messages that never succeed.
# batch-updates and webhook-delivery are not covered by that alert.
locals {
  subscription_retry_minimum_backoff = "10s"
  subscription_retry_maximum_backoff = "600s"
}

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

  retry_policy {
    minimum_backoff = local.subscription_retry_minimum_backoff
    maximum_backoff = local.subscription_retry_maximum_backoff
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

  retry_policy {
    minimum_backoff = local.subscription_retry_minimum_backoff
    maximum_backoff = local.subscription_retry_maximum_backoff
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

  retry_policy {
    minimum_backoff = local.subscription_retry_minimum_backoff
    maximum_backoff = local.subscription_retry_maximum_backoff
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

  retry_policy {
    minimum_backoff = local.subscription_retry_minimum_backoff
    maximum_backoff = local.subscription_retry_maximum_backoff
  }
}

# Main Topic: Webhook Delivery
resource "google_pubsub_topic" "webhook_delivery" {
  name    = "webhook-delivery-${var.env_id}"
  project = var.project_id
}

resource "google_pubsub_subscription" "webhook_delivery_sub" {
  name    = "webhook-delivery-sub-${var.env_id}"
  topic   = google_pubsub_topic.webhook_delivery.name
  project = var.project_id

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.delivery_dlq.id
    max_delivery_attempts = 5
  }

  retry_policy {
    minimum_backoff = local.subscription_retry_minimum_backoff
    maximum_backoff = local.subscription_retry_maximum_backoff
  }
}

# ==========================================
# 4. Code Subscriptions Pipeline
# ==========================================

# DLQ for Code Scanning & Issue Delivery
resource "google_pubsub_topic" "code_subscriptions_dlq" {
  name    = "code-subscriptions-dead-letter-${var.env_id}"
  project = var.project_id
}

resource "google_pubsub_subscription" "code_subscriptions_dlq_sub" {
  name    = "code-subscriptions-dead-letter-sub-${var.env_id}"
  topic   = google_pubsub_topic.code_subscriptions_dlq.name
  project = var.project_id
}

# Main Topic: VCS Scan Tasks (consumed by vcs_scanner worker)
resource "google_pubsub_topic" "vcs_scan_tasks" {
  name    = "vcs-scan-tasks-${var.env_id}"
  project = var.project_id
}

resource "google_pubsub_subscription" "vcs_scan_tasks_sub" {
  name    = "vcs-scan-tasks-sub-${var.env_id}"
  topic   = google_pubsub_topic.vcs_scan_tasks.name
  project = var.project_id

  # Scan tasks are idempotent "rescan this repository" triggers, and vcs-sync re-publishes
  # every repository on each run (vcs_sync_region_schedules), so older copies are redundant.
  # Keep retention short so a task that can never succeed cannot pile up thousands of copies
  # (the default is 7 days). Keep this longer than the vcs-sync interval.
  message_retention_duration = "3600s"

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.code_subscriptions_dlq.id
    max_delivery_attempts = 5
  }

  retry_policy {
    minimum_backoff = local.subscription_retry_minimum_backoff
    maximum_backoff = local.subscription_retry_maximum_backoff
  }
}

# Main Topic: GitHub Issue Delivery (consumed by github_issue_delivery worker)
resource "google_pubsub_topic" "github_issue_delivery" {
  name    = "github-issue-delivery-${var.env_id}"
  project = var.project_id
}

resource "google_pubsub_subscription" "github_issue_delivery_sub" {
  name    = "github-issue-delivery-sub-${var.env_id}"
  topic   = google_pubsub_topic.github_issue_delivery.name
  project = var.project_id

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.code_subscriptions_dlq.id
    max_delivery_attempts = 5
  }

  retry_policy {
    minimum_backoff = local.subscription_retry_minimum_backoff
    maximum_backoff = local.subscription_retry_maximum_backoff
  }
}
