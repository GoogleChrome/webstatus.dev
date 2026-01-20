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
    dockerfile = "${path.cwd}/../images/go_service.Dockerfile"
    # Use buildx default builder instead of legacy builder
    # Needed for the --mount args
    # Must also specify platform too.
    builder        = "default"
    platform       = "linux/amd64"
    build_log_file = "${path.cwd}/backend.log"
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

module "otel_sidecar" {
  source = "../modules/otel"
  providers = {
    google.project = google.public_project
  }
  docker_repository_details = var.docker_repository_details
  service_account           = google_service_account.backend.member
}

resource "google_cloud_run_v2_service" "service" {
  for_each = var.region_to_subnet_info_map
  provider = google.public_project
  name     = "${var.env_id}-${each.key}-webstatus-backend"
  location = each.key
  ingress  = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  template {
    scaling {
      min_instance_count = var.min_instance_count
      max_instance_count = var.max_instance_count
    }
    containers {
      name  = "backend"
      image = "${docker_image.backend.name}@${docker_registry_image.backend_remote_image.sha256_digest}"
      ports {
        container_port = 8080
      }
      depends_on = ["otel"]
      startup_probe {
        initial_delay_seconds = 0
        timeout_seconds       = 1
        period_seconds        = 3
        failure_threshold     = 10
        tcp_socket {
          port = 8080
        }
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
        name  = "FIREBASE_AUTH_TENANT_ID"
        value = var.firebase_settings.tenant_id
      }
      env {
        name  = "DATASTORE_DATABASE"
        value = var.datastore_info.database_name
      }
      env {
        name  = "CORS_ALLOWED_ORIGIN"
        value = var.cors_allowed_origin
      }
      env {
        name  = "VALKEYHOST"
        value = var.valkey_env_vars[each.key].host
      }
      env {
        name  = "VALKEYPORT"
        value = var.valkey_env_vars[each.key].port
      }
      env {
        name  = "CACHE_TTL"
        value = var.cache_settings.default_duration
      }
      env {
        name  = "AGGREGATED_FEATURE_STATS_TTL"
        value = var.cache_settings.aggregated_feature_stats_duration
      }
      env {
        name  = "OTEL_EXPORTER_OTLP_ENDPOINT"
        value = "http://localhost:4318"
      }
      env {
        name  = "OTEL_SERVICE_NAME"
        value = "backend"
      }
      env {
        name  = "OTEL_GCP_PROJECT_ID"
        value = var.projects.public
      }
      env {
        name  = "BASE_URL"
        value = var.backend_api_url
      }
      env {
        name  = "INGESTION_TOPIC_ID"
        value = var.ingestion_topic_id
      }
      env {
        name  = "PUBSUB_PROJECT_ID"
        value = var.pubsub_project_id
      }
    }
    containers {
      name  = "otel"
      image = module.otel_sidecar.otel_image
      liveness_probe {
        http_get {
          port = 4319
          path = "/"
        }
        initial_delay_seconds = 3
        period_seconds        = 10
        failure_threshold     = 10
        timeout_seconds       = 10
      }
      startup_probe {
        initial_delay_seconds = 0
        timeout_seconds       = 1
        period_seconds        = 3
        failure_threshold     = 10
        tcp_socket {
          port = 4319
        }
      }
    }
    vpc_access {
      network_interfaces {
        network    = "projects/${data.google_project.host_project.name}/global/networks/${var.vpc_name}"
        subnetwork = "projects/${data.google_project.host_project.name}/regions/${each.key}/subnetworks/${each.value.public}"
      }
      egress = "PRIVATE_RANGES_ONLY"
    }
    service_account = google_service_account.backend.email
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }

  depends_on = [
    google_project_iam_member.gcp_datastore_user,
  ]

  deletion_protection = var.deletion_protection
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
  role     = "roles/spanner.databaseUser"
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

resource "google_pubsub_topic_iam_member" "pub" {
  topic    = var.ingestion_topic_id
  role     = "roles/pubsub.publisher"
  member   = "serviceAccount:${google_service_account.backend.email}"
  provider = google.internal_project
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
  iap {
    enabled = false
  }
}

resource "google_compute_url_map" "url_map" {
  provider = google.public_project
  name     = "${var.env_id}-backend-url-map"

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

resource "google_compute_global_forwarding_rule" "main" {
  provider    = google.public_project
  name        = "${var.env_id}-backend-main-https-rule"
  ip_protocol = "TCP"
  port_range  = "443"
  ip_address  = google_compute_global_address.main_ip_address.id
  target      = google_compute_target_https_proxy.lb_https_proxy.id
}

resource "google_compute_global_address" "main_ip_address" {
  provider = google.public_project
  name     = "${var.env_id}-backend-main-ip"
}

resource "google_compute_target_https_proxy" "lb_https_proxy" {
  provider         = google.public_project
  name             = "${var.env_id}-backend-https-proxy"
  url_map          = google_compute_url_map.url_map.id
  ssl_certificates = length(google_compute_managed_ssl_certificate.lb_default) > 0 ? [google_compute_managed_ssl_certificate.lb_default[0].id] : var.custom_ssl_certificates
  depends_on = [
    google_compute_managed_ssl_certificate.lb_default
  ]
}

resource "google_compute_managed_ssl_certificate" "lb_default" {
  provider = google.public_project
  name     = "${var.env_id}-backend-ssl-cert"
  count    = length(var.custom_ssl_certificates) > 0 ? 0 : 1
  project  = var.projects.public
  managed {
    domains = var.domains
  }
}
