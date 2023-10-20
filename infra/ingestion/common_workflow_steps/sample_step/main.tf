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