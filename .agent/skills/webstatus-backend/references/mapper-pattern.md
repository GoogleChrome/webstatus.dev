# The Go Mapper Pattern for Spanner

Used for all interactions with the Spanner database to reduce boilerplate and ensure consistency. Defined in `lib/gcpspanner/client.go`.

## Core Concept

Use generic helpers configured with a "mapper" struct instead of custom query logic.

### Generic Helpers

- `newEntityReader`
- `newEntityWriter`
- `newEntitySynchronizer`

### Mapper Interfaces

- `baseMapper`: Defines the Spanner table name.
- `readOneMapper`: Defines selection by key.
- `mergeMapper`: Defines how to merge updates.
- `deleteByStructMapper`: Defines deletion.
- `childDeleteMapper`: Handles child deletions in batches (via `GetChildDeleteKeyMutations`) to stay under Spanner's mutation limit.

## Transactional Usage

- **DO** use the `...WithTransaction` variants (e.g., `createWithTransaction`, `updateWithTransaction`) inside a `ReadWriteTransaction`.
- **DON'T** use standard helpers inside a transaction; they will attempt to start a new transaction and fail.
- **DON'T** use `spanner.InsertStruct` manually.

## Implementation Guardrails

- **Merge Logic**: Ensure `Merge` or `mergeAndCheckChangedMapper` copies **all** fields. A missing field assignment will cause silent update failures.
- **UpdatedAt**: Always set `UpdatedAt` to `spanner.CommitTimestamp` in the input struct before merging.
