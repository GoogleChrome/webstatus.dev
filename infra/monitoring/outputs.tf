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

output "uptime_check_id" {
  description = "The ID of the Cloud Monitoring synthetic uptime check"
  value       = google_monitoring_uptime_check_config.webstatus_uptime_check.uptime_check_id
}

output "alert_policy_id" {
  description = "The ID of the outage alert policy"
  value       = google_monitoring_alert_policy.webstatus_outage_alert.name
}
