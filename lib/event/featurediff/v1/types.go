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

package v1

import "time"

// FeatureDiffEvent represents a change event for a saved search query.
// It is the contract between the Event Producer and downstream workers (e.g., Twitter Bot, RSS).
type FeatureDiffEvent struct {
	// EventID is the unique identifier for this specific execution/trigger.
	EventID string `json:"event_id"`
	// SearchID is the identifier of the saved search that was checked.
	SearchID string `json:"search_id"`
	// Query is the actual text of the query (e.g. "browsers:chrome").
	// Provided here so consumers don't need to look it up.
	Query string `json:"query"`
	// Reasons contains machine-readable tags explaining why this event was generated
	// (e.g. ["DATA_UPDATED", "QUERY_EDITED"]).
	Reasons []string `json:"reasons"`
	// Summary is a JSON string summarizing the changes (e.g. '{"added": 1, "removed": 0}').
	Summary string `json:"summary"`
	// StateID is the ID of the full snapshot blob in storage.
	StateID string `json:"state_id"`
	// DiffID is the ID of the diff blob in storage.
	DiffID string `json:"diff_id"`
	// GeneratedAt is the timestamp when the event was created.
	GeneratedAt time.Time `json:"generated_at"`
}

func (FeatureDiffEvent) Kind() string       { return "FeatureDiffEvent" }
func (FeatureDiffEvent) APIVersion() string { return "v1" }
