# Deploying

## Table Of Contents

1. Requirements
2. Deploying your own copy
3. Deploy Staging
4. Deploy Prod

## Requirements

Users will need to have the following tools installed:

- [terraform](https://www.terraform.io/)
- [wrench](https://github.com/cloudspannerecosystem/wrench)
- [Docker](https://www.docker.com/)

_Note:_ These tools are provided given you are using the devcontainer.

Users will also need to have access to GCP. In particular:

These instructions could be adapted to use any GCP project however these
instructions assume you have access to the following projects:

- staging
  - web-compass-staging
  - webstatus-dev-internal-staging
  - webstatus-dev-public-staging
- production
  - web-compass-prod
  - webstatus-dev-internal-prod
  - webstatus-dev-public-prod

Google Cloud Identity Platform:

- [Enable](https://console.cloud.google.com/marketplace/details/google-cloud-platform/customer-identity) Cloud Identity Platform for the internal project.
- [Enable](https://cloud.google.com/identity-platform/docs/multi-tenancy-quickstart) multi-tenancy in the Google Cloud Console.

## Deploying your own copy

```sh
make build
cd infra
gcloud auth login
gcloud auth application-default login --project=web-compass-staging
gcloud auth configure-docker europe-west1-docker.pkg.dev --quiet
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
    -var "notification_channel_ids=[]" \
    -var "datastore_region_id=us-east1"
```

That will print the plan to create everything. Once it looks okay, run:

```sh
terraform apply \
    -var-file=".envs/staging.tfvars" \
    -var "env_id=${ENV_ID}" \
    -var "spanner_processing_units=100" \
    -var "deletion_protection=false" \
    -var "notification_channel_ids=[]" \
    -var "datastore_region_id=us-east1"
```

Create the tables by running:

```sh
export SPANNER_PROJECT_ID=webstatus-dev-internal-staging
export SPANNER_DATABASE_ID=${ENV_ID}-database
export SPANNER_INSTANCE_ID=${ENV_ID}-spanner
go tool wrench migrate up --directory ./storage/spanner/
```

Populate data:

You can populate data with real data by manually running the workflows in the
internal project.

Or you could populate with fake data by running.

```
go run ./util/cmd/load_fake_data/main.go -spanner_project=${SPANNER_PROJECT_ID} -spanner_instance=${SPANNER_INSTANCE_ID} -spanner_database=${SPANNER_DATABASE_ID} -datastore_project=${DATASTORE_PROJECT_ID} -datastore_database=${DATASTORE_DATABASE}
```

Setup auth:

Add your domain to the allow-list of domains in the [console](https://console.cloud.google.com/customer-identity/settings?project=webstatus-dev-internal-staging).

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

Also, remove your domain from the allow-list of domains in the [console](https://console.cloud.google.com/customer-identity/settings?project=webstatus-dev-internal-staging).

## Deploy Staging

Run the script `deploy-staging` in the devcontainer and follow the output and
prompts to complete the deployment.

## Deploy Prod

Run the script `deploy-production` in the devcontainer and follow the output and
prompts to complete the deployment.
