# Worker Pipeline Architecture

The `workers/` directory contains Go applications that form an event-driven notification pipeline using Google Cloud Pub/Sub and Spanner.

The pipeline consists of three main stages, generally categorized into **Push** workers (that send notifications out based on subscriptions) and **Pull** workers (that generate feeds or data to be pulled by external clients).

## 1. Event Producer (`workers/event_producer/`)

Listens to ingestion events and batch update events.

- **Role:** The "Brain". It loads the previous state (from GCS via `StateAdapter`), fetches the current live data via `FeatureFetcher`, and uses a `FeatureDiffer` (e.g. `StateCompareWorkflow`) to calculate the differences (Added, Removed, Deleted, Moved, Split, Baseline changed, Browser Implementation changed).
- **Output:** If there are changes, it serializes a new State Blob, a Diff Blob, and publishes a new `PublishEventRequest` message to the notification topic.

## 2. Push Delivery (`workers/push_delivery/`)

Consumes `PublishEventRequest` messages from the notification topic.

- **Role:** The "Dispatcher" or "Fan-out" layer for **Push** workers. It queries Spanner (`SubscriptionFinder`) to find all users subscribed to the `SearchID` and `Frequency` of the event.
- **Filtering:** It parses the Event Summary and compares the changes against each user's `JobTrigger` list using a `SummaryVisitor`. If the user's triggers match the changes, it creates a delivery job.
- **Output:** Publishes a specific delivery job (e.g. `EmailDeliveryJob`, `WebhookDeliveryJob`) to a channel-specific topic.

## 3. Delivery Workers (e.g., `workers/email/`)

Consumes channel-specific delivery job messages (e.g., `EmailDeliveryJob`).

- **Role:** These are **Push** workers. They format the diff summary appropriately (e.g., into an HTML email or a JSON payload) and send it out.
- **State Management:** Uses `ChannelStateManager` to record delivery success or failure in Spanner. Permanent errors are ACKed, transient errors are NACKed for retry.

## Pull Workers (e.g., RSS)

- **Role:** These bypass the Push Delivery layer because they aren't tied to user subscriptions. They typically listen to the notification topic directly to pre-compute feeds or serve requests dynamically from the HTTP backend by fetching the stored Diff or State blobs.

## Schema Evolution & Blobs

- State for saved search notifications is stored in GCS blobs (`lib/blobtypes`).
- Canonical in-memory types (`lib/workertypes/comparables`) are decoupled from storage types (`lib/blobtypes/v1`).
- We use `generic.OptionallySet[T]` to gracefully handle new fields added over time. Unset fields from older blobs are ignored, whereas Set fields are processed.
