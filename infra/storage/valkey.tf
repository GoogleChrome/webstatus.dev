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

resource "google_network_connectivity_service_connection_policy" "valkey_policy" {
  provider      = google-beta
  for_each      = var.region_to_subnet_info_map
  name          = "${var.env_id}-${each.key}-valkey-policy"
  location      = each.key
  service_class = "gcp-memorystore"
  description   = "${var.env_id} service connection policy for ${each.key}"
  network       = var.vpc_id
  psc_config {
    subnetworks = [each.value.internal]
  }
}

resource "google_memorystore_instance" "valkey_instance" {
  provider    = google-beta
  for_each    = var.region_to_subnet_info_map
  instance_id = "${var.env_id}-valkey-${each.key}"
  shard_count = 2
  desired_psc_auto_connections {
    network    = var.vpc_id
    project_id = data.google_project.project.project_id
  }
  engine_version = "VALKEY_8_0"
  location       = each.key
  depends_on     = [google_network_connectivity_service_connection_policy.valkey_policy]
}
