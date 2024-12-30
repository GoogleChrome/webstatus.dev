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
    -var "datastore_region_id=us-east1" \
    -var "backend_domains=[\"${ENV_ID}-api.test.webstatus.dev\"]" \
    -var "frontend_domains=[\"${ENV_ID}-website.test.webstatus.dev\"]" \
    -var "custom_ssl_certificates_for_frontend=[]" \
    -var "custom_ssl_certificates_for_backend=[]"
```

That will print the plan to create everything. Once it looks okay, run:

```sh
terraform apply \
    -var-file=".envs/staging.tfvars" \
    -var "env_id=${ENV_ID}" \
    -var "spanner_processing_units=100" \
    -var "deletion_protection=false" \
    -var "datastore_region_id=us-east1" \
    -var "backend_domains=[\"${ENV_ID}-api.test.webstatus.dev\"]" \
    -var "frontend_domains=[\"${ENV_ID}-website.test.webstatus.dev\"]" \
    -var "custom_ssl_certificates_for_frontend=[]" \
    -var "custom_ssl_certificates_for_backend=[]"
```

**NOTE**: You may need to run it multiple times if it gets stuck.

Create the tables by running:

```sh
export SPANNER_PROJECT_ID=webstatus-dev-internal-staging
export SPANNER_DATABASE_ID=${ENV_ID}-database
export SPANNER_INSTANCE_ID=${ENV_ID}-spanner
wrench migrate up --directory ./storage/spanner/
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
    -var "datastore_region_id=us-east1" \
    -var "custom_ssl_certificates_for_frontend=[]" \
    -var "custom_ssl_certificates_for_backend=[]"

If it fails, run: `./network/clean_up.sh web-compass-staging ${ENV_ID}-webstatus-dev-network`

terraform workspace select default
terraform workspace delete $ENV_ID
```

Also, remove your domain from the allow-list of domains in the [console](https://console.cloud.google.com/customer-identity/settings?project=webstatus-dev-internal-staging).

## Deploy Staging

```sh
make build
cd infra
gcloud auth login
gcloud auth application-default login --project=web-compass-staging
gcloud auth configure-docker europe-west1-docker.pkg.dev --quiet
ENV_ID="staging"
export TF_WORKSPACE=${ENV_ID}
terraform init -reconfigure --var-file=.envs/staging.tfvars --backend-config=.envs/backend-staging.tfvars
terraform plan \
    -var-file=".envs/staging.tfvars" \
    -var "env_id=${ENV_ID}"
```

Migrate the tables if any schemas changed (assuming you already authenticated with `gcloud auth application-default login`):

```sh
export SPANNER_PROJECT_ID=webstatus-dev-internal-staging
export SPANNER_DATABASE_ID=${ENV_ID}-database
export SPANNER_INSTANCE_ID=${ENV_ID}-spanner
wrench migrate up --directory ./storage/spanner/
```

Assuming the plan output by the terraform plan command looks fine, run:

```sh
terraform apply \
    -var-file=".envs/staging.tfvars" \
    -var "env_id=${ENV_ID}"
```

### Update the value of the BCD progress

Go to https://github.com/web-platform-dx/web-features/releases and there should be a line like `X% coverage of mdn/browser-compat-data features`. Copy the percentage.

```sh
BCD_VALUE=62 # Replace the value
echo "{\"bcd_map_progress\": $BCD_VALUE}"  | gsutil -h "Content-Type:application/json" cp - gs://web-features-progress-staging/progress.json
```

### Or, disable the BCD progress text

**Note:** Normally, we should only need to update the progress. **ONLY** do
this section if there is a request to disable the text. Otherwise, you can skip
this part and consider the deployment done.

```sh
echo "{\"is_disabled\": true}"  | gsutil -h "Content-Type:application/json" cp - gs://web-features-progress-staging/progress.json
```

## Deploy Prod

```sh
make build
cd infra
gcloud auth login
gcloud auth application-default login --project=web-compass-prod
gcloud auth configure-docker europe-west1-docker.pkg.dev --quiet
ENV_ID="prod"
export TF_WORKSPACE=${ENV_ID}
terraform init -reconfigure --var-file=.envs/prod.tfvars --backend-config=.envs/backend-prod.tfvars

terraform plan \
    -var-file=".envs/prod.tfvars" \
    -var "env_id=${ENV_ID}"
```

Migrate the tables if any schemas changed (assuming you already authenticated with `gcloud auth application-default login`):

```sh
export SPANNER_PROJECT_ID=webstatus-dev-internal-prod
export SPANNER_DATABASE_ID=${ENV_ID}-database
export SPANNER_INSTANCE_ID=${ENV_ID}-spanner
wrench migrate up --directory ./storage/spanner/
```

Assuming the plan output by the terraform plan command looks fine, run:

```sh
terraform apply \
    -var-file=".envs/prod.tfvars" \
    -var "env_id=${ENV_ID}"
```

### Update the value of the BCD progress

Go to https://github.com/web-platform-dx/web-features/releases and there should be a line like `X% coverage of mdn/browser-compat-data features`. Copy the percentage.

```sh
BCD_VALUE=62 # Replace the value
echo "{\"bcd_map_progress\": $BCD_VALUE}"  | gsutil -h "Content-Type:application/json" cp - gs://web-features-progress-prod/progress.json
```

### Or, disable the BCD progress text

**Note:** Normally, we should only need to update the progress. **ONLY** do
this section if there is a request to disable the text. Otherwise, you can skip
this part and consider the deployment done.

```sh
echo "{\"is_disabled\": true}"  | gsutil -h "Content-Type:application/json" cp - gs://web-features-progress-prod/progress.json
```
