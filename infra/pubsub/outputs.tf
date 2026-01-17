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

output "ingestion_topic_id" {
  value = google_pubsub_topic.ingestion_jobs.id
}

output "ingestion_subscription_id" {
  value = google_pubsub_subscription.ingestion_jobs_sub.id
}

output "batch_updates_topic_id" {
  value = google_pubsub_topic.batch_updates.id
}

output "batch_updates_subscription_id" {
  value = google_pubsub_subscription.batch_updates_sub.id
}

output "notification_topic_id" {
  value = google_pubsub_topic.notification_events.id
}

output "notification_subscription_id" {
  value = google_pubsub_subscription.notification_events_sub.id
}

output "email_delivery_topic_id" {
  value = google_pubsub_topic.chime_delivery.id
}

output "email_delivery_subscription_id" {
  value = google_pubsub_subscription.chime_delivery_sub.id
}
