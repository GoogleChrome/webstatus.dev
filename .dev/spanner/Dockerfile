FROM google/cloud-sdk:442.0.0-emulators


ENTRYPOINT [ "gcloud", "emulators", "spanner", "start", "--host-port=0.0.0.0:9010", "--rest-port=9020", "--project=local", "--log-http", "--verbosity=debug", "--user-output-enabled" ]
