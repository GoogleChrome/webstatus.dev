---
name: webstatus-backend
description: Use when creating or modifying Go backend API endpoints, modifying Spanner database schemas, or working with OpenAPI and Spanner mappers.
---

# webstatus-backend

This skill provides guidance for developing the Go-based backend API for `webstatus.dev`.

## Core Components

- **HTTP Server (`backend/pkg/httpserver`)**: Handles routing and requests via `oapi-codegen` stubs.
- **Storage Adapter (`lib/gcpspanner/spanneradapters`)**: Translates API types to database types.
- **Spanner Client (`lib/gcpspanner`)**: Core logic for Spanner interactions using the Mapper pattern.
- **Valkey Cache (`lib/valkeycache`)**: Isolated via **Private Service Connect (PSC)** for secure internal access.

## Architecture

For a technical deep-dive into the backend implementation patterns, request flows, and auth middleware, see [references/architecture.md](references/architecture.md).

## Guides

- **[Add a New API Endpoint](references/add-api-endpoint.md)**: Mandatory spec-first process.
- **[Spanner Mapper Pattern](references/mapper-pattern.md)**: How to use the generic entity helpers.
- **[Spanner Best Practices](references/spanner-best-practices.md)**: Efficient and safe querying.
- **[Shared Libraries & Utilities](references/shared-libraries.md)**: Guidelines for `lib/`, `util/`, and the `OptionallySet` pattern.

## Architectural Patterns: Abstraction & Adapters

We use a Hexagonal-style **Adapter Pattern** to decouple application logic from infrastructure.

- **Ports**: Interfaces should be defined in the application package (e.g., `pkg/sender`).
- **Adapters**: Implementations live in `lib/` (e.g., `lib/gcpspanner/spanneradapters` or `lib/valkeycache`).
- **Why**: This enables painless unit testing using mock adapters and ensures that swapping external services (e.g., Valey to Redis) doesn't leak into business logic.

## General Do's and Don'ts

- **DO** use `spanneradapters` for DB interactions in the API.
- **DON'T** call `gcpspanner.Client` directly from `httpserver` handlers.
- **DO** use `row.ToStruct(&yourStruct)` instead of manual column scanning.
- **DO** define new Spanner table structs and query logic within `lib/gcpspanner`.
- **DO** update [FeaturesSearchVisitor.go](../../lib/gcpspanner/searchtypes/features_search_visitor.go) when adding new filter terms to the search grammar.
- **DO** use **Canonical Transport Types** from `lib/workertypes` for any data crossing service boundaries (e.g. results sent to Pub/Sub).
- **DO** write integration tests using `testcontainers-go` for any changes to the `lib/gcpspanner` layer.
- **DO** add response caching for new read-only endpoints in `backend/pkg/httpserver/cache.go`.
- **DON'T** import `lib/backendtypes` into `lib/gcpspanner` (prevents circular dependencies).
- **DO** handle business key to internal ID translation inside the `gcpspanner` client.
- **DO** ensure `Merge` functions in mappers copy ALL fields, including `UpdatedAt`.
- **DO** use `...WithTransaction` variants of helpers when inside a `ReadWriteTransaction`.
- **DO** call `eventPublisher.PublishSearchConfigurationChanged` in handlers that modify user saved searches to trigger immediate notification dispatcher updates.

## Testing & Linting

- **Precommit Suite**: Run `make precommit` to execute the full suite of Go tests, formatting, and linting.
- **Linting**: Run `make go-lint` to lint all Go code using `golangci-lint`.
- **Quick Test Iteration**: Because this project uses a multi-module workspace (`go.work`), to run tests quickly for a single package without running the whole suite, execute `go test` from _within_ the specific module directory, or provide the full module path:
  ```bash
  cd backend && go test -v ./pkg/...
  # Or
  go test -v github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/...
  ```
- **Integration Tests**: Any changes to `lib/gcpspanner` **MUST** include integration tests using `testcontainers-go` against the Spanner emulator.

## Documentation Updates

When making significant architectural changes, adding new major endpoints, or altering the database schema:

- Trigger the "Updating the Knowledge Base" prompt in `GEMINI.md` to ensure I am aware of the changes.
- Update `docs/ARCHITECTURE.md` if the system boundaries change.
- Update these very skills files if you introduce new established patterns.
