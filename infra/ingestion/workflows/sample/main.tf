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
  account_id   = "sample-workflow-${var.env_id}"
  display_name = "Sample Workflow service account for ${var.env_id}"
}

resource "google_workflows_workflow" "workflow" {
  count           = length(var.regions)
  name            = "${var.env_id}-sample-workflow-${var.regions[count.index]}"
  region          = var.regions[count.index]
  description     = "Sample workflow. Env id: ${var.env_id}"
  service_account = google_service_account.service_account.id
  source_contents = templatefile(
    "${path.root}/../workflows/sample/workflows.yaml.tftpl",
    {
      sample_custom_step_url = var.sample_custom_step_region_to_step_info_map[var.regions[count.index]].url
    }
  )
}

data "google_project" "project" {
}

resource "google_cloud_run_v2_service_iam_member" "sample_step_invoker" {
  count    = length(var.regions)
  project  = data.google_project.project.number
  location = var.regions[count.index]
  name     = var.sample_custom_step_region_to_step_info_map[var.regions[count.index]].name
  role     = "roles/run.invoker"
  member   = google_service_account.service_account.member
}