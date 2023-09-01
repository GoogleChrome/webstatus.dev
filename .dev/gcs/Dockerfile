FROM fsouza/fake-gcs-server:1.47

# https://cloud.google.com/datastore/docs/tools/datastore-emulator
CMD ["-scheme", "http", "-public-host", "gcs:4443", "-external-url", "http://gcs:4443"]
