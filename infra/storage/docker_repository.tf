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

resource "google_artifact_registry_repository" "docker" {
  project       = var.projects.internal
  location      = var.docker_repository_region
  repository_id = "${var.env_id}-docker-repository"
  description   = "${var.env_id} webcompass docker repository"
  format        = "DOCKER"

  depends_on = [null_resource.docker_auth_setup]
}

resource "null_resource" "docker_auth_setup" {
  triggers = {
    docker_region = var.docker_repository_region
    # Debug to always force the auth.
    # always_run    = "${timestamp()}"
  }
  provisioner "local-exec" {
    command = "gcloud auth configure-docker -q ${self.triggers.docker_region}-docker.pkg.dev"
  }
}

data "google_project" "public" {
  provider = google.public_project
}

# Cross project permission
resource "google_artifact_registry_repository_iam_member" "public_iam_member" {
  provider   = google.internal_project
  repository = google_artifact_registry_repository.docker.name
  location   = google_artifact_registry_repository.docker.location
  role       = "roles/artifactregistry.reader"
  member     = "serviceAccount:service-${data.google_project.public.number}@serverless-robot-prod.iam.gserviceaccount.com"
}
