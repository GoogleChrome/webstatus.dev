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

# https://cloud.google.com/vpc/docs/about-service-connection-policies#shared-vpc
# Must be created on the host project.
resource "google_network_connectivity_service_connection_policy" "valkey_policy" {
  project       = var.projects.host
  for_each      = var.region_to_subnet_info_map
  name          = "${var.env_id}-${each.key}-valkey-policy"
  location      = each.key
  service_class = "gcp-memorystore"
  description   = "${var.env_id} service connection policy for ${each.key}"
  network       = var.vpc_id
  psc_config {
    subnetworks = [each.value.internal_id]
  }
}

resource "google_memorystore_instance" "valkey_instance" {
  project     = var.projects.internal
  for_each    = var.region_to_subnet_info_map
  instance_id = "${var.env_id}-valkey-${each.key}"
  shard_count = 1
  node_type   = "SHARED_CORE_NANO"
  desired_psc_auto_connections {
    network    = var.vpc_id
    project_id = var.projects.internal
  }
  engine_version              = "VALKEY_8_0"
  location                    = each.key
  depends_on                  = [google_network_connectivity_service_connection_policy.valkey_policy]
  deletion_protection_enabled = var.deletion_protection
}
