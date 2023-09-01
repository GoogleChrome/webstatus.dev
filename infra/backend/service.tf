locals {
  service_dir = "backend"
}

resource "docker_image" "backend" {
  name = "${var.docker_repository_details.url}/backend"
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
resource "docker_registry_image" "backend_remote_image" {
  name          = docker_image.backend.name
  keep_remotely = true
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
}

resource "google_cloud_run_v2_service" "service" {
  count    = length(var.regions)
  name     = "${var.env_id}-${var.regions[count.index]}-webstatus-backend"
  location = var.regions[count.index]

  template {
    containers {
      image = "${docker_image.backend.name}@${docker_registry_image.backend_remote_image.sha256_digest}"
      ports {
        container_port = 8080
      }
      env {
        name  = "SPANNER_DATABASE"
        value = var.spanner_datails.database
      }
      env {
        name  = "SPANNER_INSTANCE"
        value = var.spanner_datails.instance
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
    service_account = google_service_account.backend.email
  }
}

resource "google_service_account" "backend" {
  account_id   = "backend-${var.env_id}"
  display_name = "Backend service account for ${var.env_id}"
}

resource "google_project_iam_member" "gcp_firestore_user" {
  role    = "roles/datastore.user"
  project = var.firestore_info.project_id
  member  = google_service_account.backend.member
}

# resource "google_cloud_run_service_iam_member" "public" {
#   count = length(google_cloud_run_v2_service.service)
#   location = google_cloud_run_v2_service.service[count.index].location
#   service  = google_cloud_run_v2_service.service[count.index].name
#   role     = "roles/run.invoker"
#   members = [
#     "allUsers"
#   ]
# }