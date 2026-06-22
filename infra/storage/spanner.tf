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
  project       = var.projects.internal
  name          = "${var.env_id}-spanner"
  config        = var.spanner_region_id
  display_name  = "${var.env_id} Spanner"
  force_destroy = !var.deletion_protection

  # If processing units are less than 1000 (fractional nodes), use static allocation.
  # Otherwise, set to null and let autoscaling manage it.
  processing_units = var.spanner_processing_units < 1000 ? var.spanner_processing_units : null

  # Only enable autoscaling if we have at least 1 node (1000 PUs) to scale with.
  dynamic "autoscaling_config" {
    for_each = var.spanner_processing_units >= 1000 ? [1] : []
    content {
      autoscaling_limits {
        min_processing_units = 1000
        max_processing_units = var.spanner_processing_units
      }
      autoscaling_targets {
        high_priority_cpu_utilization_percent = 65
        storage_utilization_percent           = 80
      }
    }
  }
}

resource "google_spanner_database" "database" {
  project                  = var.projects.internal
  instance                 = google_spanner_instance.main.name
  name                     = "${var.env_id}-database"
  version_retention_period = "3d"
  deletion_protection      = var.deletion_protection
}