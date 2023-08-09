output "spanner_details" {
  value = {
    instance = google_spanner_instance.main.name
    database = google_spanner_database.database.name
  }
}

output "firestore_details" {
  value = {
    database = "placeholder"
  }
}

# https://cloud.google.com/artifact-registry/docs/docker/store-docker-container-images#add-image
output "docker_repository_details" {
  value = {
    hostname = "${google_artifact_registry_repository.docker.location}-docker.pkg.dev"
    url      = "${google_artifact_registry_repository.docker.location}-docker.pkg.dev/${google_artifact_registry_repository.docker.project}/${google_artifact_registry_repository.docker.name}"
  }
}
