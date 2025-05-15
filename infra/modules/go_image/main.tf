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

resource "docker_image" "local_image" {
  name = "${var.docker_repository_url}/${var.image_name}"
  build {
    context = "${path.cwd}/.."
    build_args = {
      service_dir : var.go_module_path
      MAIN_BINARY : var.binary_type
    }
    dockerfile = "${path.cwd}/../images/go_service.Dockerfile"
    # Use buildx default builder instead of legacy builder
    # Needed for the --mount args
    # Must also specify platform too.
    builder        = "default"
    platform       = "linux/amd64"
    build_log_file = "${path.cwd}/${var.image_name}.log"
  }
  triggers = {
    dir_sha1 = sha1(join("",
      # Rebuild the image if anything changes in the go module itself.
      [for f in fileset(path.cwd, "/../${var.go_module_path}/**") : filesha1(f)],
      # Rebuild the go image if anything changes in the lib directory.
      [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]
    ))
  }
}

resource "docker_registry_image" "remote_image" {
  name          = docker_image.local_image.name
  keep_remotely = true
  triggers = {
    dir_sha1 = sha1(join("",
      # Rebuild the image if anything changes in the go module itself.
      [for f in fileset(path.cwd, "/../${var.go_module_path}/**") : filesha1(f)],
      # Rebuild the go image if anything changes in the lib directory.
      [for f in fileset(path.cwd, "/../lib/**") : filesha1(f)]
    ))
  }
}
