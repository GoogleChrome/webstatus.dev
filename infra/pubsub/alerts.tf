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
        "resource.label.subscription_id = one_of(\"${google_pubsub_subscription.ingestion_dlq_sub.name}\", \"${google_pubsub_subscription.notification_dlq_sub.name}\", \"${google_pubsub_subscription.delivery_dlq_sub.name}\", \"${google_pubsub_subscription.code_subscriptions_dlq_sub.name}\")"
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
        "resource.label.subscription_id = one_of(\"${google_pubsub_subscription.ingestion_jobs_sub.name}\", \"${google_pubsub_subscription.notification_events_sub.name}\", \"${google_pubsub_subscription.chime_delivery_sub.name}\", \"${google_pubsub_subscription.vcs_scan_tasks_sub.name}\", \"${google_pubsub_subscription.github_issue_delivery_sub.name}\")"
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
    content = "The oldest message in the queue is > 10 minutes old. Verify that the workers (EventProducer, PushDelivery, EmailWorker, VCS Scanner, GitHub Issue Delivery) are running and healthy."
  }
}

# ==========================================
# 3. Error: Code Scans Failing
# ==========================================
# vcs-scanner acknowledges failed scans instead of retrying them through Pub/Sub (vcs-sync
# republishes repositories whose latest scan failed), so a broken scanner no longer backs up the
# queue. This alert pages on the failures themselves: 3+ repositories failing for a reason that is
# expected to be temporary (timeouts, stalls, 5XX, panics...) for 15 minutes.
# Permanent failures (blocked or deleted repositories) and shutdown cancellations are excluded.

resource "google_logging_metric" "vcs_scan_failures" {
  project     = var.project_id
  name        = "vcs_scan_failures_${var.env_id}"
  description = "Failed repository scans in the vcs-scanner worker, by error class and repository."
  filter = join(" AND ", [
    "resource.type=\"cloud_run_worker_pool\"",
    "resource.labels.worker_pool_name:\"vcs-scanner-${var.env_id}-\"",
    "jsonPayload.message=\"repository scan failed\"",
  ])

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    labels {
      key        = "error_class"
      value_type = "STRING"
    }
    labels {
      key        = "repo"
      value_type = "STRING"
    }
  }

  label_extractors = {
    "error_class" = "EXTRACT(jsonPayload.error_class)"
    "repo"        = "EXTRACT(jsonPayload.repo)"
  }
}

resource "google_monitoring_alert_policy" "vcs_scan_failures" {
  display_name = "Code Scans Failing (${var.env_id})"
  combiner     = "OR"
  severity     = "ERROR"

  conditions {
    display_name = "3+ repositories failing to scan (non-permanent errors)"
    condition_prometheus_query_language {
      query               = <<-EOT
        count(
          count by (repo) (
            increase(logging_googleapis_com:user_${google_logging_metric.vcs_scan_failures.name}{error_class!~"PERMANENT_CLIENT|REDIRECT_NOT_FOLLOWED|INVALID_PROVIDER|CANCELED"}[30m]) > 0
          )
        ) >= 3
      EOT
      duration            = "900s" # Must persist for 15 minutes; short GitHub blips self-heal.
      evaluation_interval = "60s"
    }
  }

  notification_channels = var.notification_channel_ids

  documentation {
    content   = "Three or more repositories have been failing to scan for a non-permanent reason (see the error_class label: TIMEOUT, STALL, STREAM, TRANSIENT_SERVER, RATE_LIMITED, SPANNER, PANIC, UNKNOWN). Check the vcs-scanner logs for \"repository scan failed\" entries. Failed repositories are retried automatically by vcs-sync with backoff."
    mime_type = "text/markdown"
  }
}
