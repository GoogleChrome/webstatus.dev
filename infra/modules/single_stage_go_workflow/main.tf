# Copyright 2024 Google LLC
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

module "image" {
  source                = "../go_image"
  go_module_path        = var.go_module_path
  image_name            = var.image_name
  binary_type           = "job"
  docker_repository_url = var.docker_repository_url
}

module "job" {
  source = "../job"
  providers = {
    google.internal_project = google.internal_project
    google.public_project   = google.public_project
  }
  env_id                          = var.env_id
  regions                         = var.regions
  env_vars                        = var.env_vars
  image                           = module.image.remote_image
  deletion_protection             = var.deletion_protection
  full_name                       = var.full_name
  short_name                      = var.short_name
  timeout_seconds                 = var.timeout_seconds
  spanner_project_id              = var.spanner_details.project_id
  does_process_write_to_datastore = var.does_process_write_to_datastore
  does_process_write_to_spanner   = var.does_process_write_to_spanner
  resource_limits                 = var.resource_job_limits
}

resource "google_service_account" "service_account" {
  account_id   = "${var.short_name}-${var.env_id}"
  provider     = google.internal_project
  display_name = "${var.full_name} service account for ${var.env_id}"
}

resource "google_cloud_run_v2_job_iam_member" "job_invoker" {
  for_each = module.job.regional_job_map
  provider = google.internal_project
  location = each.key
  name     = each.value.name
  role     = "roles/run.invoker"
  member   = google_service_account.service_account.member
}

resource "google_cloud_run_v2_job_iam_member" "job_status" {
  for_each = module.job.regional_job_map
  provider = google.internal_project
  location = each.key
  name     = each.value.name
  role     = "roles/run.viewer"
  member   = google_service_account.service_account.member
}

resource "google_workflows_workflow" "workflow" {
  for_each        = module.job.regional_job_map
  provider        = google.internal_project
  name            = "${var.env_id}-${var.short_name}-${each.key}"
  region          = each.key
  description     = "${var.full_name}. Env id: ${var.env_id}"
  service_account = google_service_account.service_account.id
  source_contents = templatefile(
    "${path.root}/modules/single_stage_go_workflow/workflows.yaml.tftpl",
    {
      timeout      = var.timeout_seconds
      project_id   = each.value.project_id
      job_name     = each.value.name
      job_location = each.key
    }
  )
}

resource "google_cloud_scheduler_job" "workflow_trigger_job" {
  for_each = google_workflows_workflow.workflow
  provider = google.internal_project
  name     = "${var.env_id}-${var.short_name}-trigger-job-${each.value.region}"
  region   = each.value.region
  schedule = var.region_schedules[each.value.region]
  http_target {
    uri         = "https://workflowexecutions.googleapis.com/v1/${each.value.id}/executions"
    http_method = "POST"
    oauth_token {
      service_account_email = google_service_account.service_account.email
    }
  }
}

resource "google_project_iam_member" "invoker" {
  provider = google.internal_project
  role     = "roles/workflows.invoker"
  project  = var.project_id
  member   = google_service_account.service_account.member
}
