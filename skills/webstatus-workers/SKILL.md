---
name: webstatus-workers
description: Use when working with the webstatus notification pipeline, event producer, push delivery, or push workers (e.g., Email, Webhooks), and Pub/Sub subscribers.
---

# webstatus-workers

This skill provides architectural and implementation details for the `workers/` directory in `webstatus.dev`, which houses the event-driven notification pipeline.

## Architecture

For a detailed breakdown of the notification pipeline architecture, including the event producer, push delivery dispatcher, and email sender, see [references/architecture.md](references/architecture.md).

For guidance on adding future integrations (like Webhooks), see [references/how-to-add-a-worker.md](references/how-to-add-a-worker.md).

## Canonical Data Transport

Workers must use the shared structs in [lib/workertypes/types.go](../../lib/workertypes/types.go) for all incoming and outgoing messages. This ensures that a change in the Spanner schema or API doesn't immediately break the worker pipeline (decoupling).

## Local Development

- The workers run locally via Skaffold and connect to local emulators for Spanner (`spanner:9010`) and Pub/Sub (`pubsub:8060`).
- The `FRONTEND_BASE_URL` locally is usually `http://localhost:5555`.

## Infrastructure Abstraction (The Adapter Pattern)

All workers must be decoupled from GCP-specific SDKs.

- **Interfaces**: Define the "What" (e.g., `interface EmailSender`) in the worker's package.
- **Implementations**: Put the "How" (e.g., `struct PubSubSender`) in `lib/gcppubsub/gcppubsubadapters`.
- This ensures workers are testable without Pub/Sub or GCS emulators.

## General Guidelines

- **DO** write tests for all new parsers, generators, and differ logic using table-driven tests.
- **DO** use the `generic.OptionallySet[T]` pattern when defining blob structures or canonical in-memory state to handle forward/backward compatibility gracefully (quiet rollouts).
- **DON'T** modify generated code.
- **DO** ensure the worker uses `slog` for logging and bubbles up transient errors using `errors.Join(event.ErrTransientFailure, err)` to NACK the message for retry.

## Testing & Linting

- **Precommit Suite**: Run `make precommit` to execute the full suite of Go tests, formatting, and linting.
- **Linting**: Run `make go-lint` to lint all Go code using `golangci-lint`.
- **Quick Test Iteration**: Because this project uses a multi-module workspace (`go.work`), to run tests quickly for a single package without running the whole suite, execute `go test` from _within_ the specific module directory:
  ```bash
  cd workers/<worker_name> && go test -v ./...
  ```

## Documentation Updates

When you add a new worker, update the notification pipeline, or change data flow:

- Update `docs/ARCHITECTURE.md` to reflect the new pipeline step.
- Trigger the "Updating the Knowledge Base" prompt in `GEMINI.md` to ensure I am aware of the changes.
- Update these very skills files if you introduce new structural patterns or rules.
