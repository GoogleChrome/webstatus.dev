# Copyright 2024 Google LLC
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


resource "docker_image" "otel_collector" {
  name = "${var.docker_repository_details.url}/otel_collector"
  build {
    context    = "${path.cwd}/../otel"
    dockerfile = "Dockerfile"
  }
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../otel/**") : filesha1(f)]))
  }
}
resource "docker_registry_image" "otel_collector_remote_image" {
  name          = docker_image.otel_collector.name
  keep_remotely = true
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../otel/**") : filesha1(f)]))
  }
}

output "otel_image" {
  value = "${docker_image.otel_collector.name}@${docker_registry_image.otel_collector_remote_image.sha256_digest}"
}


data "google_project" "project" {
  provider = google.project
}

resource "google_project_iam_member" "gcp_metric_permission" {
  role     = "roles/monitoring.metricWriter"
  provider = google.project
  project  = data.google_project.project.id
  member   = var.service_account
}

resource "google_project_iam_member" "gcp_log_permission" {
  role     = "roles/logging.logWriter"
  provider = google.project
  project  = data.google_project.project.id
  member   = var.service_account
}

resource "google_project_iam_member" "gcp_trace_permission" {
  role     = "roles/cloudtrace.agent"
  provider = google.project
  project  = data.google_project.project.id
  member   = var.service_account
}

