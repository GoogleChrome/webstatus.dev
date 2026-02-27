# How to Add a New Notification Worker

The `webstatus.dev` notification system uses a Pub/Sub event-driven architecture that separates the detection of changes (Producer) from the delivery of notifications (Workers).

When adding a new type of worker, it will typically be a **Push** worker (e.g., Email, Webhooks).

## Push Workers (e.g., Webhooks, SMS, Slack)

Push workers actively send notifications to users based on user-configured subscriptions.

### Architecture Integration

1. **Push Delivery Layer (`workers/push_delivery/`)**:
   - The `push_delivery` worker acts as the dispatcher. It consumes `PublishEventRequest` events.
   - You must update the `Dispatcher` to query Spanner for the new type of channel (e.g., webhook configurations) associated with the `SearchID`.
   - The `Dispatcher` will generate a new type of job (e.g., `WebhookDeliveryJob`) and publish it to a dedicated Pub/Sub topic for your new worker (e.g., `WEBHOOK_TOPIC_ID`).
2. **The New Worker (`workers/<new_worker>/`)**:
   - Create a new directory under `workers/`.
   - The worker must subscribe to its dedicated Pub/Sub topic (e.g., `webhook-delivery-sub-id`).
   - It should consume the job payload, format the payload appropriately (e.g., into a JSON webhook payload), and perform the network request.
   - **State Management:** It must use a `ChannelStateManager` to record delivery successes or failures back to Spanner.
   - **Error Handling:** Permanent errors (e.g., 404 Not Found on a webhook URL) should be ACKed and marked as a permanent failure in the DB. Transient errors (e.g., 500 Internal Server Error) should be NACKed via `errors.Join(event.ErrTransientFailure, err)` to trigger a Pub/Sub retry.

## Pull Workers (e.g., RSS Feeds, Public API endpoints)

Pull workers do not "send" data; they serve data on demand to clients who request it.

### Architecture Integration

1. **Bypass Push Delivery**:
   - Pull workers generally do **not** need to integrate with `workers/push_delivery/`, as they are not triggered per user subscription.
2. **Read from Artifacts**:
   - When the Event Producer detects a change, it stores a `StateBlob` and a `DiffBlob` in GCS and publishes the metadata.
   - For a system like RSS, you might have a generic endpoint in the HTTP backend (e.g., `/v1/features/{id}/feed.xml`).
   - The endpoint would dynamically read the latest event metadata from Spanner, fetch the corresponding Diff or State blob from GCS, and render it into the RSS XML format on the fly.
3. **Alternatively, Pre-computation Worker**:
   - If computing the RSS feed on the fly is too expensive, you could create a worker that listens directly to the `NOTIFICATION_TOPIC_ID` (alongside `push_delivery`), computes the generic XML feeds for the changed searches, and stores them in GCS for fast retrieval by the frontend/backend.

## Adding the Worker

1. Create your Go code in `workers/<new_worker>`.
2. Initialize the Go module: `cd workers/<new_worker> && go mod init github.com/GoogleChrome/webstatus.dev/workers/<new_worker>`.
3. Add `replace` directives to your `go.mod` to use the local `lib` and `lib/gen` directories:
   ```go
   replace github.com/GoogleChrome/webstatus.dev/lib => ../../lib
   replace github.com/GoogleChrome/webstatus.dev/lib/gen => ../../lib/gen
   ```
4. Add a `manifests/pod.yaml` for local Kubernetes deployment. Unlike scheduled jobs, long-running event-driven workers must use a Kubernetes `Pod` (or `Deployment`), not a `Job`. Set `restartPolicy: Never` for local dev.
5. Add a `skaffold.yaml` referencing your Dockerfile.
6. Add the necessary local emulator environment variables to your manifest.
7. Update the root `Makefile` to include your new worker in the `go-workspace-setup` target.
8. **Terraform Integration ([`infra/`](../../../infra/) directory)**:
   - **Pub/Sub**: If the new worker requires its own topic and subscription, define them in [`infra/pubsub/main.tf`](../../../infra/pubsub/main.tf) along with their Dead Letter Queues (DLQs). Export their IDs in [`infra/pubsub/outputs.tf`](../../../infra/pubsub/outputs.tf) and ensure any new DLQs or latency metrics are added to [`infra/pubsub/alerts.tf`](../../../infra/pubsub/alerts.tf).
   - **Worker Module**: Add a new module directory in [`infra/workers/<new_worker>`](../../../infra/workers/).
     - **`main.tf`**: Define the `google_cloud_run_v2_worker_pool` resource. Most workers will also create their own Service Account here (`google_service_account`), though some (like email) may accept a pre-existing one.
     - **`iam.tf`**: Explicitly grant permissions to the worker's service account. At a minimum, long-running workers need roles for logging, monitoring, and tracing. You will also need to grant access to the specific Spanner databases, GCS buckets, and Pub/Sub subscriptions/topics it uses.
     - **`variables.tf` / `providers.tf`**: Define necessary inputs and the internal project provider.
   - **Provider & Project Placement**: Most backend infrastructure (Workers, Pub/Sub, Spanner, GCS) resides in the **internal** project. You **MUST** ensure that all resources in your new module (Service Accounts, IAM members, Worker Pools) use the internal provider.
     - **`providers.tf`**: Include `google.internal_project` in the `configuration_aliases`.
     - **Resource Definitions**: Always include `provider = google.internal_project` in the resource blocks.
   - **Pipeline Wiring**: Update [`infra/workers/main.tf`](../../../infra/workers/main.tf) to build the new worker's image using the `go_image` module and instantiate your new worker module, passing in the necessary Pub/Sub outputs and Spanner details. Ensure you pass the provider:
     ```hcl
     module "my_new_worker" {
       source = "./my_new_worker"
       providers = {
         google.internal_project = google.internal_project
       }
       # ... other variables
     }
     ```
   - **Environment Variables**: If the new worker requires manual instance counts or specific configurations, add corresponding fields to `worker_manual_instance_counts` (or other maps) in [`infra/.envs/staging.tfvars`](../../../infra/.envs/staging.tfvars) and [`infra/.envs/prod.tfvars`](../../../infra/.envs/prod.tfvars), and update the definitions in [`infra/main.tf`](../../../infra/main.tf) and [`infra/workers/variables.tf`](../../../infra/workers/variables.tf).

## Verify, Don't Assume

Always review the existing implementation of workers in `infra/workers/` (e.g. `event_producer` or `push_delivery`) as the canonical examples for structure, naming conventions, and permission sets to prevent incorrect assumptions.

## Documentation Updates

When you add a new worker, remember to update:

- `docs/ARCHITECTURE.md` to reflect the new pipeline step.
- `GEMINI.md` by triggering the "Updating the Knowledge Base" prompt to ensure I am aware of the new component.
- Any relevant files in the `skills/` directory if the worker introduces new structural patterns or rules.
