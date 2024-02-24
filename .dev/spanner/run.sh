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


gcloud emulators spanner start --host-port=0.0.0.0:9010 --rest-port=9020 --project="${SPANNER_PROJECT_ID}" --log-http --verbosity=debug --user-output-enabled &
while ! curl -s -o /dev/null localhost:9020; do
  sleep 1 # Wait 1 second before checking again
  echo "waiting until spanner emulator responds before finishing setup"
done

# For the following commands, exit on any error.
set -e

gcloud spanner instances create "${SPANNER_INSTANCE_ID}" --config=emulator-config --description='Local Instance' --nodes=1 --verbosity=debug
# shellcheck disable=SC2091
$(gcloud emulators spanner env-init)

# Setup database
wrench  create --directory ./schemas/

# Print migrations for debugging purposes.
for file in ./schemas/migrations/*; do
  # Check if it's a regular file
  if [ -f "$file" ]; then
    echo "----- File: $file -----"
    cat "$file"
    echo "------------------------"
  fi
done

# Perform migrations
wrench migrate up --directory ./schemas/

echo "Spanner setup for webstatus.dev finished"


wait
