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

// EmailJobEvent represents an email job event.
type EmailJobEvent struct {
	// SubscriptionID is the ID of the subscription that triggered this job.
	SubscriptionID string `json:"subscription_id"`
	// RecipientEmail is the email address of the recipient.
	RecipientEmail string `json:"recipient_email"`
	// SummaryRaw is the raw JSON bytes of the event summary.
	SummaryRaw []byte `json:"summary_raw"`
	// Metadata contains additional metadata about the event.
	Metadata EmailJobEventMetadata `json:"metadata"`
	// ChannelID is the ID of the channel associated with this job.
	ChannelID string `json:"channel_id"`
}

type EmailJobEventMetadata struct {
	// EventID is the ID of the original event that triggered this job.
	EventID string `json:"event_id"`
	// SearchID is the ID of the search that generated the event.
	SearchID string `json:"search_id"`
	// Query is the query string used for the search.
	Query string `json:"query"`
	// Frequency is the frequency of the job (e.g., "daily", "weekly").
	Frequency JobFrequency `json:"frequency"`
	// GeneratedAt is the timestamp when the original event was generated.
	GeneratedAt time.Time `json:"generated_at"`
}

func (EmailJobEvent) Kind() string       { return "EmailJobEvent" }
func (EmailJobEvent) APIVersion() string { return "v1" }

type JobFrequency string

const (
	FrequencyUnknown   JobFrequency = "UNKNOWN"
	FrequencyImmediate JobFrequency = "IMMEDIATE"
	FrequencyWeekly    JobFrequency = "WEEKLY"
	FrequencyMonthly   JobFrequency = "MONTHLY"
)

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
