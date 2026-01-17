# Copyright 2026 Google LLC
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

resource "google_storage_bucket" "notification_state" {
  provider                    = google.internal_project
  project                     = var.projects.internal
  name                        = "notification-state-${var.projects.internal}-${var.env_id}"
  location                    = "US"
  uniform_bucket_level_access = true
  storage_class               = "MULTI_REGIONAL"

  versioning {
    enabled = true
  }
}

# Export the name so it can be passed to the application
output "notification_state_bucket_name" {
  value = google_storage_bucket.notification_state.name
}
