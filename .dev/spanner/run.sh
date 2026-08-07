#!/bin/bash
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


/bin/gateway_main --hostname=0.0.0.0 --http_port=9020 --grpc_port=9010 &

while ! curl -s -o /dev/null http://localhost:9020/; do
  sleep 1 # Wait 1 second before checking again
  echo "waiting until spanner emulator responds before finishing setup"
done

# For the following commands, exit on any error.
set -e

# Create instance via Spanner Emulator REST API
curl -s -X POST "http://localhost:9020/v1/projects/${SPANNER_PROJECT_ID}/instances" \
  -H "Content-Type: application/json" \
  -d "{\"instanceId\": \"${SPANNER_INSTANCE_ID}\", \"instance\": {\"displayName\": \"Local Instance\"}}"

export SPANNER_EMULATOR_HOST="localhost:9010"

# Setup database
wrench create --directory ./schemas/

# Perform migrations
wrench migrate up --directory ./schemas/

echo "Spanner setup for webstatus.dev finished"


wait
