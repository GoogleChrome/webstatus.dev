# Development

## Table Of Contents
1. Requirements
2. Project Structure
3. Running Locally
4. Populate Data Locally
5. Generating Code
6. Clean Up


## Requirements

The recommended way to do development is through the provided devcontainer. To
run a devcontainer, check out this
[web page](https://code.visualstudio.com/docs/devcontainers/containers#_system-requirements)
for the latest requirements to run devcontainer. The devcontainer will have
everything pre-installed.


## Project Structure
- backend/: Go backend code.
- frontend/: TypeScript frontend code.
- workflows/: Data pipelines for fetching and processing data.
- openapi/: OpenAPI specifications for APIs.
- jsonschema/: JSON schemas for data validation.
- antlr: Description of search grammar
- lib/gen: Output of generated code

## Running locally


```sh
# Terminal 1
make start-local
```

This command will build the necessary Docker images, start Minikube
(a local Kubernetes cluster), and deploy the webstatus.dev services to it.

Once everything comes up, open a second terminal.

**Important**: By default, the services running in Minikube are not accessible
from your host machine. To enable access, in a new terminal, run:

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

This will establish port forwarding, allowing you to access the backend at
http://localhost:8080 and the frontend at http://localhost:5555.


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

_In the event the servers are not responsive, make a temporary change to a file_
_in a watched directory (e.g. backend). This will rebuild and expose the_
_services._

## Populate Data Locally

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

An option could be to populate the database with fake data. This is useful if
the live data sources are down or constantly changing.

```sh
# Terminal 2 - Run local workflows
make dev_fake_data
```

#### Verify the database has data

Open `http://localhost:8080/v1/features` to see the features populated
from the latest snapshot from the web-features repo.

## OpenAPI

| Resource      | Location                                                     |
| ------------- | ------------------------------------------------------------ |
| backend       | [openapi/backend/openapi.yaml](openapi/backend/openapi.yaml) |

## Generating Code
### OpenAPI
#### Go and OpenAPI

There two common configurations used to generate code for Go.

- [openapi/server.cfg.yaml](openapi/server.cfg.yaml)
- [openapi/types.cfg.yaml](openapi/types.cfg.yaml)

This repository uses
[deepmap/oapi-codegen](https://github.com/deepmap/oapi-codegen) to generate the
types.

#### TypeScript and OpenAPI

The project use [openapi-typescript](https://github.com/drwpow/openapi-typescript)
to generate types.

#### Generate OpenAPI Code
If changes are made to the openapi definition, run:

```sh
make -B openapi
```

### JSON Schema

The project uses json schema to generate types from:
- [browser-compat-data](https://github.com/mdn/browser-compat-data/tree/main/schemas)
- [web-features](https://github.com/web-platform-dx/web-features/blob/main/schemas/defs.schema.json)
  - We make minor tweaks to the web-features schema to work with the
    [QuickType](https://github.com/glideapps/quicktype) tool to successfully generate the types

We vendor the files schemas locally in the [jsonschema](./jsonschema/) folder.

#### Generating JSON Schema Types

`make jsonschema`

### ANTLR

We use ANTLR v4 to describe the grammar for our search.

You can find the grammar in this [file](./antlr/FeatureSearch.g4)

In the same directory, is the [README](./antlr/FeatureSearch.md) for the grammar.

#### Go & ANTLR

Run `make antlr-gen`

#### TypeScript & ANTLR

TODO

## Clean Up

To clean up the resources, do things in reverse:
- Stop Port Forwarding: `make port-forward-terminate`
- Stop Services (Local Setup): `make stop-local`