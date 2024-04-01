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

locals {
  service_dir = "backend"
}

resource "docker_image" "backend" {
  name = "${var.docker_repository_details.url}/backend"
  build {
    context = "${path.cwd}/.."
    build_args = {
      service_dir : local.service_dir
    }
    dockerfile = "images/go_service.Dockerfile"
  }
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
}
resource "docker_registry_image" "backend_remote_image" {
  name          = docker_image.backend.name
  keep_remotely = true
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
}

data "google_project" "host_project" {
}


resource "google_cloud_run_v2_service" "service" {
  for_each     = var.region_to_subnet_info_map
  provider     = google.public_project
  launch_stage = "BETA"
  name         = "${var.env_id}-${each.key}-webstatus-backend"
  location     = each.key
  ingress      = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  template {
    containers {
      image = "${docker_image.backend.name}@${docker_registry_image.backend_remote_image.sha256_digest}"
      ports {
        container_port = 8080
      }
      env {
        name  = "SPANNER_DATABASE"
        value = var.spanner_datails.database
      }
      env {
        name  = "SPANNER_INSTANCE"
        value = var.spanner_datails.instance
      }
      env {
        name  = "PROJECT_ID"
        value = var.datastore_info.project_id
      }
      env {
        name  = "DATASTORE_DATABASE"
        value = var.datastore_info.database_name
      }
      env {
        name  = "CORS_ALLOWED_ORIGIN"
        value = "https://website-webstatus-dev.corp.goog"
      }
    }
    # vpc_access {
    #   network_interfaces {
    #     network    = "projects/${data.google_project.host_project.name}/global/networks/${var.vpc_name}"
    #     subnetwork = "projects/${data.google_project.host_project.name}/regions/${each.key}/subnetworks/${each.value.public}"
    #   }
    #   egress = "ALL_TRAFFIC"
    # }
    service_account = google_service_account.backend.email
  }
  depends_on = [
    google_project_iam_member.gcp_datastore_user,
  ]
}

resource "google_service_account" "backend" {
  account_id   = "backend-${var.env_id}"
  provider     = google.public_project
  display_name = "Backend service account for ${var.env_id}"
}

resource "google_project_iam_member" "gcp_datastore_user" {
  role     = "roles/datastore.user"
  provider = google.internal_project
  project  = var.datastore_info.project_id
  member   = google_service_account.backend.member
}

resource "google_project_iam_member" "gcp_spanner_user" {
  role     = "roles/spanner.databaseReader"
  provider = google.internal_project
  project  = var.datastore_info.project_id
  member   = google_service_account.backend.member
}

resource "google_cloud_run_service_iam_member" "public" {
  provider = google.public_project
  for_each = google_cloud_run_v2_service.service
  location = each.value.location
  service  = each.value.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_compute_region_network_endpoint_group" "neg" {
  provider = google.public_project
  for_each = google_cloud_run_v2_service.service

  name                  = "${var.env_id}-backend-neg-${each.value.location}"
  network_endpoint_type = "SERVERLESS"
  region                = each.value.location

  cloud_run {
    service = each.value.name
  }
  depends_on = [
    google_cloud_run_v2_service.service
  ]
}

resource "google_compute_backend_service" "lb_backend" {
  provider = google.public_project
  name     = "${var.env_id}-backend-service"
  dynamic "backend" {
    for_each = google_compute_region_network_endpoint_group.neg
    content {
      group = backend.value.id
    }
  }
}

resource "google_compute_url_map" "url_map" {
  provider = google.public_project
  name     = "${var.env_id}-backend-url-map"

  default_service = google_compute_backend_service.lb_backend.id
}

resource "google_compute_global_forwarding_rule" "https" {
  provider    = google.public_project
  name        = "${var.env_id}-backend-https-rule"
  ip_protocol = "TCP"
  port_range  = "443"
  ip_address  = google_compute_global_address.ub_ip_address.id
  target      = google_compute_target_https_proxy.lb_https_proxy.id
}

resource "google_compute_global_address" "ub_ip_address" {
  provider = google.public_project
  name     = "${var.env_id}-backend-ip"
}

resource "google_compute_target_https_proxy" "lb_https_proxy" {
  provider = google.public_project
  name     = "${var.env_id}-backend-https-proxy"
  url_map  = google_compute_url_map.url_map.id
  ssl_certificates = [
    "ub-self-sign" # Temporary for UB
  ]
}
