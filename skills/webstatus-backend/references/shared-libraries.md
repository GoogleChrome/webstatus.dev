# Shared Go Libraries & Utilities

The `lib/` directory contains shared code used by the `backend` and `workflows`.

## Key Subdirectories

- **`lib/gen`**: Auto-generated code (OpenAPI, JSON Schema, ANTLR). **Never edit manually.**
- **`lib/gcpspanner`**: Spanner client, models, generic mapper helpers.
- **`lib/gcpspanner/spanneradapters`**: Translation layer between external services and the DB client.
- **`lib/generic`**: Utilities like `OptionallySet[T]` for handling optional fields.
- **`lib/blobtypes`**: GCS blob storage definitions.

## Blob Storage & Schema Evolution

When storing persistent state (like saved search notifications in GCS blobs):

- **Canonical vs. Storage Types**: Keep internal business logic (`lib/workertypes/comparables`) separate from persistent types (`lib/blobtypes/v1`).
- **The `OptionallySet` Pattern**: To handle forward and backward compatibility, wrap new struct fields in `generic.OptionallySet[T]`.
  - _Old Blob_: Field missing -> `IsSet: false`. Logic ignores it (Quiet Rollout).
  - _New Data_: Field present -> `IsSet: true`. Logic processes it.
- **DON'T** change the meaning of existing fields in `lib/blobtypes`; create `v2` if a breaking change is needed.

## Utility Scripts

Small CLI tools and helper scripts reside in `util/`:

- `util/run_job.sh`: Runs data ingestion locally via Minikube.
- `util/cmd/load_fake_data/`: Emulators fake data (`make dev_fake_data`).
- `util/cmd/load_test_users/`: Emulators fake users (`make dev_fake_users`).
- **DON'T** put production application logic in `util/`.

## Verify, Don't Assume

Always consult the canonical sources of truth instead of assuming based on general patterns:

- **API Contracts**: `openapi/backend/openapi.yaml`
- **Database Schema**: `infra/storage/spanner/migrations/`
- **Search Grammar**: `antlr/FeatureSearch.g4`
- **External Schemas**: `jsonschema/`
