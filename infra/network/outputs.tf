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

output "region_to_subnet_info_map" {
  value = {
    for region, value in var.region_to_subnet_map :
    region => {
      internal    = google_compute_subnetwork.subnetwork_internal[region].name
      internal_id = google_compute_subnetwork.subnetwork_internal[region].id
      public      = google_compute_subnetwork.subnetwork_public[region].name
    }
  }
}

output "vpc_name" {
  value = google_compute_network.shared_vpc.name
}
output "vpc_id" {
  value = google_compute_network.shared_vpc.id
}
