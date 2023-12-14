```
docker run -d --name fake-gcs-server -p 4443:4443 -v ${PWD}/gcs-data:/data fsouza/fake-gcs-server -scheme http
STORAGE_EMULATOR_HOST=http://localhost:4443
BUCKET=testbucket
```
