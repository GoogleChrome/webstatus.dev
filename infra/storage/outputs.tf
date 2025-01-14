# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

output "spanner_info" {
  value = {
    instance   = google_spanner_instance.main.name
    database   = google_spanner_database.database.name
    project_id = google_spanner_instance.main.project
  }
}

output "datastore_info" {
  value = {
    database_name = google_firestore_database.datastore_db.name
    project_id    = google_firestore_database.datastore_db.project
  }
}

# https://cloud.google.com/artifact-registry/docs/docker/store-docker-container-images#add-image
output "docker_repository_details" {
  value = {
    hostname = "${google_artifact_registry_repository.docker.location}-docker.pkg.dev"
    url      = "${google_artifact_registry_repository.docker.location}-docker.pkg.dev/${google_artifact_registry_repository.docker.project}/${google_artifact_registry_repository.docker.name}"
    location = google_artifact_registry_repository.docker.location
    name     = google_artifact_registry_repository.docker.name
  }
}

output "buckets" {
  value = {
    repo_download_bucket = google_storage_bucket.repo_storage_bucket.name
  }
}

output "valkey_env_vars" {
  value = {
    for region, _ in var.region_to_subnet_info_map :
    region => {
      host = google_memorystore_instance.valkey_instance[region].discovery_endpoints[0].address
      port = google_memorystore_instance.valkey_instance[region].discovery_endpoints[0].port
    }
  }
}
