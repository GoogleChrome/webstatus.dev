# Development

## Requirements

The recommended way to do development is through the provided devcontainer. To
run a devcontainer, check out this
[web page](https://code.visualstudio.com/docs/devcontainers/containers#_system-requirements)
for the latest requirements to run devcontainer. The devcontainer will have
everything pre-installed.

# Running locally

## Running

```sh
# Terminal 1
skaffold dev -p local
```

### Locally Deployed Resources

The above skaffold command deploys multiple resources:

| Resource | Description | Port Forwarded Address | Internal Address |
| --- | ----------- | --------------- | --------------- |
| backend | Backend service in ./backend | http://localhost:8080 | http://backend:8080 |
| frontend | Frontend service in ./frontend | http://localhost:5555 | http://frontend:5555 |
| datastore | Datastore Emulator | N/A | http://datastore:8085 |
| spanner | Spanner Emulator | N/A | spanner:9010 (grpc)<br />http://spanner:9020 (rest) |
| gcs | Google Cloud Storage Emulator | N/A | http://gcs:4443 |
| repo-downloader | Repo Downloader Workflow Step in<br />./workflows/steps/services/common/repo_downloader | http://localhost:8091 | http://repo-downloader:8080 |
| web-feature-consumer | Web Feature Consumer Step in<br />./workflows/steps/services/web_feature_consumer | http://localhost:8092 | http://web-feature-consumer:8080 |

### Populate Data Locally

After doing an initial deployment, the databases will be empty, run the following:

```sh
# Terminal 2 - Populate data
DOWNLOAD_RESPONSE=$(curl -X 'POST' \
  'http://localhost:8091/v1/github.com/web-platform-dx/web-features' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
  "archive": {
    "type": "TAR",
    "tar_strip_components": 1
  },
  "file_filters": [
    {
      "prefix": "feature-group-definitions",
      "suffix": ".yml"
    }
  ]
}')
OBJECT_PREFIX=$(echo $DOWNLOAD_RESPONSE | jq -r -c '.destination.gcs.repo_prefix')
BUCKET=$(echo $DOWNLOAD_RESPONSE | jq -r -c '.destination.gcs.bucket')
echo $DOWNLOAD_RESPONSE | jq -r -c '.destination.gcs.filenames[]' | while read object; do
    curl -X 'POST' \
        'http://localhost:8092/v1/web-features' \
        -H 'accept: */*' \
        -H 'Content-Type: application/json' \
        -d "{
            \"location\": {
            \"gcs\": {
                \"bucket\": \"${BUCKET}\",
                \"object\": \"${OBJECT_PREFIX}/${object}\"
            }
        }
    }"
done
```

Then open `http://localhost:8080/v1/features` to see the features populated
from the latest snapshot from the web-features repo.

## OpenAPI

Every web service has its own OpenAPI description.

| Resource | Location |
| -------- | -------- |
| backend  | [openapi/backend/openapi.yaml](openapi/backend/openapi.yaml) |
| repo-downloader | [openapi/workflows/steps/common/repo_downloader/openapi.yaml](openapi/workflows/steps/common/repo_downloader/openapi.yaml) |
| web-feature-consumer | [openapi/workflows/steps/web_feature_consumer/openapi.yaml](openapi/workflows/steps/web_feature_consumer/openapi.yaml) |

### Go and OpenAPI

There two common configurations used to generate code for Go.

- [openapi/server.cfg.yaml](openapi/server.cfg.yaml)
- [openapi/types.cfg.yaml](openapi/types.cfg.yaml)

This repository uses
[deepmap/oapi-codegen](https://github.com/deepmap/oapi-codegen) to generate the
types.

If changes are made, run:

```sh
make -B openapi
```

### TypeScript and OpenAPI

TODO

## JSONSchema

TODO

# Deploying

## Deploying own copy

```sh
cd infra
gcloud auth application-default login --project=web-compass-staging
ENV_ID="some-unique-string-here" # Something 6 characters long. Could use "openssl rand -hex 3"
# SAVE THAT ENV_ID
terraform workspace new $ENV_ID
terraform init --var-file=.envs/staging.tfvars
terraform plan \
    -var-file=".envs/staging.tfvars" \
    -var "env_id=${ENV_ID}" \
    -var "spanner_processing_units=100" \
    -var "deletion_protection=false" \
    -var "datastore_region_id=us-east1"
```

When you are done with your own copy

```sh
terraform destroy \
    -var-file=".envs/staging.tfvars" \
    -var "env_id=${ENV_ID}" \
    -var "spanner_processing_units=100" \
    -var "deletion_protection=false" \
    -var "datastore_region_id=us-east1"
terraform workspace select default
terraform workspace delete $ENV_ID
```

## Deploying staging

TODO
