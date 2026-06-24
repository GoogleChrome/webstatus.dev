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
  # Google Cloud Spanner requires a minimum of 1000 Processing Units (1 node)
  # to enable the Managed Autoscaler.
  min_autoscaling_processing_units = 1000
}

resource "google_spanner_instance" "main" {
  project       = var.projects.internal
  name          = "${var.env_id}-spanner"
  config        = var.spanner_region_id
  display_name  = "${var.env_id} Spanner"
  force_destroy = !var.deletion_protection
  edition       = var.spanner_edition

  # If processing units are less than the minimum required for autoscaling, use static allocation.
  # Otherwise, set to null and let the autoscaler manage it.
  processing_units = var.spanner_processing_units < local.min_autoscaling_processing_units ? var.spanner_processing_units : null

  # Only enable autoscaling if we have at least 1 node (1000 PUs) to scale with.
  dynamic "autoscaling_config" {
    for_each = var.spanner_processing_units >= local.min_autoscaling_processing_units ? [1] : []
    content {
      autoscaling_limits {
        min_processing_units = local.min_autoscaling_processing_units
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