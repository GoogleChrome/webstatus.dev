FROM google/cloud-sdk:442.0.0-emulators

# https://cloud.google.com/firestore/docs/emulator
ENTRYPOINT [ "gcloud", "emulators", "firestore", "start", "--host-port=0.0.0.0:8085", "--project=local", "--log-http", "--verbosity=debug", "--user-output-enabled" ]
