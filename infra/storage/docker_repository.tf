resource "google_artifact_registry_repository" "docker" {
  location      = var.docker_repository_region
  repository_id = "${var.env_id}-docker-repository"
  description   = "${var.env_id} webcompass docker repository"
  format        = "DOCKER"
}