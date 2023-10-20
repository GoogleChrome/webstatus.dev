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

resource "google_spanner_instance" "main" {
  name             = "${var.env_id}-spanner"
  config           = var.spanner_region_id
  display_name     = "${var.env_id} Spanner"
  processing_units = var.spanner_processing_units
  force_destroy    = !var.deletion_protection
}

resource "google_spanner_database" "database" {
  instance                 = google_spanner_instance.main.name
  name                     = "${var.env_id}-database"
  version_retention_period = "3d"
  deletion_protection      = var.deletion_protection
}