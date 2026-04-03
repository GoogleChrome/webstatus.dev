# Go Spanner Query Best Practices

## Database Migrations

- **Creation**: Run `make spanner_new_migration` to create a new migration file in [`infra/storage/spanner/migrations/`](../../../infra/storage/spanner/migrations/).
- **Data Migrations**: For complex data migrations (e.g. renaming a feature key), use the generic migrator in [`lib/gcpspanner/spanneradapters/migration.go`](../../../lib/gcpspanner/spanneradapters/migration.go).

## Scanning Rows

- **DO** use `row.ToStruct(&yourStruct)` to scan results. It leverages struct tags (e.g., `spanner:"ColumnName"`) and is less error-prone.
- **DON'T** manually scan columns using `r.ColumnByName`. This is verbose and fragile to schema changes.

## Foreign Keys

- **Cascade Deletes**: Use `ON DELETE CASCADE` for relationships.
- **Batched Deletes**: If a cascade would delete thousands of rows, implement `GetChildDeleteKeyMutations` in the parent mapper to delete children in batches first.

## Mappers vs Clients

- **DO** look for existing mappers in [`lib/gcpspanner/`](../../../lib/gcpspanner/) (e.g. `webFeatureSpannerMapper`) before creating new ones.
- **DO** translate business keys to internal IDs inside the `gcpspanner` client so that adapters/workflows remain unaware of internal DB IDs.

## Sorting & Order Strategies

When implementing integer-based explicit ordering columns, choose an approach based on the growth pattern of the data:

- **Chronological / Infinite Growth Lists (e.g. Years, Global Baselines)**
  - Use `ORDER BY Column DESC` (Highest is Top).
  - Start seeding at high values (e.g. 10,000) and step downwards.
  - **Why**: This prevents integer exhaustion at `0`. When new, more recent items launch, they naturally increment (e.g., 11,000) and take the top spot securely.
  
- **Curated / Bounded Lists (e.g. Top Issues, Fixed Categorizations)**
  - Use `ORDER BY Column ASC` (Lowest is Top).
  - Start at `10` and increment by `10`s (10, 20, 30).
  - **Why**: If you need to reorder or inject an item between two priorities without shifting the entire list, you can safely use intermediate values (e.g., `15`).
