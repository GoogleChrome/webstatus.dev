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

E2E tests are split into three decoupled suites in `e2e/`:

- **Synthetic Smoke Probes (`e2e/synthetic/`)**: Fast health check (~15s) ensuring cluster connectivity and routing (`make playwright-synthetic`).
- **Visual Regressions (`e2e/visual/`)**: Dual-theme UI and layout checks against static mock fixtures (`make playwright-visual`).
- **Functional User Journeys (`e2e/functional/`)**: Deep stateful tests against live emulators, sharded 3-way in CI (`make playwright-functional`).

## 3. CI/PR Validation Lifecycle

Every Pull Request undergoes automated validation:

1. **Static Analysis (`build` job)**: `make precommit` runs Go/TS linters, unit tests, and license checks.
2. **Parallel E2E Validation**: Matrix runners execute `synthetic`, `visual`, and 3-way sharded `functional` suites in parallel across Chromium, Firefox, and WebKit.
3. **Report Merging**: `merge-reports` combines blob reports from all runners into a unified HTML dashboard.

## 4. Data Population Strategies

| Command | Tool | Purpose / Flags |
| :--- | :--- | :--- |
| `make dev_fake_users` | [`util/cmd/load_test_users`](../../../util/cmd/load_test_users/main.go) | Seeds predictable test accounts into the Auth emulator. |
| `make dev_fake_data` | [`util/cmd/load_fake_data`](../../../util/cmd/load_fake_data/main.go) | Seeds consistent entities into Spanner.<br>• `-num_features=N` (default `80`)<br>• `-releases_per_browser=N` (default `30`)<br>• `-runs_per_browser_channel=N` (default `35`)<br>• `-reset=true` (reset test user data)<br>• `-scope=all\|user` |
| `make dev_workflows` | [`util/run_job.sh`](../../../util/run_job.sh) | Orchestrates a real ingestion run using live data sources. |
