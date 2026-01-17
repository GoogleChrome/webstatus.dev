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
# 1. Critical: Dead Letter Queue Monitoring
# ==========================================
# Triggers if ANY message lands in a DLQ. This indicates a permanent failure
# that requires manual investigation.

resource "google_monitoring_alert_policy" "dlq_non_empty" {
  display_name = "Pub/Sub DLQ Non-Empty (${var.env_id})"
  combiner     = "OR"

  conditions {
    display_name = "Messages waiting in DLQ"
    condition_threshold {
      filter = join(" AND ", [
        "resource.type=\"pubsub_subscription\"",
        "metric.type=\"pubsub.googleapis.com/subscription/num_undelivered_messages\"",
        "resource.label.subscription_id = one_of(\"${google_pubsub_subscription.ingestion_dlq_sub.name}\", \"${google_pubsub_subscription.notification_dlq_sub.name}\", \"${google_pubsub_subscription.delivery_dlq_sub.name}\")"
      ])

      duration        = "60s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0

      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_MAX"
      }
    }
  }

  notification_channels = var.notification_channel_ids

  documentation {
    content = "Messages have landed in a Dead Letter Queue. This means they failed processing 5 times. Check worker logs for panic/errors."
  }
}

# ==========================================
# 2. Warning: Worker Latency / Stuck Processing
# ==========================================
# Triggers if the oldest message in the main queues is older than 10 minutes.
# This indicates workers are down, stuck, or overwhelmed.

resource "google_monitoring_alert_policy" "queue_latency" {
  display_name = "Pub/Sub High Latency (${var.env_id})"
  combiner     = "OR"

  conditions {
    display_name = "Oldest Unacked Message > 10m"
    condition_threshold {
      filter = join(" AND ", [
        "resource.type=\"pubsub_subscription\"",
        "metric.type=\"pubsub.googleapis.com/subscription/oldest_unacked_message_age\"",
        "resource.label.subscription_id = one_of(\"${google_pubsub_subscription.ingestion_jobs_sub.name}\", \"${google_pubsub_subscription.notification_events_sub.name}\", \"${google_pubsub_subscription.chime_delivery_sub.name}\")"
      ])

      duration        = "300s" # Condition must persist for 5 minutes
      comparison      = "COMPARISON_GT"
      threshold_value = 600 # 10 minutes (in seconds)

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }
    }
  }

  notification_channels = var.notification_channel_ids

  documentation {
    content = "The oldest message in the queue is > 10 minutes old. Verify that the workers (EventProducer, PushDelivery, EmailWorker) are running and healthy."
  }
}
