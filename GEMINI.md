# Gemini Code Assist Configuration for webstatus.dev

This document provides context to Gemini Code Assist to help it generate more accurate and project-specific code suggestions.

## 1. Project Overview

`webstatus.dev` tracks the status and implementation of web platform features across major browsers. It aggregates data from various sources, including Web Platform Tests (WPT), Browser-Compat-Data (BCD), and Chromium's UMA (Usage Metrics). The architecture consists of a Go API (running as a Cloud Run service), multiple Go jobs (running as Cloud Run jobs), and a TypeScript frontend (running as a Cloud Run service). It utilizes Spanner for the database and Valkey as a cache layer.

## 2. Local Development Workflow

This section describes the tools and commands for local development.

- **Devcontainer**: The project uses a devcontainer for a consistent development environment.
- **Skaffold & Minikube**: Local development is managed by `skaffold`, which deploys services to a local `minikube` Kubernetes cluster.
- **Makefile**: Common development tasks are scripted in the `Makefile`. See below for key commands.

### Key Makefile Commands

- **`make start-local`**: Starts the complete local development environment using Skaffold and Minikube. This includes live-reloading for code changes.
- **`make port-forward-manual`**: After starting the environment, run this to expose the services (frontend, backend, etc.) on `localhost`.
- **`make test`**: Runs the Go and TypeScript unit tests. Use `make go-test` to run only Go tests.
- **`make precommit`**: Runs a comprehensive suite of checks including tests, linting, and license header verification. This is the main command to run before submitting a pull request.
- **`make gen`**: Regenerates all auto-generated code (from OpenAPI, JSON Schema, ANTLR). Use `make openapi` for just OpenAPI changes.
- **`make dev_workflows`**: Populates the local Spanner database by running the data ingestion jobs against live data sources.
- **`make dev_fake_data`**: Populates the local Spanner database with a consistent set of fake data for testing.
- **`make spanner_new_migration`**: Creates a new Spanner database migration file in `infra/storage/spanner/migrations`.

## 3. Codebase Architecture

This section details the main components of the application and how they interact.

### 3.1. Application Components

These are the primary, runnable parts of the application, corresponding to top-level directories.

#### Backend (`backend/`)

The Go-based backend application that provides the primary REST API.

- **Architecture**:
  1.  **HTTP Server (`backend/pkg/httpserver`)**: Handles incoming HTTP requests, routing, and response serialization. Uses `oapi-codegen` to generate server stubs from the OpenAPI specification.
  2.  **Storage Adapter (`lib/gcpspanner/spanneradapters`)**: A crucial abstraction layer that sits between the HTTP server and the Spanner database client. It translates data between API-level types and database types.
  3.  **Database Client (`lib/gcpspanner`)**: Contains the core logic for interacting with Google Cloud Spanner, including data models and generic helpers.
- **Coding Conventions & Patterns**:
  - **Error Handling**: Use `errors.Is` for checking specific error types and custom error types defined in `lib/gcpspanner/spanneradapters/backendtypes` and `lib/gcpspanner`.
  - **Testing**: Use table-driven unit tests with mocks for dependencies. Use `testcontainers-go` for integration tests against a real Spanner emulator.
  - **Dependency Injection**: The `httpserver.Server` struct receives its dependencies as interfaces during initialization to promote loose coupling.
- **"Do's and Don'ts" (Backend)**:
  - **DO** use the `spanneradapters` layer for database interactions in the API.
  - **DON'T** call the `gcpspanner.Client` directly from the `httpserver` handlers.
  - **DO** add response caching for new read-only API endpoints in `backend/pkg/httpserver/cache.go`.
  - **DO** define new Spanner table structs and query logic within the `lib/gcpspanner` package.
  - **DO** use the generic entity helpers in `lib/gcpspanner/client.go` (like `entityWriter`) to minimize boilerplate.

#### Frontend (`frontend/`)

The frontend Single Page Application (SPA).

- **Architecture & Technology**:
  - **Framework**: Built with **TypeScript**, **Lit**, and **Web Components**. It also utilizes the **Shoelace** component library.
  - **API Interaction**: Communicates with the Go backend using TypeScript types generated from the OpenAPI specification (`make node-openapi`).
  - **State Management**: Uses **Lit Context** for dependency injection and state management via a service container pattern (`<webstatus-services-container>`).
  - **Runtime**: The production build consists of static assets served by an **Nginx** container.
- **"Do's and Don'ts" (Frontend)**:
  - **DO** create new UI elements as custom elements extending Lit's `LitElement`.
  - **DO** leverage Shoelace components for common UI patterns.
  - **DON'T** introduce other UI frameworks like React or Vue.
  - **DO** use the Lit Context and service container pattern to access shared services.
  - **DON'T** create new global state management solutions.
  - **DO** add Playwright tests for new user-facing features and unit tests for component logic.

#### Workflows / Data Ingestion Jobs (`workflows/`)

Standalone Go applications that populate the Spanner database from external sources.

- **Overview**: Each workflow corresponds to a specific data source (e.g., `bcd_consumer`, `wpt_consumer`). They run as scheduled Cloud Run Jobs in production.
- **Development & Execution**:
  - **Local**: Run all main workflows with `make dev_workflows`.
  - **Production**: Deployed as `google_cloud_run_v2_job` resources via Terraform.
- **"Do's and Don'ts" (Workflows)**:
  - **DO** follow the existing pattern for new workflows: new directory, `main.go`, and `manifests/job.yaml`.
  - **DO** use consumer-specific `spanneradapters` (e.g., `BCDConsumer`).
  - **DON'T** call the `Backend` spanner adapter from a workflow.
  - **DO** use the `entitySynchronizer` for bulk data updates.
  - **DO** add a new target to the `make dev_workflows` command in `Makefile` for any new workflow.

### 3.2. Shared Go Libraries (`lib/`)

Shared Go libraries used by the `backend` and `workflows`.

- **Key Subdirectories**:
  - **`lib/gen`**: Contains all auto-generated code. **Never edit manually.**
  - **`lib/gcpspanner`**: The data access layer, containing the Spanner client, data models, and generic helpers.
  - **`lib/gcpspanner/spanneradapters`**: The abstraction layer between services and the database client.
  - **`lib/cachetypes`**: Common interfaces and types for the caching layer.
- **"Do's and Don'ts" (Libraries)**:
  - **DO** place reusable Go code shared between services here.
  - **DON'T** put service-specific logic in `lib/`.
  - **DO** define new database table structs in `lib/gcpspanner`.
  - **DO** create or extend adapters in `lib/gcpspanner/spanneradapters` to expose new database queries.

### 3.3. End-to-End Data Flow Example

This example illustrates how data is ingested by a workflow and then served by the API, highlighting the different components involved.

**1. Data Ingestion (`web_feature_consumer` workflow)**

The goal is to ingest feature definitions from the `web-platform-dx/web-features` repository into the Spanner `WebFeatures` table.

- A developer runs `make dev_workflows`, which executes the `web_feature_consumer` job via `util/run_job.sh`.
- The `web_feature_consumer` fetches the latest feature data.
- It uses its dedicated adapter, `spanneradapters.WebFeatureConsumer`, to process and store the data.
- The adapter calls `gcpspanner.Client`, which uses the generic `entitySynchronizer` to efficiently batch-write the feature data into the `WebFeatures` table in the database.

**2. Data Serving (`getFeature` API endpoint)**

The goal is to retrieve a specific feature's data and serve it via the REST API.

- A user's browser sends a `GET /v1/features/{feature_id}` request to the backend service.
- The request is handled by `httpserver.GetFeature`.
- The handler first checks the `operationResponseCache` for a valid cached response.
- If no cached response is found, the handler calls the API's data adapter: `spanneradapters.Backend.GetFeature(feature_id)`.
- The `Backend` adapter calls the underlying `gcpspanner.Client.GetFeature(feature_id)` method, which queries the `WebFeatures` table (and others) for the requested feature.
- The data is returned up the chain. The `Backend` adapter transforms the database models into API models.
- The `httpserver` handler caches the successful response and sends it back to the user as JSON.

## 4. Specifications & Generated Code

This section covers the specifications that define contracts and data structures, from which code is generated.

### 4.1. API Specification (`openapi/`)

The `openapi/backend/openapi.yaml` file is the single source of truth for the backend API contract.

- **Code Generation**: `oapi-codegen` (Go) and `openapi-typescript` (TypeScript) are used to generate code from the spec. Run `make openapi` to regenerate.
- **"Do's and Don'ts"**:
  - **DO** edit `openapi.yaml` to make API changes.
  - **DON'T** edit generated files in `lib/gen/`.

### 4.2. Search Grammar (`antlr/`)

The grammar for the feature search query (`q` parameter) is defined using ANTLR v4.

- **Grammar File**: The source of truth for the search syntax is `antlr/FeatureSearch.g4`.
- **Code Generation**: The Go parser code is generated from the grammar file. To regenerate the parser, run `make antlr-gen`.
- **"Do's and Don'ts"**:
  - **DO** modify `antlr/FeatureSearch.g4` to add new keywords, terms, or syntax rules.
  - **DON'T** edit the generated parser files in `lib/gen/featuresearch/parser/` directly.
  - **DO** update the visitor in `lib/gcpspanner/searchtypes/features_search_visitor.go` to handle any new grammar rules.
  - **DO** add parsing tests in `lib/gcpspanner/searchtypes/features_search_parse_test.go` to validate new syntax.

### 4.3. JSON Schema (`jsonschema/`)

The project uses JSON schemas to define the structure of data from external sources like `web-features`.

- **Schema Files**: Schemas are vendored in the `jsonschema/` directory.
- **Code Generation**: The project uses `quicktype` to generate Go types from these schemas. To regenerate the types, run `make jsonschema`.
- **"Do's and Don'ts"**:
  - **DO** update the local schema file if the upstream schema changes.
  - **DON'T** edit the generated Go types in `lib/gen/jsonschema/` directly.
  - **DO** run `make jsonschema` after updating a schema file.

## 5. Core Processes & Architectural Principles

This section covers key processes and architectural decisions that apply across the project.

### 5.1. Testing

- **End-to-End (E2E) Tests (`e2e/`)**: E2E tests are written in **TypeScript** using **Playwright** to test complete user flows.
  - **Execution**: Run with `make playwright-test`. This command sets up a fresh, clean environment.
  - **DO** add E2E tests for critical user journeys.
  - **DON'T** write E2E tests for small component-level interactions.
  - **DO** use resilient selectors like `data-testid`.
- **Unit Tests**:
  - **Go**: Use table-driven unit tests with mocks for dependencies.
  - **TypeScript**: Use `npm run test -w frontend`.

### 5.2. CI/CD (`.github/`)

Continuous integration is handled by GitHub Actions, defined in `.github/workflows/ci.yml`.

- **Key Checks**: The `make precommit` command runs linting and unit tests. `make playwright-test` runs E2E tests.
- **"Do's and Don'ts"**:
  - **DO** run `make precommit` locally before pushing changes to avoid CI failures.
  - **DON'T** merge pull requests if CI checks are failing.

### 5.3. Infrastructure & Deployment (`infra/`)

The project's infrastructure is managed with **Terraform**.

- **Environments**: The project uses a multi-environment setup (e.g., `staging`, `prod`), configured via `.tfvars` files.
- **Deployment**: Handled via `terraform plan` and `terraform apply`. See `DEPLOYMENT.md` for details.
- **Key GCP Resources**: Cloud Run, Cloud Spanner, Identity Platform, Artifact Registry, Valkey/Memorystore.
- **"Do's and Don'ts"**:
  - **DO** use `.tfvars` files for environment-specific configurations.
  - **DON'T** hardcode environment-specific values in `.tf` files.
  - **DO** run `make tf-lint` before committing.

### 5.4. Database Migrations & Foreign Keys

- **Creation**: Use `make spanner_new_migration` to create a new migration file.
- **Cascade Deletes**: Prefer using `ON DELETE CASCADE` for foreign key relationships to maintain data integrity. Add an integration test to verify this behavior (see `lib/gcpspanner/web_features_fk_test.go`).
- **Cascade Caveat**: If a cascade could delete thousands of child entities, it may exceed Spanner's mutation limit. In such cases, implement the `GetChildDeleteKeyMutations` method in the parent's `spannerMapper` to handle child deletions in batches before deleting the parent.

### 5.5. Caching

The `httpserver` package includes a generic response caching mechanism (`operationResponseCache`).

- **Be careful when caching authenticated user data.** To prevent data leakage between users, ensure the cache key includes a unique user identifier. If not possible, avoid caching that endpoint for authenticated requests.

### 5.6. Utility Scripts (`util/`)

Helper scripts and small CLI tools for local development.

- **Key Utilities**:
  - `util/run_job.sh`: Runs a data ingestion job locally in Minikube.
  - `util/cmd/load_fake_data/`: Populates emulators with fake data (`make dev_fake_data`).
  - `util/cmd/load_test_users/`: Populates the auth emulator with test users (`make dev_fake_users`).
- **"Do's and Don'ts"**:
  - **DO** place new one-off development scripts here.
  - **DON'T** put production application logic in `util/`.

## 6. How-To Guides

This section provides step-by-step guides for common development tasks. When working on a specific part of the application, use the corresponding section in this document as your primary guide. For example:

- **Adding a backend API endpoint**: Start with the "API Specification (`openapi/`)" section, then implement the logic following the patterns in the "Backend (`backend/`)" section.
- **Adding a new frontend component**: Follow the patterns in the "Frontend (`frontend/`)" section.
- **Fixing a data ingestion bug**: Refer to the "Workflows (`workflows/`)" section.

### 6.1. How-To: Add a New Search Term

This guide outlines the process for adding a new search term (e.g., `is:discouraged`) to the feature search functionality.

1.  **Update Grammar (`antlr/FeatureSearch.g4`)**:
    - Add the new term to the `search_criteria` rule in the grammar file. For example, add `| discouraged_term`.
    - Define the new `discouraged_term` rule, e.g., `discouraged_term: 'is' ':' 'discouraged';`.

2.  **Regenerate Parser**:
    - Run `make antlr-gen`. This will update the files in `lib/gen/featuresearch/parser/`.

3.  **Update Visitor (`lib/gcpspanner/searchtypes/`)**:
    - Add a new `SearchIdentifier` for your term in `searchtypes.go` (e.g., `IdentifierIsDiscouraged`).
    - In `features_search_visitor.go`, implement the `VisitDiscouraged_termContext` method. This method should create and return a `SearchNode` with the new identifier.

4.  **Update Query Builder (`lib/gcpspanner/feature_search_query.go`)**:
    - In `FeatureSearchFilterBuilder.traverseAndGenerateFilters`, add a `case` for your new `SearchIdentifier`.
    - This case should generate the appropriate Spanner SQL `WHERE` clause for the filter. For example, it might check if a feature key exists in the `FeatureDiscouragedDetails` table.

5.  **Add Tests**:
    - Add a test case to `lib/gcpspanner/searchtypes/features_search_parse_test.go` to verify the parser and visitor correctly handle the new syntax.
    - Add a test case to `lib/gcpspanner/feature_search_query_test.go` to verify the correct SQL is generated.
    - Add an integration test in `lib/gcpspanner/feature_search_test.go` to test the full search flow with the new term.

6.  **Update Frontend (`frontend/`)**:
    - Add the new search term to the search builder UI to make it discoverable to users. This involves updating the search vocabulary in `frontend/src/static/js/utils/constants.ts`.

### 6.2. How-To: Update Toolchain Versions

This guide outlines the process for updating the versions of various tools used in the project. Most tool versions are managed within `.devcontainer/devcontainer.json`.

#### Updating Go

The Go version is managed in two places: the production Docker image and the devcontainer configuration.

1.  **Update Production Image**: In `images/go_service.Dockerfile`, update the `FROM golang:X.Y.Z-alpine...` line to the desired version.
2.  **Update Devcontainer**: In `.devcontainer/devcontainer.json`, find the `ghcr.io/devcontainers/features/go:1` feature and update the `version` property to match the version from the Dockerfile.
3.  **Rebuild Devcontainer**: Rebuild and reopen the project in the devcontainer to apply the new Go version.
4.  **Update Dependencies**: Run `make go-update && make go-tidy` to update Go modules and ensure they are compatible with the new toolchain.

#### Updating Node.js

Similar to Go, the Node.js version is managed in the production Docker image and the devcontainer.

1.  **Update Production Image**: In `images/nodejs_service.Dockerfile`, update the `FROM node:X.Y.Z-alpine...` line to the desired version.
2.  **Update Devcontainer**: In `.devcontainer/devcontainer.json`, find the `ghcr.io/devcontainers/features/node:1` feature and update the `version` property to match.
3.  **Rebuild Devcontainer**: Rebuild and reopen the project in the devcontainer.
4.  **Update Dependencies**: Run `make node-update` to update npm packages.

#### Updating Other Devcontainer Tools (Terraform, Skaffold, etc.)

For other tools defined as features in the devcontainer:

1.  **Update Devcontainer**: In `.devcontainer/devcontainer.json`, find the feature for the tool you want to update (e.g., `ghcr.io/devcontainers/features/terraform:1`) and change its `version`.
2.  **Rebuild Devcontainer**: Rebuild and reopen the project in the devcontainer to use the new version.
