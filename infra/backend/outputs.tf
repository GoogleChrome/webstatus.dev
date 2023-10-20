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
  backend_regional_info = tolist([
    for service in google_cloud_run_v2_service.service :
    {
      "url" : service.uri
      "name" : service.name
    }
  ])
}

output "region_to_backend_info_map" {
  value = zipmap(var.regions, local.backend_regional_info)
}

# output "backend_dns_host" {
#   value = 
#   description = "High level DNS Host"
# }