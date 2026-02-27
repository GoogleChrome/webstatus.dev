# Worker Architecture & Implementation

This document provides a comprehensive technical guide for the `workers/` directory, covering both high-level roles and code-level choreography.

## 1. Pipeline Stages & Roles

The notification pipeline consists of three main stages, strictly categorized as **Push** workers that send notifications out based on subscriptions.

### A. Event Producer (`workers/event_producer/`)

- **Role**: The "Brain". It calculates the differences (Added, Removed, Moved, Baseline changed, etc.) between the current Spanner state and the last GCS snapshot.
- **Outcome**: Serializes a State/Diff Blob and publishes a `PublishEventRequest` to the notification topic.
- **Key Files**: [`lib/gcpspanner/browser_feature_support_event.go`](../../../lib/gcpspanner/browser_feature_support_event.go) (Diff logic).

### B. Push Delivery (`workers/push_delivery/`)

- **Role**: The "Dispatcher" or "Fan-out" engine. It queries Spanner (`SubscriptionFinder`) to find all users subscribed to the search triggered by the event.
- **Filtering**: Compares the event's changes against each user's `JobTrigger` list using a `SummaryVisitor`.
- **Outcome**: Publishes channel-specific jobs (e.g., `EmailDeliveryJob`) to downstream workers.
- **Key Files**: [`workers/push_delivery/pkg/dispatcher/`](../../../workers/push_delivery/pkg/dispatcher/) (Fan-out logic).

### C. Delivery Workers (e.g., `workers/email/`)

- **Role**: The "Hands". They format the diff summary appropriately (e.g., into an HTML email) and perform the final delivery.
- **Outcome**: Delivers the notification and updates the `ChannelStateManager` in Spanner with the result.
- **Key Files**: [`lib/email/`](../../../lib/email/) (Templates & Mappers).

### D. On-Demand Workers (e.g., RSS, API Feeds)

- **Role**: Pull-based delivery. These are **subscription-bound** (tied to a user's Saved Search) but bypass the dispatcher's push mechanism. These are not implemented yet.
- **Outcome**: Data is pre-computed or served dynamically via the **API layer** when requested by the client.
- **Flow**: Event Producer publishes `batch-update` -> API/Worker updates cache/feed -> Client pulls via subscription endpoint.

## 2. Implementation Patterns

### Interface & Adapter Pattern

Workers are decoupled from GCP SDKs to facilitate unit testing.

- **Ports**: Defined in the worker's `pkg/` (e.g., `interface SubscriptionFinder`).
- **Adapters**: Found in [`lib/gcpspanner/spanneradapters/`](../../../lib/gcpspanner/spanneradapters/). Live adapters satisfy the interfaces in production.

### Shared Workertypes

All workers MUST use the shared structs in [lib/workertypes/types.go](../../../lib/workertypes/types.go).

- **`PublishEventRequest`**: The canonical message published by the Event Producer.
- **`DeliveryJob`**: The generic job type dispatched to individual delivery workers.

## 3. Schema Evolution & The SummaryVisitor

Historical event data is versioned in the `Summary` JSON column in Spanner. To prevent breaking changes:

1.  **Versioned Blobs**: Store snapshots as immutable versioned blobs (e.g., `v1`, `v2`).
2.  **Visitor Interface**: Use the `SummaryVisitor` defined in [`lib/workertypes/events.go`](../../../lib/workertypes/events.go) to parse legacy JSON versions into the common `EventSummary` struct.
3.  **Forward Compatibility**: Use `generic.OptionallySet[T]` to handle new optional fields without crashing older consumer instances.
