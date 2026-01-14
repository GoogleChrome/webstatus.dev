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

// JobFrequency defines the frequency that triggered this batch refresh.
type JobFrequency string

const (
	FrequencyUnknown   JobFrequency = "UNKNOWN"
	FrequencyImmediate JobFrequency = "IMMEDIATE"
	FrequencyWeekly    JobFrequency = "WEEKLY"
	FrequencyMonthly   JobFrequency = "MONTHLY"
)

func (f JobFrequency) ToWorkerTypeJobFrequency() workertypes.JobFrequency {
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

// SearchConfigurationChangedEvent is the signal sent by the API when a user modifies a search.
// It corresponds to the "Cold Start" (Creation) or "Query Change" (Update).
type SearchConfigurationChangedEvent struct {
	// SearchID is the target saved search.
	SearchID string `json:"search_id"`

	// Query is the NEW search filter.
	Query string `json:"query"`

	// UserID identifies who made the change.
	// While the Differ doesn't need this, it is critical for audit logging.
	UserID string `json:"user_id"`

	// Timestamp is when the user performed the action.
	Timestamp time.Time `json:"timestamp"`

	// IsCreation is a helper flag to distinguish between Create (true) and Update (false).
	// Note: The Differ will auto-detect "Cold Start" regardless of this flag based on
	// missing state, but this helps with routing/logging logic.
	IsCreation bool `json:"is_creation"`

	// Frequency indicates which schedule triggered this refresh.
	Frequency JobFrequency `json:"frequency"`
}

func (SearchConfigurationChangedEvent) Kind() string       { return "SearchConfigurationChangedEvent" }
func (SearchConfigurationChangedEvent) APIVersion() string { return "v1" }
