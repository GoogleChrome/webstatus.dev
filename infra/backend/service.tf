resource "docker_image" "backend" {
  name = "${var.docker_repository_details.url}/backend"
  build {
    context = "${path.cwd}/../backend"
  }
}
resource "docker_registry_image" "backend_remote_image" {
  name          = docker_image.backend.name
  keep_remotely = true
}

data "google_project" "project" {
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
        value = data.google_project.project.number
      }
      env {
        name  = "FIRESTORE_DATABASE"
        value = var.firestore_datails.database
      }
    }
  }
}

# resource "google_cloud_run_service_iam_binding" "public" {
#   count = length(google_cloud_run_v2_service.service)
#   location = google_cloud_run_v2_service.service[count.index].location
#   service  = google_cloud_run_v2_service.service[count.index].name
#   role     = "roles/run.invoker"
#   members = [
#     "allUsers"
#   ]
# }