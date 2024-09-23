# Copyright 2024 Google LLC
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

variable "go_module_path" {
  type        = string
  description = "The path to go module. This path is relative to the root of the repository."
}

variable "image_name" {
  description = "Name of the docker image"
  type        = string
}

variable "binary_type" {
  description = "The arg used in to tell go_service.Dockerfile whether it is a long standing 'server' or an ad-hoc 'job'"
  type        = string
  validation {
    condition     = contains(["job", "server"], var.binary_type)
    error_message = "Valid values for var: binary_type are (job, server)."
  }
}

variable "docker_repository_url" {
  description = "The docker repository url"
}
