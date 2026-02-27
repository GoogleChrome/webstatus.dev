---
name: webstatus-ingestion
description: Use when working on Go data ingestion workflows, scheduled Cloud Run jobs, or adding new scrapers for BCD, WPT, or other data sources.
---

# webstatus-ingestion

This skill provides guidance for developing and deploying the scheduled data ingestion workflows (Cloud Run Jobs) in the `workflows/` directory.

## Architecture

- **Location**: Workflows are stand-alone Go applications located in `workflows/steps/services/`.
- **Varied Consumers**: Specialized workflows (BCD, WPT, UMA, Mapping, Dev Signals) are defined in [infra/ingestion/workflows.tf](../../infra/ingestion/workflows.tf).
- **Pattern**: Most workflows follow a "Downloader -> Parser -> Processor" separation of concerns.
- **Trigger**: When ingestion flows complete, they trigger the `event_producer` to begin the notification diffing process.

## Architecture

For a detailed map of data sources, Spanner target tables, and job orchestration patterns, see [references/architecture.md](references/architecture.md).

## Infrastructure Abstraction (The Adapter Pattern)

Ingestion jobs must be decoupled from the core DB logic and the "Backend" API.

- **Consumer Adapters**: Each workflow uses a purpose-built `spanneradapter` (e.g., `BCDConsumerAdapter`).
- **Why**: This prevents ingestion logic from breaking the Public API if the data schema changes. It also allows mocking the database during parser tests.

## Guides

- **[Add a New Scheduled Workflow](references/add-workflow.md)**: Steps for creation, scheduling, and Terraform integration.
- **[Ingestion Patterns](references/ingestion-patterns.md)**: Choosing between Sync, Batch Upsert, and Simple Insert.

## General Do's and Don'ts

- **DO** use consumer-specific `spanneradapters` (e.g. `BCDConsumer`).
- **DON'T** call the `Backend` spanner adapter from a workflow.
- **DO** separate data fetching/parsing from the main workflow processor (use `pkg/data/downloader.go` and `parser.go`).
- **DO** use `web_features_mapping_consumer` when syncing browser-specific implementation keys or "implementer" metadata.
- **DO** use `uma_export` for any changes involving Chromium usage metrics or histograms.
- **DO** use intermediate types in `lib/` (e.g. `lib/webdxfeaturetypes`) to decouple logic from external source schemas.
- **Tip**: While `make dev_workflows` exists to pull live data, it is generally preferred to use `make dev_fake_data` for UI and Backend development to ensure stability.
- **DO** use `manifests/job.yaml` for workflows (scheduled jobs), unlike workers which use `pod.yaml`.

## Testing & Linting

- **Precommit Suite**: Run `make precommit` to execute the full suite of Go tests, formatting, and linting.
- **Linting**: Run `make go-lint` to lint all Go code using `golangci-lint`.
- **Quick Test Iteration**: Because this project uses a multi-module workspace (`go.work`), to run tests quickly for a single package without running the whole suite, execute `go test` from _within_ the specific module directory:
  ```bash
  cd workflows/steps/services/<workflow_name> && go test -v ./...
  ```

## Documentation Updates

When you add a new workflow or change the ingestion patterns:

- Update `docs/ARCHITECTURE.md` to reflect the new external source or data flow.
- Trigger the "Updating the Knowledge Base" prompt in `GEMINI.md` to ensure I am aware of the changes.
- Update these very skills files if you introduce new structural patterns.
