resource "docker_image" "frontend" {
  name = "${var.docker_repository_details.url}/frontend"
  build {
    context = "${path.cwd}/../frontend"
  }
}
resource "docker_registry_image" "frontend_remote_image" {
  name          = docker_image.frontend.name
  keep_remotely = true
}

data "google_project" "project" {
}


resource "google_cloud_run_v2_service" "service" {
  count    = length(var.regions)
  name     = "${var.env_id}-${var.regions[count.index]}-webstatus-frontend"
  location = var.regions[count.index]

  template {
    containers {
      image = "${docker_image.frontend.name}@${docker_registry_image.frontend_remote_image.sha256_digest}"
      ports {
        container_port = 8000
      }
      env {
        name  = "BACKEND_API_HOST"
        value = var.backend_api_host
      }
      env {
        name  = "PROJECT_ID"
        value = data.google_project.project.number
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