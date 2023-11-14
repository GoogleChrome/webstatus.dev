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

resource "google_compute_network" "shared_vpc" {
  name                            = "${var.env_id}-webstatus-dev-network"
  project                         = var.host_project_id
  auto_create_subnetworks         = false
  routing_mode                    = "GLOBAL"
  delete_default_routes_on_create = true
}

resource "null_resource" "vpc_extra_cleanup" {
  triggers = {
    vpc_name   = google_compute_network.shared_vpc.name
    project_id = var.host_project_id
  }
  provisioner "local-exec" {
    when    = destroy
    command = "${path.module}/clean_up.sh ${self.triggers.project_id} ${self.triggers.vpc_name}"
  }
}


resource "google_compute_subnetwork" "subnetwork_internal" {
  for_each = var.region_to_subnet_map
  name     = "${var.env_id}-webstatus-dev-internal-subnet"

  ip_cidr_range = each.value.internal.cidr
  region        = each.key

  stack_type = "IPV4_ONLY"

  network = google_compute_network.shared_vpc.id
}

resource "google_compute_subnetwork" "subnetwork_public" {
  for_each = var.region_to_subnet_map
  name     = "${var.env_id}-webstatus-dev-public-subnet"

  ip_cidr_range = each.value.public.cidr
  region        = each.key

  stack_type = "IPV4_ONLY"

  network = google_compute_network.shared_vpc.id
}

data "google_project" "public" {
  provider = google.public_project
}

data "google_project" "internal" {
  provider = google.internal_project
}

resource "google_project_iam_member" "public_network_user" {
  project = var.host_project_id
  role    = "roles/compute.networkUser"
  member  = "serviceAccount:service-${data.google_project.public.number}@serverless-robot-prod.iam.gserviceaccount.com"
}


resource "google_project_iam_member" "internal_network_user" {
  project = var.host_project_id
  role    = "roles/compute.networkUser"
  member  = "serviceAccount:service-${data.google_project.internal.number}@serverless-robot-prod.iam.gserviceaccount.com"
}