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
make start-local
```

Everything will start up without port forwarding enabled. (Sometimes skaffold
thinks a port is occcupied and will assign a different port so we disabled it
that logic.) As you make changes, the service will recompile and deploy automatically.

Once everything comes up, open a second terminal.

```sh
# Terminal 2
make port-forward-manual
```

The output should match this:

```sh
$ make port-forward-manual
pkill kubectl -9 || true
kubectl wait --for=condition=ready pod/frontend
pod/frontend condition met
kubectl wait --for=condition=ready pod/backend
pod/backend condition met
kubectl port-forward --address 127.0.0.1 pod/frontend 5555:5555 2>&1 >/dev/null &
kubectl port-forward --address 127.0.0.1 pod/backend 8080:8080 2>&1 >/dev/null &
```

_Note_: Ignore the WARN statements that are printed in terminal 1 as a result of this
command. However, if the WARN statements appear later on and you did not run
this command or the `make port-forward-terminate` command, something may be wrong.

If you terminate everything in terminal 1, run this to clean up:

```sh
# Terminal 2
make port-forward-terminate
```

_Note_: Ignore the WARN statements that are printed in terminal 1 as a result of this
command. However, if the WARN statements appear later on and you did not run
this command or the `make port-forward-manual` command, something may be wrong.

### Locally Deployed Resources

The above skaffold command deploys multiple resources:

| Resource             | Description                                                                             | Port Forwarded Address | Internal Address                                    |
| -------------------- | --------------------------------------------------------------------------------------- | ---------------------- | --------------------------------------------------- |
| backend              | Backend service in ./backend                                                            | http://localhost:8080  | http://backend:8080                                 |
| frontend             | Frontend service in ./frontend                                                          | http://localhost:5555  | http://frontend:5555                                |
| datastore            | Datastore Emulator                                                                      | N/A                    | http://datastore:8085                               |
| spanner              | Spanner Emulator                                                                        | N/A                    | spanner:9010 (grpc)<br />http://spanner:9020 (rest) |
| redis                | Redis                                                                                   | N/A                    | redis:6379                                          |
| gcs                  | Google Cloud Storage Emulator                                                           | N/A                    | http://gcs:4443                                     |
| repo-downloader      | Repo Downloader Workflow Step in<br />./workflows/steps/services/common/repo_downloader | http://localhost:8091  | http://repo-downloader:8080                         |
| web-feature-consumer | Web Feature Consumer Step in<br />./workflows/steps/services/web_feature_consumer       | http://localhost:8092  | http://web-feature-consumer:8080                    |

_In the event the servers are not responsive, make a temporary change to a file_
_in a watched directory (e.g. backend). This will rebuild and expose the_
_services._

### Populate Data Locally

After doing an initial deployment, the databases will be empty. Currently, you
can run a local version of the workflow to populate your database.

#### Option 1: Run local workflow to populate database

Run the following:

```sh
# Terminal 2 - Run local workflows
make dev_workflows
```

_Note: If the command fails, there might be a problem with the live data it is pulling_

#### Option 2: Run command to populate with fake data

An option could be to populate the database with dummy data. This is useful if
the live data sources are down or constantly changing.

_TODO_

#### Verify the database has data

Open `http://localhost:8080/v1/features` to see the features populated
from the latest snapshot from the web-features repo.

## OpenAPI

Every web service has its own OpenAPI description.

| Resource             | Location                                                                                                                   |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| backend              | [openapi/backend/openapi.yaml](openapi/backend/openapi.yaml)                                                               |
| repo-downloader      | [openapi/workflows/steps/common/repo_downloader/openapi.yaml](openapi/workflows/steps/common/repo_downloader/openapi.yaml) |
| web-feature-consumer | [openapi/workflows/steps/web_feature_consumer/openapi.yaml](openapi/workflows/steps/web_feature_consumer/openapi.yaml)     |

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
gcloud auth login
gcloud auth application-default login --project=web-compass-staging --no-browser
# Something 6 characters long. Could use "openssl rand -hex 3"
ENV_ID="some-unique-string-here"
# SAVE THAT ENV_ID
terraform init -reconfigure --var-file=.envs/staging.tfvars --backend-config=.envs/backend-staging.tfvars
terraform workspace new $ENV_ID
terraform plan \
    -var-file=".envs/staging.tfvars" \
    -var "env_id=${ENV_ID}" \
    -var "spanner_processing_units=100" \
    -var "deletion_protection=false" \
    -var "datastore_region_id=us-east1"
```

That will print the plan to create everything. Once it looks okay, run:

```sh
terraform apply \
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

```sh
cd infra
gcloud auth login
gcloud auth application-default login --project=web-compass-staging --no-browser
ENV_ID="staging"
terraform init -reconfigure --var-file=.envs/staging.tfvars --backend-config=.envs/backend-staging.tfvars
terraform workspace select $ENV_ID
terraform plan \
    -var-file=".envs/staging.tfvars" \
    -var "env_id=${ENV_ID}"
```

That will print the plan to create everything. Once it looks okay, run:

```sh
terraform apply \
    -var-file=".envs/staging.tfvars" \
    -var "env_id=${ENV_ID}"
```

```sh
export SPANNER_PROJECT_ID=webstatus-dev-internal-staging
gcloud auth application-default login --project=${SPANNER_PROJECT_ID} --no-browser
export SPANNER_DATABASE_ID=${ENV_ID}-database
export SPANNER_INSTANCE_ID=${ENV_ID}-spanner
wrench migrate up --directory ./storage/spanner/

# In root directory
go run ./util/cmd/load_fake_data/main.go -spanner_project=${SPANNER_PROJECT_ID} -spanner_instance=${SPANNER_INSTANCE_ID} -spanner_database=${SPANNER_DATABASE_ID}
```

## Deploying prod

```sh
cd infra
gcloud auth login
gcloud auth application-default login --project=web-compass-prod --no-browser
ENV_ID="prod"
terraform init -reconfigure --var-file=.envs/prod.tfvars --backend-config=.envs/backend-prod.tfvars
terraform workspace select $ENV_ID

terraform plan \
    -var-file=".envs/prod.tfvars" \
    -var "env_id=${ENV_ID}"
```

That will print the plan to create everything. Once it looks okay, run:

```sh
terraform apply \
    -var-file=".envs/prod.tfvars" \
    -var "env_id=${ENV_ID}"
```

```sh
export SPANNER_PROJECT_ID=webstatus-dev-internal-prod
gcloud auth application-default login --project=${SPANNER_PROJECT_ID} --no-browser
export SPANNER_DATABASE_ID=${ENV_ID}-database
export SPANNER_INSTANCE_ID=${ENV_ID}-spanner
wrench migrate up --directory ./storage/spanner/
```
