# Maintenance

Most dependencies are handled in their appropriate lock files. This doc pinpoint
any other dependencies that may not be clear.

## Devcontainer dependencies

| Binary                  | File to update                                          |
| ----------------------- | ------------------------------------------------------- |
| gcloud                  | devcontainer [Dockerfile](../.devcontainer/Dockerfile)  |
| go                      | [devcontainer.json](../.devcontainer/devcontainer.json) |
| Github CLI              | [devcontainer.json](../.devcontainer/devcontainer.json) |
| terraform               | [devcontainer.json](../.devcontainer/devcontainer.json) |
| shellcheck              | [devcontainer.json](../.devcontainer/devcontainer.json) |
| kubectl, helm, minikube | [devcontainer.json](../.devcontainer/devcontainer.json) |
| skaffold                | [devcontainer.json](../.devcontainer/devcontainer.json) |
| oapi-codegen            | [post_create.sh](../.devcontainer/post_create.sh)       |

## Dockerfiles Used In Production

Update these Dockerfiles as they are used in production

- [Generic Go Dockerfile](../images/go_service.Dockerfile)
- [Generic Node + Nginx Dockerfile](../images/nodejs_service.Dockerfile)

## Dockerfiles Used For Development

Update these Dockerfiles so that the developer experience maintains usage of the latest files

- [DevContainer Node Version](../.devcontainer/Dockerfile)
  - **This image is important to upgrade when upgrading the version of Node used for development**
- [Datastore](../.dev/datastore/Dockerfile)
- [GCS](../.dev/gcs/Dockerfile)
- [Spanner](../.dev/spanner/Dockerfile)
