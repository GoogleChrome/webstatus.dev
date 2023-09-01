locals {
  service_dir = "workflows/steps/services/web_feature_consumer"
}
resource "docker_image" "web_feature_consumer_image" {
  name = "${var.docker_repository_details.url}/web_feature_consumer_image"
  build {
    context = "${path.cwd}/.."
    build_args = {
      service_dir : local.service_dir
    }
    dockerfile = "images/go_service.Dockerfile"
  }
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
}
resource "docker_registry_image" "web_feature_consumer_remote_image" {
  name          = docker_image.web_feature_consumer_image.name
  keep_remotely = true
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
}

resource "google_storage_bucket_iam_member" "web_feature_consumer_binding" {
  bucket = var.repo_bucket
  role   = "roles/storage.objectViewer"
  member = google_service_account.web_feature_consumer_service_account.member
}

resource "google_service_account" "web_feature_consumer_service_account" {
  account_id   = "web-feature-consumer-${var.env_id}"
  display_name = "Web Feature Consumer service account for ${var.env_id}"
}

resource "google_project_iam_member" "gcp_firestore_user" {
  role    = "roles/datastore.user"
  project = var.firestore_info.project_id
  member  = google_service_account.web_feature_consumer_service_account.member

}

resource "google_cloud_run_v2_service" "web_feature_service" {
  count    = length(var.regions)
  name     = "${var.env_id}-${var.regions[count.index]}-web-feature-consumer-srv"
  location = var.regions[count.index]

  template {
    containers {
      image = "${docker_image.web_feature_consumer_image.name}@${docker_registry_image.web_feature_consumer_remote_image.sha256_digest}"
      env {
        name  = "BUCKET"
        value = var.repo_bucket
      }
      env {
        name  = "PROJECT_ID"
        value = var.firestore_info.project_id
      }
      env {
        name  = "DATASTORE_DATABASE"
        value = var.firestore_info.database_name
      }
    }
    service_account = google_service_account.web_feature_consumer_service_account.email
  }
  depends_on = [
    google_storage_bucket_iam_member.web_feature_consumer_binding,
  ]
}

resource "google_cloud_run_v2_service_iam_binding" "web_feature_step_invoker" {
  count    = length(var.regions)
  project  = data.google_project.project.number
  location = var.regions[count.index]
  name     = google_cloud_run_v2_service.web_feature_service[count.index].name
  # name     = var.repo_downloader_step_region_to_step_info_map[var.regions[count.index]].name
  role = "roles/run.invoker"
  members = [
    google_service_account.service_account.member,
  ]
}