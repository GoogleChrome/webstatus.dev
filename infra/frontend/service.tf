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
    target     = var.docker_build_target
    dockerfile = "${path.cwd}/../images/nodejs_service.Dockerfile"
    # Use buildx default builder instead of legacy builder
    # Needed for the --mount args
    # Must also specify platform too.
    builder        = "default"
    platform       = "linux/amd64"
    build_log_file = "${path.cwd}/frontend.log"
  }
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
}

resource "docker_registry_image" "frontend_remote_image" {
  name          = docker_image.frontend.name
  keep_remotely = true
  triggers = {
    dir_sha1 = sha1(join("", [for f in fileset(path.cwd, "/../${local.service_dir}/**") : filesha1(f)], [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]))
  }
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
  for_each = var.region_to_subnet_info_map
  provider = google.public_project
  name     = "${var.env_id}-${each.key}-webstatus-frontend"
  location = each.key
  ingress  = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  template {
    scaling {
      min_instance_count = var.min_instance_count
      max_instance_count = var.max_instance_count
    }
    containers {
      image = "${docker_image.frontend.name}@${docker_registry_image.frontend_remote_image.sha256_digest}"
      ports {
        container_port = 5555
      }
      env {
        name  = "API_URL"
        value = var.backend_api_host
      }
      env {
        name  = "GOOGLE_ANALYTICS_ID"
        value = var.google_analytics_id
      }
      env {
        name  = "PROJECT_ID"
        value = data.google_project.datastore_project.number
      }
      env {
        name  = "FIREBASE_APP_AUTH_DOMAIN"
        value = var.firebase_settings.auth_domain
      }
      env {
        name  = "FIREBASE_APP_API_KEY"
        value = var.firebase_settings.api_key
      }
      env {
        name  = "FIREBASE_AUTH_TENANT_ID"
        value = var.firebase_settings.tenant_id
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

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }

  deletion_protection = var.deletion_protection
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
  iap {
    enabled = false
  }
}

resource "google_compute_url_map" "url_map" {
  provider = google.public_project
  name     = "${var.env_id}-frontend-url-map"

  host_rule {
    hosts        = var.domains
    path_matcher = "allpaths"
  }

  path_matcher {
    name            = "allpaths"
    default_service = google_compute_backend_service.lb_backend.id
  }
  default_url_redirect {
    host_redirect  = "github.com"
    https_redirect = true
    path_redirect  = "/GoogleChrome/webstatus.dev"
    strip_query    = true
  }
}

resource "google_compute_global_forwarding_rule" "https" {
  provider    = google.public_project
  name        = "${var.env_id}-frontend-https-rule"
  ip_protocol = "TCP"
  port_range  = "443"
  ip_address  = google_compute_global_address.ub_ip_address.id
  target      = google_compute_target_https_proxy.lb_https_proxy.id
}

resource "google_compute_global_address" "ub_ip_address" {
  provider = google.public_project
  name     = "${var.env_id}-frontend-ip"
}

resource "google_compute_global_forwarding_rule" "main" {
  provider    = google.public_project
  name        = "${var.env_id}-frontend-main-https-rule"
  ip_protocol = "TCP"
  port_range  = "443"
  ip_address  = google_compute_global_address.main_ip_address.id
  target      = google_compute_target_https_proxy.lb_https_proxy.id
}

resource "google_compute_global_address" "main_ip_address" {
  provider = google.public_project
  name     = "${var.env_id}-frontend-main-ip"
}

resource "google_compute_target_https_proxy" "lb_https_proxy" {
  provider         = google.public_project
  name             = "${var.env_id}-frontend-https-proxy"
  url_map          = google_compute_url_map.url_map.id
  ssl_certificates = length(google_compute_managed_ssl_certificate.lb_default) > 0 ? [google_compute_managed_ssl_certificate.lb_default[0].id] : var.custom_ssl_certificates
  depends_on = [
    google_compute_managed_ssl_certificate.lb_default
  ]
}

resource "google_compute_managed_ssl_certificate" "lb_default" {
  provider = google.public_project
  name     = "${var.env_id}-frontend-ssl-cert"
  count    = length(var.custom_ssl_certificates) > 0 ? 0 : 1
  project  = var.projects.public
  managed {
    domains = var.domains
  }
}
