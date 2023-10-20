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

resource "google_firestore_database" "firestore_db" {
  # project                     = data.google_project.project.id
  name        = "${var.env_id}-db"
  location_id = "us-east1"
  type        = "DATASTORE_MODE"
  # concurrency_mode            = "OPTIMISTIC"
  # app_engine_integration_mode = "DISABLED"

}