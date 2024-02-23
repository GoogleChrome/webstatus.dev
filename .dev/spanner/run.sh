#!/bin/bash

gcloud emulators spanner start --host-port=0.0.0.0:9010 --rest-port=9020 --project="${SPANNER_PROJECT_ID}" --log-http --verbosity=debug --user-output-enabled &
while ! curl -s -o /dev/null localhost:9020; do
  sleep 1 # Wait 1 second before checking again
  echo "waiting until 9020 responds"
done
gcloud spanner instances create "${SPANNER_INSTANCE_ID}" --config=emulator-config --description='Local Instance' --nodes=1 --verbosity=debug
gcloud spanner databases create "${SPANNER_DATABASE_ID}" --instance "${SPANNER_INSTANCE_ID}" --verbosity=debug
# shellcheck disable=SC2091
$(gcloud emulators spanner env-init)

# Setup database
wrench reset --directory ./schemas/ --schema_file spanner.sql

wait
