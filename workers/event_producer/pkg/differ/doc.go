// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package differ is the "Brain" of the notification system.
It is responsible for detecting changes in Saved Searches over time and producing
structured events for the Delivery Worker to act upon.

# Architecture: The Diffing Pipeline

This package operates as a unidirectional pipeline that transforms "State" into "Diffs".
It does not know about Users, Subscriptions, or Notification Channels. Its only job is to
report facts about how the data has changed.

The pipeline consists of four distinct phases managed by the FeatureDiffer (Orchestrator):

 1. Context Loading & Migration (Infrastructure Layer)
    We read the Previous State blob from GCS. Because state files persist indefinitely,
    the schema may have drifted from the current code. We use 'lib/blobtypes' to
    automatically upgrade old JSON blobs (e.g. "v1") to the current struct structure
    before business logic ever sees them.

 2. Planning (Orchestration Layer)
    We decide *what* to fetch based on the input:
    - Cold Start: No previous state? Fetch current data, save it, but generate no diffs.
    - Query Change: Did the user edit the filter? We perform a "Flush" (fetch data using
    the OLD query) to ensure we capture legitimate data changes before switching to the new query.
    - Standard: Compare Previous State vs. Current Live Data.

 3. Comparison (Pure Logic Layer)
    We compare the Normalized Previous State against the Current Live Data.
    This logic ('comparator.go') is version-agnostic. It relies on the Migrator to have
    already upgraded the old data. It produces a list of Added, Removed, and Modified features.

 4. Reconciliation (Enrichment Layer)
    We analyze the "Removed" features to distinguish between "Deleted" vs. "Moved/Split".
    We query the history (MovedWebFeatures/SplitWebFeatures tables) to correlate
    a Removal with a corresponding Addition, transforming them into a "Move" or "Split" event.

# Handling Data Evolution (Schema Changes)

The system handles the long-tail of persistent state files where the stored schema might
lag behind the current code by months or years.

 1. Adding New Fields (The "OptionallySet Strategy"):
    When we add a new field to the struct, existing blobs in GCS will not have this data.
    Unmarshalling results in zero-values. To prevent false positive "Added" alerts (e.g. ""->"foo"),
    we must distinguish "Field Missing" from "Field Zero/Empty".

    - Consistency: We wrap ALL evolving fields (both primitives and complex types) in a
    generic 'OptionallySet[T]' struct.
    - Behavior:
    - Old Blob (Field Missing): 'IsSet' is false.
    - New Data (Zero Value): 'IsSet' is true. 'Value' is zero.
    - Action: The Comparator checks 'if old.Field.IsSet' before diffing. If false,
    it skips the comparison, suppressing false positives during the rollout.

 2. Removing Fields:
    If a field is removed from the code, 'json.Unmarshal' effectively ignores the data
    present in old blobs. The Comparator logic for that field is deleted.
    Result: No diffs generated. The data is silently dropped from the next snapshot.

 3. Renaming Fields:
    Renaming is treated as "Remove Old" + "Add New". Without intervention, this loses history.
    To preserve history, use the Migrator (lib/blobtypes) to map the old JSON key to the
    new struct field during the 'loadPreviousContext' phase.

# Handling Concurrency: Upstream vs. Schema Changes

It is possible for the Web Platform data to change (Data Update) at the exact same time
we deploy a new Worker version (Schema Update).

  - Mixed Scenario (Data + Schema): If an existing field updates (e.g. Status change) AND
    a new field appears (Schema update) in the same run:
    1. The existing field change IS detected and reported (Logic for stable fields remains valid).
    2. The new field appearance is suppressed (to avoid "Added" noise).
    Result: The user receives a valid alert about the Status change, but the email
    won't mention the new field yet. This is the desired "Quiet Rollout" behavior.

  - New Fields Only: If the ONLY change is the appearance of a new field, the system
    suppresses the alert entirely. To guarantee capture of updates for new fields immediately
    upon rollout (without waiting for a second run), pause upstream data ingestion during deployment.
    (Alternative: Backfilling historical state blobs to include default values for new fields was
    considered to solve this race condition, but rejected due to high operational complexity and cost).

# Dependencies

  - lib/blobtypes: Handles the generic migration of storage blobs.
  - lib/backendtypes: Provides the Visitor pattern for history lookups.
*/
package differ
