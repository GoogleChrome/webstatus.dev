# Backend Architecture & Implementation

This document provides a technical deep-dive into the Go Backend API implementation patterns and request flows.

## 1. Request Flow & Middleware

Requests travel from the browser through a secure flow to the internal database.

### Auth & JWT Verification

- **Middleware**: [backend/pkg/httpmiddlewares/auth.go](../../../backend/pkg/httpmiddlewares/auth.go)
- **Mechanism**: Verifies Firebase Auth JWTs.
- **Identity**: Maps the Firebase UID to the internal user profile, which is required for managing Saved Searches and Notification Channels.

### Caching Strategy

- **File**: [cache.go](../../../backend/pkg/httpserver/cache.go)
- **Logic**: Uses Valkey (Memorystore) for public statistics.
- **Bypass**: Authenticated user data (e.g., `/v1/users/me/*`) **always bypasses the cache** to ensure freshness and security.

## 2. Technical Patterns

### Spanner Adapters

The backend relies on the **Adapter Pattern** to decouple business logic from Spanner-specific SQL.

- **Reference**: [lib/gcpspanner/spanneradapters/](../../../lib/gcpspanner/spanneradapters/)
- **Usage**: Handlers call interfaces (Ports) which are satisfied by these adapters.

### Search Grammar Integration

Complex search queries (e.g., `baseline:widely`) are processed via an ANTLR4 parser.

- **Parsing**: The API calls the [FeaturesSearchVisitor](../../../lib/gcpspanner/searchtypes/features_search_visitor.go) to translate search strings into optimized Spanner SQL.
- **Extension**: New search terms must be added to the grammar and then mapped in the visitor logic.

## 3. Infrastructure Connectivity

- **VPC Service Controls**: Protects the database from unauthorized external access.
- **Private Service Connect (PSC)**: Provides a dedicated secure endpoint for the Valkey cache within the private subnet.
