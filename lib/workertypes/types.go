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

package workertypes

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	v1 "github.com/GoogleChrome/webstatus.dev/lib/blobtypes/featurelistdiff/v1"
)

var (
	ErrUnknownSummaryVersion    = errors.New("unknown summary version")
	ErrFailedToSerializeSummary = errors.New("failed to serialize summary")
)

const (
	// VersionEventSummaryV1 defines the schema version for v1 of the EventSummary.
	VersionEventSummaryV1 = "v1"
)

type SavedSearchState struct {
	StateBlobPath *string
}

type SavedSearchStateUpdateRequest struct {
	StateBlobPath *string

	UpdateMask []SavedSearchStateUpdateRequestUpdateMask
}

type SavedSearchStateUpdateRequestUpdateMask string

const (
	SavedSearchStateUpdateRequestStateBlobPath SavedSearchStateUpdateRequestUpdateMask = "state_blob_path"
)

// NotificationEventRequest encapsulates the data needed to insert a row into the Events table.
type NotificationEventRequest struct {
	EventID      string
	SearchID     string
	SnapshotType string
	Reasons      []string
	DiffBlobPath string
	Summary      EventSummary
	NewStatePath string
	WorkerID     string
}

// SummaryCategories defines the specific counts for different change types.
type SummaryCategories struct {
	QueryChanged    int `json:"query_changed,omitzero"`
	Added           int `json:"added,omitzero"`
	Removed         int `json:"removed,omitzero"`
	Moved           int `json:"moved,omitzero"`
	Split           int `json:"split,omitzero"`
	Updated         int `json:"updated,omitzero"`
	UpdatedImpl     int `json:"updated_impl,omitzero"`
	UpdatedRename   int `json:"updated_rename,omitzero"`
	UpdatedBaseline int `json:"updated_baseline,omitzero"`
}

// EventSummary matches the JSON structure stored in the database 'Summary' column.
type EventSummary struct {
	SchemaVersion string            `json:"schemaVersion"`
	Text          string            `json:"text"`
	Categories    SummaryCategories `json:"categories,omitzero"`
}

// SummaryVisitor defines the contract for consuming immutable Event Summaries.
// Unlike state blobs which are migrated to the latest schema, summaries are
// historical records that should be rendered as-is. The Visitor pattern forces
// consumers to explicitly handle each schema version (e.g. V1, V2) independently.
type SummaryVisitor interface {
	VisitV1(s EventSummary) error
}

// ParseEventSummary handles the version detection and dispatching logic.
// Consumers (like the Delivery Worker) should use this instead of raw json.Unmarshal.
func ParseEventSummary(data []byte, v SummaryVisitor) error {
	// 1. Peek at version
	var header struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return fmt.Errorf("invalid summary json: %w", err)
	}

	// 2. Dispatch
	switch header.SchemaVersion {
	case VersionEventSummaryV1:
		var s EventSummary
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("failed to parse v1 summary: %w", err)
		}

		return v.VisitV1(s)
	default:
		return fmt.Errorf("%w: %q", ErrUnknownSummaryVersion, header.SchemaVersion)
	}
}

type FeatureDiffV1SummaryGenerator struct{}

// GenerateJSONEventSummaryFromFeatureDiffV1 generates and serializes.
func (g FeatureDiffV1SummaryGenerator) GenerateJSONSummary(
	d v1.FeatureDiff) ([]byte, error) {
	var s EventSummary
	s.SchemaVersion = VersionEventSummaryV1
	var parts []string

	if d.QueryChanged {
		parts = append(parts, "Search criteria updated")
		s.Categories.QueryChanged = 1
	}

	if len(d.Added) > 0 {
		parts = append(parts, fmt.Sprintf("%d features added", len(d.Added)))
		s.Categories.Added = len(d.Added)
	}
	if len(d.Removed) > 0 {
		parts = append(parts, fmt.Sprintf("%d features removed", len(d.Removed)))
		s.Categories.Removed = len(d.Removed)
	}
	if len(d.Moves) > 0 {
		parts = append(parts, fmt.Sprintf("%d features moved/renamed", len(d.Moves)))
		s.Categories.Moved = len(d.Moves)
	}
	if len(d.Splits) > 0 {
		parts = append(parts, fmt.Sprintf("%d features split", len(d.Splits)))
		s.Categories.Split = len(d.Splits)
	}

	if len(d.Modified) > 0 {
		parts = append(parts, fmt.Sprintf("%d features updated", len(d.Modified)))
		s.Categories.Updated = len(d.Modified)

		for _, m := range d.Modified {
			if len(m.BrowserChanges) > 0 {
				s.Categories.UpdatedImpl++
			}
			if m.NameChange != nil {
				s.Categories.UpdatedRename++
			}
			if m.BaselineChange != nil {
				s.Categories.UpdatedBaseline++
			}
		}
	}

	if len(parts) == 0 {
		s.Text = "No changes detected"
	} else {
		s.Text = strings.Join(parts, ", ")
	}

	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToSerializeSummary, err)
	}

	return b, nil
}

type Reason string

const (
	ReasonQueryChanged = "QUERY_CHANGED"
	ReasonDataUpdated  = "DATA_UPDATED"
)

type PublishEventRequest struct {
	EventID  string
	StateID  string
	DiffID   string
	SearchID string
	Summary  []byte
	Reasons  []Reason
}

type LatestEventInfo struct {
	EventID string
	StateID string
}
