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
Package differ is the generic "Brain" of the notification system.
It is responsible for detecting changes in Saved Searches over time and producing
structured events for the Delivery Worker to act upon.

# Architecture: The Diffing Pipeline

This package operates as a unidirectional pipeline that transforms "State" into "Diffs".
It is version-agnostic and relies on injected interfaces to handle specific data shapes.

The pipeline consists of distinct phases managed by the FeatureDiffer (Orchestrator):

 1. Context Loading (StateAdapter)
    We read the opaque Previous State blob from bytes. The `StateAdapter` is responsible
    for migration, unmarshaling, and converting the versioned blob into a canonical
    in-memory format (`comparables.Feature`).

 2. Planning (Orchestration Layer)
    We decide *what* to fetch based on the input:
    - Cold Start: No previous state? Fetch current data, save it, but generate no diffs.
    - Query Change: Did the user edit the filter? We perform a "Flush" (fetch data using
    the OLD query) to ensure we capture legitimate data changes before switching to the new query.
    - Standard: Compare Previous State vs. Current Live Data.

 3. Workflow Execution (Business Logic Layer)
    We delegate the core logic to the `StateCompareWorkflow`.
    - Calculation: Compares the old and new canonical maps to identify additions, removals, and modifications.
    - Reconciliation: Checks history to distinguish "Deleted" from "Moved" or "Split".
    - Summary: Generates a human/machine-readable summary of the changes.

 4. Serialization (Serializer Layer)
    If changes are detected, we use the `StateAdapter` to serialize the new snapshot and
    the `DiffSerializer` to serialize the diff report. These artifacts are returned to
    the caller for persistence.

# Handling Concurrency: Upstream vs. Schema Changes

It is possible for the Web Platform data to change (Data Update) at the exact same time
we deploy a new Worker version (Schema Update). The `FeatureDiffer` relies on the
robustness of the injected `StateAdapter` and `Workflow` to handle these race conditions
(typically via "Quiet Rollouts" using OptionallySet fields).

# Dependencies

  - lib/workertypes/comparables: Defines the canonical in-memory data structures.
  - lib/generic: Provides utilities for handling optional fields.
*/
package differ
