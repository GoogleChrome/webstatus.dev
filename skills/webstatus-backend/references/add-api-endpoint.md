# How to Add a New Backend API Endpoint

This project follows a **mandatory "spec-first"** process. You must define the API contract before writing any Go code to avoid compilation errors from missing types.

## Step-by-Step Process

### 1. Define the Contract (`openapi/backend/openapi.yaml`)

- Open `openapi/backend/openapi.yaml`.
- Define the new endpoint, parameters, and request/response schemas.
- If using new data structures, define them in `components.schemas`.
- Ensure you set a clear `operationId`.

### 2. Generate Types

- Run `make openapi`.
- This reads the YAML and generates Go server stubs, request/response structs, and TypeScript client code in `lib/gen/backend/`.
- **CRITICAL:** Do not proceed until this command completes successfully.

### 3. Implement the HTTP Handler (`backend/pkg/httpserver/`)

- Add a new method to the `Server` struct. The method name must match the `operationId`.
- This handler parses the request, calls the adapter layer, and writes the response.

### 4. Implement the Adapter Method (`lib/gcpspanner/spanneradapters/`)

- Add a method to the `Backend` struct.
- Translate generated API types (from `lib/gen/backend/`) to internal Spanner types.
- Call the underlying Spanner client.

### 5. Implement the Spanner Client Method (`lib/gcpspanner/`)

- Add a method to the `Client` struct.
- Contains the actual database logic (queries, writes, transactions), ideally using the mapper pattern.

### 6. Add Tests

- **Unit Test**: In `lib/gcpspanner/spanneradapters/backend_test.go` using mocks.
- **Integration Test**: In `lib/gcpspanner/*_test.go` using `testcontainers-go` against a Spanner emulator.
