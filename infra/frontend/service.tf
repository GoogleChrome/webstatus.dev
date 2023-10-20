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

# resource "google_cloud_run_service_iam_member" "public" {
#   count = length(google_cloud_run_v2_service.service)
#   location = google_cloud_run_v2_service.service[count.index].location
#   service  = google_cloud_run_v2_service.service[count.index].name
#   role     = "roles/run.invoker"
#   members = [
#     "allUsers"
#   ]
# }