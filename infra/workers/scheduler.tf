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

locals {
  primary_scheduler_region = tolist(var.regions)[0]
}

# 1. Weekly Job
resource "google_cloud_scheduler_job" "batch_weekly" {
  name        = "batch-refresh-weekly-${var.env_id}"
  description = "Triggers the Weekly Saved Search Refresh Batch"
  schedule    = "0 9 * * 1" # Every Monday at 9:00 AM
  time_zone   = "Etc/UTC"
  region      = local.primary_scheduler_region
  project     = var.internal_project_id
  provider    = google.internal_project

  pubsub_target {
    topic_name = var.pubsub_details.batch_topic_id
    data       = base64encode(file("${path.module}/payloads/batch_weekly.json"))
  }
}

# 2. Monthly Job
resource "google_cloud_scheduler_job" "batch_monthly" {
  name        = "batch-refresh-monthly-${var.env_id}"
  description = "Triggers the Monthly Saved Search Refresh Batch"
  schedule    = "0 9 1 * *" # 1st of every month at 9:00 AM
  time_zone   = "Etc/UTC"
  region      = local.primary_scheduler_region
  project     = var.internal_project_id
  provider    = google.internal_project

  pubsub_target {
    topic_name = var.pubsub_details.batch_topic_id
    data       = base64encode(file("${path.module}/payloads/batch_monthly.json"))
  }
}

# 3. Immediate Job (Every 15 minutes)
resource "google_cloud_scheduler_job" "batch_immediate" {
  name        = "batch-refresh-immediate-${var.env_id}"
  description = "Triggers the Immediate Saved Search Refresh Batch (Every 15m)"
  schedule    = "*/15 * * * *"
  time_zone   = "Etc/UTC"
  region      = local.primary_scheduler_region
  project     = var.internal_project_id
  provider    = google.internal_project

  pubsub_target {
    topic_name = var.pubsub_details.batch_topic_id
    data       = base64encode(file("${path.module}/payloads/batch_immediate.json"))
  }
}
