# Copyright 2023 Google LLC
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

resource "google_service_account" "service_account" {
  account_id   = "wpt-${var.env_id}"
  provider     = google.internal_project
  display_name = "WPT Workflow service account for ${var.env_id}"
}

resource "google_workflows_workflow" "workflow" {
  count           = length(var.regions)
  provider        = google.internal_project
  name            = "${var.env_id}-wpt-${var.regions[count.index]}"
  region          = var.regions[count.index]
  description     = "WPT Workflow. Env id: ${var.env_id}"
  service_account = google_service_account.service_account.id
  source_contents = templatefile(
    "${path.root}/../workflows/wpt/workflows.yaml.tftpl",
    {
      project_id   = google_cloud_run_v2_job.wpt[count.index].project
      job_name     = google_cloud_run_v2_job.wpt[count.index].name
      job_location = google_cloud_run_v2_job.wpt[count.index].location
      timeout      = var.timeout_seconds
    }
  )
}

resource "google_cloud_scheduler_job" "workflow_trigger_job" {
  count    = length(var.regions)
  provider = google.internal_project
  name     = "${var.env_id}-wpt-trigger-job-${var.regions[count.index]}"
  region   = var.regions[count.index]
  schedule = var.region_schedules[var.regions[count.index]]
  http_target {
    uri         = "https://workflowexecutions.googleapis.com/v1/${google_workflows_workflow.workflow[count.index].id}/executions"
    http_method = "POST"
    oauth_token {
      service_account_email = google_service_account.service_account.email
    }
  }
}

resource "google_project_iam_member" "invoker" {
  provider = google.internal_project
  role     = "roles/workflows.invoker"
  project  = var.spanner_datails.project_id
  member   = google_service_account.service_account.member
}
