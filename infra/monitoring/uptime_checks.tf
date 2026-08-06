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
# 1. Global Multi-Region Synthetic Uptime Check
# ==========================================
resource "google_monitoring_uptime_check_config" "webstatus_uptime_check" {
  provider     = google.internal_project
  display_name = "WebStatus Frontend Uptime (${var.env_id})"
  timeout      = "10s"
  period       = "60s"

  http_check {
    path           = "/"
    port           = 443
    use_ssl        = true
    validate_ssl   = true
    request_method = "GET"
  }

  monitored_resource {
    type = "uptime_url"
    labels = {
      project_id = var.project_id
      host       = var.target_host
    }
  }

  selected_regions = [
    "EUROPE",
    "USA_VIRGINIA",
    "USA_OREGON",
    "USA_IOWA",
    "ASIA_PACIFIC",
    "SOUTH_AMERICA"
  ]
}

# ==========================================
# 2. Critical: WebStatus Outage Alert Policy
# ==========================================
resource "google_monitoring_alert_policy" "webstatus_outage_alert" {
  provider     = google.internal_project
  display_name = "WebStatus Service Outage (${var.env_id})"
  combiner     = "OR"

  conditions {
    display_name = "Synthetic Uptime Check Failed"
    condition_threshold {
      filter = join(" AND ", [
        "resource.type=\"uptime_url\"",
        "metric.type=\"monitoring.googleapis.com/uptime_check/check_passed\"",
        "metric.label.check_id=\"${google_monitoring_uptime_check_config.webstatus_uptime_check.uptime_check_id}\""
      ])

      duration        = "120s"
      comparison      = "COMPARISON_GT"
      threshold_value = 1

      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_NEXT_OLDER"
        cross_series_reducer = "REDUCE_COUNT_FALSE"
        group_by_fields      = ["resource.label.*"]
      }
    }
  }

  notification_channels = var.notification_channel_ids

  documentation {
    content   = "WebStatus (${var.env_id}) uptime probe failed from multiple global monitoring regions. Check Cloud Run frontend/backend status and logs."
    mime_type = "text/markdown"
  }
}
