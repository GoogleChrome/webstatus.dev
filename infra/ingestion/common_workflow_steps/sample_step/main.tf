resource "docker_image" "sample_service_image" {
  name = "${var.docker_repository_details.url}/sample_service_image"
  build {
    context = "${path.cwd}/../workflows/common_custom_steps/sample_service"
  }
}
resource "docker_registry_image" "sample_service_remote_image" {
  name          = docker_image.sample_service_image.name
  keep_remotely = true
}

resource "google_cloud_run_v2_service" "service" {
  count    = length(var.regions)
  name     = "${var.env_id}-${var.regions[count.index]}-sample-srv"
  location = var.regions[count.index]

  template {
    containers {
      image = "${docker_image.sample_service_image.name}@${docker_registry_image.sample_service_remote_image.sha256_digest}"
    }
  }
}