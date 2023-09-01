data "google_project" "project" {
}

resource "google_storage_bucket" "repo_storage_bucket" {
  name          = "repo-storage-bucket-${data.google_project.project.name}-${var.env_id}"
  location      = "US"
  storage_class = "STANDARD"

  public_access_prevention    = "enforced"
  uniform_bucket_level_access = true
  retention_policy {
    retention_period = 2630000 # 1 month
  }
}

