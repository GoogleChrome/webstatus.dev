# Data Ingestion Patterns

When writing a new Spanner mapper for an ingestion workflow, choose the correct generic database helper based on the nature of the data.

## 1. Full Synchronization (`newEntitySynchronizer`)

- **Use when**: The incoming data is a complete source of truth, and you need to handle creates, updates, and **deletes** to keep a table perfectly in sync with the external source.
- **Mapper Interface Required**: `syncableEntityMapper`
- **Example**: Syncing the `WebFeatures` table from the `web-features` git repository. If a feature is removed from the git repo, it must be removed from the database.

## 2. Batch Upsert (`newEntityWriter`)

- **Use when**: You are adding or updating records in bulk but **not** deleting old records. Common for append-only or time-series data. Usually done in a loop or with a custom batching function.
- **Mapper Interface Required**: `writeableEntityMapper`
- **Example**: Storing daily UMA metrics or WPT results for a specific run.

## 3. Simple Insert (`newEntityWriter`)

- **Use when**: You are processing and inserting records one-by-one inside a loop.
- **Mapper Interface Required**: `writeableEntityMapper`
- **Example**: Ingesting the list of BCD browser releases as they are processed sequentially.
