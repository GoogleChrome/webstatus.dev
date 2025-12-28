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
Package v1 implements the Version 1 logic for feature list diffing.
It includes the storage types, the comparator logic, and the reconciliation workflows.

# Handling Data Evolution (Schema Changes)

The system handles the long-tail of persistent state files where the stored schema might
lag behind the current code by months or years.

 1. Adding New Fields (The "OptionallySet Strategy"):
    When we add a new field to the struct, existing blobs in GCS will not have this data.
    Unmarshalling results in zero-values. To prevent false positive "Added" alerts (e.g. ""->"foo"),
    we must distinguish "Field Missing" from "Field Zero/Empty".

    - Consistency: We wrap ALL evolving fields (both primitives and complex types) in a
    generic `generic.OptionallySet[T]` struct.
    - Behavior:
    - Old Blob (Field Missing): 'IsSet' is false.
    - New Data (Zero Value): 'IsSet' is true. 'Value' is zero.
    - Action: The `comparator.go` logic checks 'if old.Field.IsSet' before diffing. If false,
    it skips the comparison, suppressing false positives during the rollout.

 2. Removing Fields:
    If a field is removed from the code, `json.Unmarshal` effectively ignores the data
    present in old blobs. The Comparator logic for that field is deleted.
    Result: No diffs generated. The data is silently dropped from the next snapshot.

# DEVELOPER GUIDE: How to Add a New Field

If you need to track a new data point (e.g. "SpecLink"), follow this checklist to ensure
you don't trigger 10,000 false-positive alerts on deployment.

 1. Update Canonical Types (`lib/workertypes/comparables`):
    Add the field to `comparables.Feature`. Wrap it in `OptionallySet[T]`.

 2. Update V1 Types (`lib/blobtypes/featurelist/v1`):
    Add the field to the V1 `Feature` struct so it can be persisted.

 3. Update Adapters (`workers/event_producer/pkg/producer/diff.go`):
    Update `convertV1FeatureToComparable` and `convertComparableToV1Feature` to map
    the data between storage and memory.

 4. Update Fetcher (`workers/event_producer/pkg/differ/differ.go`):
    Update `toComparable` to populate the field from the backend API response.
    You MUST set `IsSet: true` explicitly for the new data.

 5. Update Logic (`lib/blobtypes/featurelistdiff/v1/comparator.go`):
    Add a check in `compareFeature`. You MUST guard it with `.IsSet`.
    ```go
    if oldF.SpecLink.IsSet && oldF.SpecLink.Value != newF.SpecLink.Value {
    // record change
    }
    ```
*/
package v1
