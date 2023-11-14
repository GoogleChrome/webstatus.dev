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
  service_dir = "frontend"
}

resource "docker_image" "frontend" {
  name = "${var.docker_repository_details.url}/frontend"
  build {
    context = "${path.cwd}/.."
    build_args = {
      service_dir : local.service_dir
    }
    dockerfile = "images/nodejs_service.Dockerfile"
  }
}
resource "docker_registry_image" "frontend_remote_image" {
  name          = docker_image.frontend.name
  keep_remotely = true
}


resource "google_service_account" "frontend" {
  account_id   = "frontend-${var.env_id}"
  provider     = google.public_project
  display_name = "Frontend service account for ${var.env_id}"
}

data "google_project" "host_project" {
}

data "google_project" "datastore_project" {
}



resource "google_cloud_run_v2_service" "service" {
  for_each     = var.region_to_subnet_info_map
  provider     = google.public_project
  launch_stage = "BETA"
  name         = "${var.env_id}-${each.key}-webstatus-frontend"
  location     = each.key
  ingress      = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  template {
    containers {
      image = "${docker_image.frontend.name}@${docker_registry_image.frontend_remote_image.sha256_digest}"
      ports {
        container_port = 5555
      }
      env {
        name  = "BACKEND_API_HOST"
        value = var.backend_api_host
      }
      env {
        name  = "PROJECT_ID"
        value = data.google_project.datastore_project.number
      }
    }
    vpc_access {
      network_interfaces {
        network    = "projects/${data.google_project.host_project.name}/global/networks/${var.vpc_name}"
        subnetwork = "projects/${data.google_project.host_project.name}/regions/${each.key}/subnetworks/${each.value.public}"
      }
      egress = "ALL_TRAFFIC"
    }
    service_account = google_service_account.frontend.email
  }
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

  name                  = "${var.env_id}-frontend-neg-${each.value.location}"
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
  name     = "${var.env_id}-frontend-service"
  dynamic "backend" {
    for_each = google_compute_region_network_endpoint_group.neg
    content {
      group = backend.value.id
    }
  }
  # health_checks = [google_compute_http_health_check.default.id]
}

resource "google_compute_url_map" "url_map" {
  provider = google.public_project
  name     = "${var.env_id}-frontend-url-map"

  default_service = google_compute_backend_service.lb_backend.id
}

resource "google_compute_target_http_proxy" "lb_http_proxy" {
  provider = google.public_project
  name     = "${var.env_id}-frontend-http-proxy"

  url_map = google_compute_url_map.url_map.id
}

# resource "google_compute_global_forwarding_rule" "https" {
#   provider = google.public_project
#   name        = "${var.env_id}-frontend-https-rule"
#   ip_protocol = "TCP"
#   port_range   = "443"
#   target       = google_compute_target_http_proxy.lb_https_proxy.id
# }

# resource "google_compute_target_https_proxy" "lb_https_proxy" {
#   name             = "${var.env_id}-frontend-https-proxy"
#   url_map          = google_compute_url_map.url_map.id
#   ssl_certificates = [google_compute_ssl_certificate.default.id]
# }


# Fake certificate
# resource "google_compute_ssl_certificate" "default" {
#   # The name will contain 8 random hex digits,
#   # e.g. "my-certificate-48ab27cd2a"
#   name        = random_id.certificate.hex
#   private_key = file("path/to/private.key")
#   certificate = file("path/to/certificate.crt")

#   lifecycle {
#     create_before_destroy = true
#   }
# }

# resource "random_id" "certificate" {
#   byte_length = 4
#   prefix      = "frontend-certificate-"

#   # For security, do not expose raw certificate values in the output
#   keepers = {
#     private_key = filebase64sha256("path/to/private.key")
#     certificate = filebase64sha256("path/to/certificate.crt")
#   }
# }
