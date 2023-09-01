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