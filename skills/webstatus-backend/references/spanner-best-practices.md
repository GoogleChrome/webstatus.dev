# Go Spanner Query Best Practices

## Database Migrations

- **Creation**: Run `make spanner_new_migration` to create a new migration file in `infra/storage/spanner/migrations/`.
- **Data Migrations**: For complex data migrations (e.g. renaming a feature key), use the generic migrator in `lib/gcpspanner/spanneradapters/migration.go`.

## Scanning Rows

- **DO** use `row.ToStruct(&yourStruct)` to scan results. It leverages struct tags (e.g., `spanner:"ColumnName"`) and is less error-prone.
- **DON'T** manually scan columns using `r.ColumnByName`. This is verbose and fragile to schema changes.

## Foreign Keys

- **Cascade Deletes**: Use `ON DELETE CASCADE` for relationships.
- **Batched Deletes**: If a cascade would delete thousands of rows, implement `GetChildDeleteKeyMutations` in the parent mapper to delete children in batches first.

## Mappers vs Clients

- **DO** look for existing mappers in `lib/gcpspanner/` (e.g. `webFeatureSpannerMapper`) before creating new ones.
- **DO** translate business keys to internal IDs inside the `gcpspanner` client so that adapters/workflows remain unaware of internal DB IDs.
