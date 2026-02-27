# E2E Testing & CI Architecture

This document provides a technical guide for the local development environment, E2E testing infrastructure, and CI/PR validation lifecycle.

## 1. Local Development Environment

We use **Skaffold** and **Minikube** to orchestrate a production-like environment on developer machines.

### Service Orchestration

- **Cluster**: Minikube manages the local Kubernetes environment.
- **Sync**: Skaffold watches for code changes and performs live-reloads of the Go backend and Lit frontend services into the cluster.

### GCP Emulators & Mocks

To facilitate local testing without cloud dependencies:

- **Spanner**: Uses the Spanner Emulator for database operations.
- **Auth**: Uses the Firebase Auth Emulator for user login and JWT generation.
- **Wiremock**: Mocks external APIs like GitHub and Chime to simulate real-world service responses.

## 2. E2E Testing with Playwright

E2E tests ensure that the entire system (Frontend -> Backend -> Database) functions correctly together.

- **Source**: [e2e/tests/](../../../e2e/tests/)
- **Determinism**: Tests rely on a seeded state created via the `make dev_fake_data` and `make dev_fake_users` commands.
- **Execution**: Run locally using `make playwright-test` or via the VS Code Playwright extension.

## 3. CI/PR Validation Lifecycle

Every Pull Request undergoes rigorous automated validation before it is merged into `main`.

1.  **Static Analysis**: `make precommit` runs Go and TS linters, unit tests, and license header checks.
2.  **E2E Validation**: A full E2E suite is executed against the PR code using a containerized Skaffold environment.
3.  **Approval**: Both `precommit` and `playwright-test` gates must pass (green checkmarks) for a PR to be eligible for merge.

## 4. Data Population Strategies

| Command               | Tool                                                                    | Purpose                                                      |
| :-------------------- | :---------------------------------------------------------------------- | :----------------------------------------------------------- |
| `make dev_fake_users` | [`util/cmd/load_test_users`](../../../util/cmd/load_test_users/main.go) | Seeds predictable test accounts into the Auth emulator.      |
| `make dev_fake_data`  | [`util/cmd/load_fake_data`](../../../util/cmd/load_fake_data/main.go)   | Seeds consistent entities (Features, Searches) into Spanner. |
| `make dev_workflows`  | [`util/run_job.sh`](../../../util/run_job.sh)                           | Orchestrates a real ingestion run using live data sources.   |
