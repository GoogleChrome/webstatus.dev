# Development

## Requirements

The recommended way to do development is through the provided devcontainer. To
run a devcontainer, check out this
[web page](https://code.visualstudio.com/docs/devcontainers/containers#_system-requirements)
for the latest requirements to run devcontainer. The devcontainer will have
everything pre-installed.

# Running locally

*Notice:* Due to this
[issue](https://github.com/GoogleContainerTools/skaffold/issues/9006), we need
to use a manually built version of skaffold. It is installed already in the
devcontainer. Also, `skaffold dev` currently creates new containers on new ports
instead of replacing them on existing ports. As a result, we need to disable the
auto build functionality.

## Running

```sh
skaffold dev -p local --auto-build=true
```

It should print something like:

```
[backend] Forwarding container port 8080 -> local port http://127.0.0.1:XXXX
[frontend] Forwarding container port 8000 -> local port http://127.0.0.1:XXXX
```

You'll want the backend and frontned to be forwarded to http://127.0.0.1:8080
and http://127.0.0.1:8000 respectively. If not, check if there's something
already running. If you may have another set of docker containers running, refer
to the helpful commands.

You may open your browser to those two addresses now! :)

## Helpful Commands

- `docker kill $(docker ps -q)` delete all running containers.

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
    -var "deletion_protection=false"
```

When you are done with your own copy

```sh
terraform destroy \
    -var-file=".envs/staging.tfvars" \
    -var "env_id=${ENV_ID}" \
    -var "spanner_processing_units=100" \
    -var "deletion_protection=false"
terraform workspace select default
terraform workspace delete $ENV_ID
```

## Deploying staging