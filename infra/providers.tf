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

terraform {
  backend "gcs" {}
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.4.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = ">= 3.0.2"
    }
  }
}

provider "google" {
  project = var.project_name
}

provider "google" {
  alias   = "internal_project"
  project = var.projects.internal
  # Need user_project_override=true for identity platform
  # https://stackoverflow.com/a/78203631
  user_project_override = true
}

provider "google" {
  alias   = "public_project"
  project = var.projects.public
}

provider "docker" {
  host = "unix:///var/run/docker.sock"
  registry_auth {
    address     = module.storage.docker_repository_details.hostname
    config_file = pathexpand("~/.docker/config.json")
  }
}
