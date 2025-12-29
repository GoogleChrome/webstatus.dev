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

import (
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type JobFrequency string

const (
	FrequencyUnknown   JobFrequency = "UNKNOWN"
	FrequencyImmediate JobFrequency = "IMMEDIATE"
	FrequencyWeekly    JobFrequency = "WEEKLY"
	FrequencyMonthly   JobFrequency = "MONTHLY"
)

type Reason string

const (
	ReasonQueryChanged Reason = "QUERY_CHANGED"
	ReasonDataUpdated  Reason = "DATA_UPDATED"
)

// FeatureDiffEvent represents a change event for a saved search query.
// It is the contract between the Event Producer and downstream workers (e.g. Push Delivery Worker).
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
	Reasons []Reason `json:"reasons"`
	// Summary is a JSON string summarizing the changes (e.g. '{"added": 1, "removed": 0}').
	Summary []byte `json:"summary"`
	// StateID is the ID of the full snapshot blob in storage.
	StateID string `json:"state_id"`
	// StateBlobPath is the path to the full snapshot blob in storage.
	StateBlobPath string `json:"state_blob_path"`
	// DiffID is the ID of the diff blob in storage.
	DiffID string `json:"diff_id"`
	// DiffBlobPath is the path to the diff blob in storage.
	DiffBlobPath string `json:"diff_blob_path"`
	// GeneratedAt is the timestamp when the event was created.
	GeneratedAt time.Time `json:"generated_at"`
	// Frequency is the frequency that triggered the generation of this diff.
	Frequency JobFrequency `json:"frequency"`
}

func (FeatureDiffEvent) Kind() string       { return "FeatureDiffEvent" }
func (FeatureDiffEvent) APIVersion() string { return "v1" }

func (f JobFrequency) ToWorkertypes() workertypes.JobFrequency {
	switch f {
	case FrequencyImmediate:
		return workertypes.FrequencyImmediate
	case FrequencyWeekly:
		return workertypes.FrequencyWeekly
	case FrequencyMonthly:
		return workertypes.FrequencyMonthly
	case FrequencyUnknown:
		return workertypes.FrequencyUnknown
	}

	return workertypes.FrequencyUnknown
}

func ToJobFrequency(freq workertypes.JobFrequency) JobFrequency {
	switch freq {
	case workertypes.FrequencyImmediate:
		return FrequencyImmediate
	case workertypes.FrequencyWeekly:
		return FrequencyWeekly
	case workertypes.FrequencyMonthly:
		return FrequencyMonthly
	case workertypes.FrequencyUnknown:
		return FrequencyUnknown
	}

	return FrequencyUnknown
}

func ToReasons(reasons []workertypes.Reason) []Reason {
	if len(reasons) == 0 {
		return nil
	}
	result := make([]Reason, 0, len(reasons))
	for _, r := range reasons {
		switch r {
		case workertypes.ReasonQueryChanged:
			result = append(result, ReasonQueryChanged)
		case workertypes.ReasonDataUpdated:
			result = append(result, ReasonDataUpdated)
		}
	}

	return result
}
