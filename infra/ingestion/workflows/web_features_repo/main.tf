resource "google_service_account" "service_account" {
  account_id   = "web-features-repo-${var.env_id}"
  display_name = "Web Features Repo service account for ${var.env_id}"
}

resource "google_workflows_workflow" "workflow" {
  count           = length(var.regions)
  name            = "${var.env_id}-web-features-repo-${var.regions[count.index]}"
  region          = var.regions[count.index]
  description     = "Web Feature Repo Workflow. Env id: ${var.env_id}"
  service_account = google_service_account.service_account.id
  source_contents = templatefile(
    "${path.root}/../workflows/web-features-repo/workflows.yaml.tftpl",
    {
      web_feature_consume_step_url = google_cloud_run_v2_service.web_feature_service[count.index].uri
      repo_downloader_step_url     = var.repo_downloader_step_region_to_step_info_map[var.regions[count.index]].url
    }
  )
}

data "google_project" "project" {
}

resource "google_cloud_run_v2_service_iam_member" "repo_downloader_step_invoker" {
  count    = length(var.regions)
  project  = data.google_project.project.number
  location = var.regions[count.index]
  name     = var.repo_downloader_step_region_to_step_info_map[var.regions[count.index]].name
  role     = "roles/run.invoker"
  member   = google_service_account.service_account.member
}