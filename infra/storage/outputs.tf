output "spanner_info" {
  value = {
    instance = google_spanner_instance.main.name
    database = google_spanner_database.database.name
  }
}

output "firestore_info" {
  value = {
    database_name = google_firestore_database.firestore_db.name
    project_id    = google_firestore_database.firestore_db.project
  }
}

# https://cloud.google.com/artifact-registry/docs/docker/store-docker-container-images#add-image
output "docker_repository_details" {
  value = {
    hostname = "${google_artifact_registry_repository.docker.location}-docker.pkg.dev"
    url      = "${google_artifact_registry_repository.docker.location}-docker.pkg.dev/${google_artifact_registry_repository.docker.project}/${google_artifact_registry_repository.docker.name}"
  }
}

output "buckets" {
  value = {
    repo_download_bucket = google_storage_bucket.repo_storage_bucket.name
  }
}
