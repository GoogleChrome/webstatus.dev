#!/bin/bash
# Copyright 2025 Google LLC
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


# Function to create a Topic. gcloud does not support topic creation in the emulator, so we use curl.
create_topic() {
    local topic_name=$1

    if [[ -z "$topic_name" ]]; then
        echo "Error: Topic name required."
        return 1
    fi

    echo "Creating topic: $topic_name..."

    curl -s -X PUT "http://0.0.0.0:${PORT}/v1/projects/${PROJECT_ID}/topics/${topic_name}" \
        -H "Content-Type: application/json"

    echo -e "\nTopic ${topic_name} created."
}

# Function to create a Subscription (Pull). gcloud does not support subscription creation in the emulator, so we use curl.
# Usage: create_subscription <topic_name> <sub_name> [dlq_topic_name]
create_subscription() {
    local topic_name=$1
    local sub_name=$2
    local dlq_topic_name=$3

    if [[ -z "$topic_name" || -z "$sub_name" ]]; then
        echo "Error: Topic name and Subscription name required."
        return 1
    fi

    echo "Creating subscription: ${sub_name} for topic: ${topic_name}..."

    # The emulator requires the full path to the topic in the JSON body
    local topic_path="projects/${PROJECT_ID}/topics/${topic_name}"

    # Base JSON payload
    local json_payload="{\"topic\": \"$topic_path\""

    # If a DLQ topic is provided, add the deadLetterPolicy
    if [[ -n "$dlq_topic_name" ]]; then
        local dlq_topic_path="projects/${PROJECT_ID}/topics/${dlq_topic_name}"
        # MaxDeliveryAttempts is set to 5 as a standard default
        json_payload="${json_payload}, \"deadLetterPolicy\": {\"deadLetterTopic\": \"${dlq_topic_path}\", \"maxDeliveryAttempts\": 5}"
        echo "  -> With Dead Letter Queue: ${dlq_topic_name}"
    fi

    # Close the JSON object
    json_payload="${json_payload}}"

    curl -s -X PUT "http://0.0.0.0:${PORT}/v1/projects/${PROJECT_ID}/subscriptions/${sub_name}" \
        -H "Content-Type: application/json" \
        -d "$json_payload"

    echo -e "\nSubscription ${sub_name} for topic: ${topic_name} created."
}

gcloud beta emulators pubsub start --project="$PROJECT_ID" --host-port="0.0.0.0:$PORT" &
while ! curl -s -f "http://0.0.0.0:${PORT}/v1/projects/${PROJECT_ID}/topics"; do
  sleep 1 # Wait 1 second before checking again
  echo "waiting until pubsub emulator responds before finishing setup"
done

# --- 1. Dead Letter Queues (Create these first so main subs can reference them) ---

# Ingestion DLQ
create_topic "ingestion-jobs-dead-letter-topic-id"
create_subscription "ingestion-jobs-dead-letter-topic-id" "ingestion-jobs-dead-letter-sub-id"

# Notification/Fan-out DLQ
create_topic "notification-events-dead-letter-topic-id"
create_subscription "notification-events-dead-letter-topic-id" "notification-events-dead-letter-sub-id"

# Delivery DLQ
create_topic "delivery-dead-letter-topic-id"
create_subscription "delivery-dead-letter-topic-id" "delivery-dead-letter-sub-id"


# --- 2. Main Topics and Subscriptions ---
create_topic "batch-updates-topic-id"
create_subscription "batch-updates-topic-id" "batch-updates-sub-id" "ingestion-jobs-dead-letter-topic-id"

create_topic "ingestion-jobs-topic-id"
create_subscription "ingestion-jobs-topic-id" "ingestion-jobs-sub-id" "ingestion-jobs-dead-letter-topic-id"

create_topic "notification-events-topic-id"
create_subscription "notification-events-topic-id" "notification-events-sub-id" "notification-events-dead-letter-topic-id"

create_topic "chime-delivery-topic-id"
create_subscription "chime-delivery-topic-id" "chime-delivery-sub-id" "delivery-dead-letter-topic-id"

echo "Pub/Sub setup for webstatus.dev finished"

wait
