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

variable "env_id" {
  type = string
}

variable "host_project_id" {
  type = string
}

variable "region_to_subnet_map" {
  type = map(map(object({
    cidr = string
  })))
  description = <<EOF
  maps region to a map. that map is the purpose of the subnet. the secondary map maps to an object which contains the subnet info
EOF
}