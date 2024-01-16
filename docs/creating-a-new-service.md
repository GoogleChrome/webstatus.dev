# Creating A New Service

This document outlines information to add a new service. In this context, a
"service" is a HTTP server. This is useful for standalone APIs or custom code
to be used in GCP Workflows.

## Code Organization

Workflow Services are located in the `workflows/steps/services` [folder](../workflows/steps/services/).

External Services (e.g. Public APIs, external websites) are located in the root directory. Examples:

- [backend](../backend/)
- [frontend](../frontend/)

- [ ] Create a folder in appropriate location depending on the type of service

## OpenAPI

If any process will be making REST API calls to it, there should be an OpenAPI
for it. [Docs](https://swagger.io/docs/specification/about/). This will allow
for auto-generation of code for any language.

- [ ] Create a openapi.yaml file in the root of the new folder you created.
- [ ] Add the necessary route(s) to the file.
- [ ] Add a command to the openapi target in [Makefile](../Makefile) to generate
      objects and/or handlers for your project.

## Build Your App

- [ ] Code the app

## Prepare It For Skaffold & Local Kubernetes

[Skaffold](https://skaffold.dev/) is configured to deploy to a local Kubernetes
cluster. As a result, developers can edit the code and skaffold will
automatically rebuild and deploy the new version. Each version is a built Docker
image. The following sub-sections describe how to configure the app to use this

### Dockerize The App

Docker is the core of the developer ecosystem in this project. The same docker
image that is run locally will be used in the deployed version.

Currently, there are 2 generic docker images

- [Go](../images/go_service.Dockerfile)
  - Is a multi stage build that builds a minimal docker image.
  - Users of the image can specify the correct `service_dir` build argument to
    point to the root of the newly created directory.
- [Node](../images/nodejs_service.Dockerfile)

  - Has two targets that can be used in the final image:
    - `production` - for running a node based program
    - `static` - for running a static website that is built by node

- [ ] Plan to use one of the already created Dockerfiles or create a new one.

### Kubernetes Manifests

Kubernetes deploys resources that are described via YAML. Majority of the cases
will only need two types of manifest resources. A Pod resource and a service
resource. A [Pod](https://kubernetes.io/docs/concepts/workloads/pods/) resource
deploys a docker image. A
[Service](https://kubernetes.io/docs/concepts/services-networking/service/)
resource exposes that Pod. It provides a name that other Pods can use for
inter-Pod communincation.

You will need to create those resource files. Feel free to refer to fill in the
examples below.

Assume the following variables:

- APP_NAME
  - Lower kebab case name of the app
- INTERNAL_PORT
  - The port number of the app running inside the container. This port is not
    exposed to the host so it can be a common port that other apps use.
- ENV_VARS
  - List of items. Each item looks like:
  - ```
    - name: PROJECT_ID
      value: local
    ```

<details>
  <summary>pod.yaml (click to expand)</summary>

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: $APP_NAME
  labels:
    app.kubernetes.io/name: $APP_NAME
spec:
  containers:
    - name: $APP_NAME
      image: $APP_NAME
      ports:
        - containerPort: $INTERNAL_PORT
          name: http
      env:
      # $ENV_VARS
      resources:
        limits:
          cpu: 250m
          memory: 512Mi
```

</details>

<details>
  <summary>service.yaml (click to expand)</summary>

```yaml
apiVersion: v1
kind: Service
metadata:
  name: $APP_NAME
spec:
  selector:
    app.kubernetes.io/name: $APP_NAME
  ports:
    - protocol: TCP
      port: $INTERNAL_PORT
      targetPort: http
```

</details>

- [ ] Create a `manifests` sub-directory
- [ ] Create pod and service yamls in that `manifests` sub-directory

### Skaffold

Locally, skaffold deploys everything to a local minikube server.

Assume the following variables:

- APP_NAME
  - Lower kebab case name of the app
- REQUIRED_APPS
  - Optional list of required apps. Each item in the list points to a
    relative directory containing another skaffold.yaml file. Example:
  - ```yaml
    - path: ../.dev/datastore # This app requires datastore to be up
    ```
- RUNTIME_TYPE
  - One of the supported languages according to the skaffold
    [docs](https://skaffold.dev/docs/workflows/debug/). By telling skaffold the
    language, it can automatically add debug hooks. If the language is not
    supported, That is fine.
- EXTERNAL_PORT
  - The port number to expose locally on the computer. This needs to be unique
    amoung all services so that it does not collide
- ENV_VARS
  - List of items. Each item looks like:
  - ```
    - name: PROJECT_ID
      value: local
    ```

<details>
  <summary>skaffold.yaml (click to expand)</summary>

```yaml
apiVersion: skaffold/v4beta9
kind: Config
metadata:
  name: $APP_NAME-config
requires:
  # $REQUIRED_APPS
profiles:
  - name: local
    build:
      artifacts:
        - image: $APP_NAME
          context: ..
          runtimeType: $RUNTIME_TYPE # If not applicable, remove the whole line.
          docker:
            dockerfile: images/go_service.Dockerfile
            buildArgs:
              service_dir: $APP_NAME
      local: {}
    manifests:
      rawYaml:
        - manifests/*
    deploy:
      kubectl: {}
    # If you the host machine does not need access, you can skip the section below
    portForward:
      - resourceType: pod
        resourceName: $APP_NAME
        port: $EXTERNAL_PORT
```

</details>

- [ ] Create a new skaffold.yaml in the root of the new folder.
- [ ] Fill in the skaffold.yaml appropriately
- [ ] If it is a public service, add it to the requirements in the root
      [skaffold.yaml](../skaffold.yaml). If it is a workflow service, add it to
      the requirements in the
      [workflow skaffold.yaml](../workflows/skaffold.yaml).
- [ ] Make sure the container comes up successfully when running both the
      `make start-local` and `make debug-local` commands.



## Prepare It For GCP Deployment

Terraform allows the team to describe the desired state of the infrastructure
as code. When a deployment happens, terraform will automatically deploy your
service for you. Terraform also comes with added benefits such as the ability to
[deploy your own stack](deploy-own-stack-to-gcp.md). This section details how to
write the terraform for your service.

### Terraform For A Public service

TODO

### Terraform For A Workflow

TODO

## Example Set of PRs To Introduce This

TODO
