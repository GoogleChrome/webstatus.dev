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

package differ

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/blobtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

const (
	// KindFeatureListSnapshot identifies a full state dump of features.
	KindFeatureListSnapshot = "FeatureListSnapshot"

	// VersionFeatureListSnapshot identifies v1 of the FeatureListSnapshot schema.
	V1FeatureListSnapshot = "v1"

	// KindFeatureListDiff identifies a delta report of feature changes.
	KindFeatureListDiff = "FeatureListDiff"

	// V1FeatureListDiff identifies version v1 of the FeatureListDiff schema.
	V1FeatureListDiff = "v1"
)

// BlobFormat defines the serialization format and file extension for blobs.
type BlobFormat string

const (
	// BlobFormatJSON indicates the blob is serialized as standard JSON.
	BlobFormatJSON BlobFormat = "json"
)

// DiffResult encapsulates the complete output of a Run.
// It provides the caller with opaque bytes for storage and structured data for the DB,
// isolating them from the internal versioning of FeatureDiff or Snapshot structs.
type DiffResult struct {
	HasChanges bool

	// Format indicates the serialization format of StateBytes and DiffBytes.
	// The consumer should use this to determine the file extension (e.g. ".json").
	Format BlobFormat

	// State Persistence
	// The new state snapshot to be saved to blob storage.
	StateBytes []byte
	StateID    string // The unique ID generated for this snapshot (e.g. "state_<timestamp>")

	// Diff Persistence
	// The diff blob to be saved to blob storage. Only present if HasChanges is true.
	DiffBytes []byte
	DiffID    string // The unique ID generated for this event (UUID).

	// DB Event Data
	// Structured data required to publish the notification event to the database.
	Summary workertypes.EventSummary
	Reasons []string // e.g. ["DATA_UPDATED", "QUERY_EDITED"]
}

var (
	ErrTransient = errors.New("transient failure")
	ErrFatal     = errors.New("fatal error")
)

// ChangeReason describes why a feature was added or removed.
type ChangeReason string

const (
	ReasonNewMatch  ChangeReason = "new_match"
	ReasonUnmatched ChangeReason = "unmatched"
	ReasonDeleted   ChangeReason = "deleted"
)

// FeatureFetcher abstracts the external API.
type FeatureFetcher interface {
	FetchFeatures(ctx context.Context, query string) ([]backend.Feature, error)
	GetFeature(ctx context.Context, featureID string) (*backendtypes.GetFeatureResult, error)
}

type FeatureDiffer struct {
	client   FeatureFetcher
	migrator *blobtypes.Migrator
	// For testing purposes
	idGen idGenerator
	now   func() time.Time
}

func NewFeatureDiffer(client FeatureFetcher) *FeatureDiffer {
	m := blobtypes.NewMigrator()
	d := &FeatureDiffer{
		client:   client,
		migrator: m,
		idGen:    &defaultIDGenerator{},
		now:      time.Now,
	}

	return d
}

// --- Generics ---

// OptionallySet allows distinguishing between "Missing Field" (Schema Cold Start)
// and "Zero Value" (Valid Data).
//
// ARCHITECTURE NOTE:
// This wrapper is used exclusively for STATE types (Snapshots) stored in GCS.
// It allows us to safely evolve the schema over time.
//
// - If a field is added to the struct, old blobs won't have it.
// - json.Unmarshal skips it, leaving IsSet=false.
// - The Comparator sees IsSet=false and skips diffing that field.
//
// Do NOT use this for Diff/Event types, as they are generated fresh and do not
// have "missing history" problems.
type OptionallySet[T any] struct {
	Value T
	IsSet bool
}

// IsZero enables the 'omitzero' JSON tag to work correctly.
// If IsSet is false, this struct is considered "Zero" and will be omitted from JSON output.
func (o OptionallySet[T]) IsZero() bool {
	return !o.IsSet
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (o *OptionallySet[T]) UnmarshalJSON(data []byte) error {
	o.IsSet = true

	return json.Unmarshal(data, &o.Value)
}

// MarshalJSON implements the json.Marshaler interface.
func (o OptionallySet[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Value)
}

// Change represents a value transition from Old to New.
type Change[T any] struct {
	From T `json:"from"`
	To   T `json:"to"`
}

// --- State Types (Input/Output of Ingestion) ---

// FeatureListSnapshot represents the persisted state of a search.
type FeatureListSnapshot struct {
	Metadata StateMetadata   `json:"metadata"`
	Data     FeatureListData `json:"data"`
}

func (s FeatureListSnapshot) Kind() string    { return KindFeatureListSnapshot }
func (s FeatureListSnapshot) Version() string { return V1FeatureListSnapshot }

type StateMetadata struct {
	ID             string    `json:"id"`
	GeneratedAt    time.Time `json:"generatedAt"`
	SearchID       string    `json:"searchId"`
	QuerySignature string    `json:"querySignature"`
	EventID        string    `json:"eventId,omitempty"`
}

type FeatureListData struct {
	Features map[string]ComparableFeature `json:"features"`
}

// BaselineState captures the full status context for a feature's baseline.
// We use OptionallySet for fields here to ensure consistency with other state structs.
type BaselineState struct {
	Status   OptionallySet[backend.BaselineInfoStatus] `json:"status,omitzero"`
	LowDate  OptionallySet[*time.Time]                 `json:"lowDate,omitzero"`
	HighDate OptionallySet[*time.Time]                 `json:"highDate,omitzero"`
}

// ComparableFeature is the struct we generate the signature from.
type ComparableFeature struct {
	ID             string                                `json:"id"`
	Name           OptionallySet[string]                 `json:"name,omitzero"`
	BaselineStatus OptionallySet[BaselineState]          `json:"baselineStatus,omitzero"`
	BrowserImpls   OptionallySet[BrowserImplementations] `json:"browserImplementations,omitzero"`
	Docs           OptionallySet[Docs]                   `json:"docs,omitzero"`
}

type Docs struct {
	MdnDocs OptionallySet[[]MdnDoc] `json:"mdnDocs,omitzero"`
}

// Representation of https://github.com/web-platform-dx/web-features-mappings/blob/main/mappings/mdn-docs.json
// Mapping data can change structure so mark all of these as pointers.
type MdnDoc struct {
	URL   OptionallySet[*string] `json:"url,omitzero"`
	Title OptionallySet[*string] `json:"title,omitzero"`
	Slug  OptionallySet[*string] `json:"slug,omitzero"`
}

// setBrowserState is a helper to set the correct browser field in BrowserImplementations.
func (b *BrowserImplementations) setBrowserState(browser backend.SupportedBrowsers, state OptionallySet[BrowserState]) {
	switch browser {
	case backend.Chrome:
		b.Chrome = state
	case backend.ChromeAndroid:
		b.ChromeAndroid = state
	case backend.Edge:
		b.Edge = state
	case backend.Firefox:
		b.Firefox = state
	case backend.FirefoxAndroid:
		b.FirefoxAndroid = state
	case backend.Safari:
		b.Safari = state
	case backend.SafariIos:
		b.SafariIos = state
	}
}

// BrowserImplementations defines the specific browsers we track.
// Using a struct with OptionallySet allows us to add new browsers (e.g. Ladybird)
// in the future without triggering false "Added" alerts on old blobs.
type BrowserImplementations struct {
	Chrome         OptionallySet[BrowserState] `json:"chrome"`
	ChromeAndroid  OptionallySet[BrowserState] `json:"chrome_android"`
	Edge           OptionallySet[BrowserState] `json:"edge"`
	Firefox        OptionallySet[BrowserState] `json:"firefox"`
	FirefoxAndroid OptionallySet[BrowserState] `json:"firefox_android"`
	Safari         OptionallySet[BrowserState] `json:"safari"`
	SafariIos      OptionallySet[BrowserState] `json:"safari_ios"`
}

// BrowserState captures the implementation details for a specific browser.
type BrowserState struct {
	Status  OptionallySet[backend.BrowserImplementationStatus] `json:"status,omitzero"`
	Date    OptionallySet[*time.Time]                          `json:"date,omitzero"`
	Version OptionallySet[*string]                             `json:"version,omitzero"`
}

// --- Diff Types (Output of Ingestion / Input of Delivery) ---
// These types represent the EVENT generated by comparing two States.
// We do NOT use OptionallySet here because Diffs are immutable records created
// at a point in time. If a field is empty in a Diff, it simply means no change occurred
// or the data wasn't available, which is handled by pointers or zero values.

// LatestFeatureDiff is an alias for the latest version of the FeatureDiff struct.
// When a new version is created (e.g. FeatureDiffV2), this alias should be updated to point to the new version.
type LatestFeatureDiff = FeatureDiffV1

// LatestFeatureDiffSnapshot is an alias for the latest version of the FeatureDiffSnapshot struct.
type LatestFeatureDiffSnapshot = FeatureDiffSnapshotV1

// LatestFeatureDiffVersion is a constant for the latest version of the FeatureListDiff schema.
const LatestFeatureDiffVersion = V1FeatureListDiff

type FeatureDiffSnapshotV1 struct {
	Metadata DiffMetadataV1 `json:"metadata"`
	Data     FeatureDiffV1  `json:"data"`
}

func (d FeatureDiffSnapshotV1) Kind() string    { return KindFeatureListDiff }
func (d FeatureDiffSnapshotV1) Version() string { return V1FeatureListDiff }

type DiffMetadataV1 struct {
	GeneratedAt     time.Time `json:"generatedAt"`
	EventID         string    `json:"eventId"`
	SearchID        string    `json:"searchId"`
	ID              string    `json:"id"`
	PreviousStateID string    `json:"previousStateId,omitempty"`
	NewStateID      string    `json:"newStateId"`
}

type FeatureDiffV1 struct {
	QueryChanged bool              `json:"queryChanged,omitempty"`
	Added        []FeatureAdded    `json:"added,omitempty"`
	Removed      []FeatureRemoved  `json:"removed,omitempty"`
	Modified     []FeatureModified `json:"modified,omitempty"`
	Moves        []FeatureMoved    `json:"moves,omitempty"`
	Splits       []FeatureSplit    `json:"splits,omitempty"`
}

// Sort orders all slices deterministically by Name (primary) and ID (secondary).
// This ensures stable JSON output and organized UI/Email lists.
func (d *FeatureDiffV1) Sort() {
	sort.Slice(d.Added, func(i, j int) bool {
		if d.Added[i].Name != d.Added[j].Name {
			return d.Added[i].Name < d.Added[j].Name
		}

		return d.Added[i].ID < d.Added[j].ID
	})
	sort.Slice(d.Removed, func(i, j int) bool {
		if d.Removed[i].Name != d.Removed[j].Name {
			return d.Removed[i].Name < d.Removed[j].Name
		}

		return d.Removed[i].ID < d.Removed[j].ID
	})
	sort.Slice(d.Modified, func(i, j int) bool {
		if d.Modified[i].Name != d.Modified[j].Name {
			return d.Modified[i].Name < d.Modified[j].Name
		}

		return d.Modified[i].ID < d.Modified[j].ID
	})
	sort.Slice(d.Moves, func(i, j int) bool {
		if d.Moves[i].FromName != d.Moves[j].FromName {
			return d.Moves[i].FromName < d.Moves[j].FromName
		}

		return d.Moves[i].FromID < d.Moves[j].FromID
	})
	sort.Slice(d.Splits, func(i, j int) bool {
		if d.Splits[i].FromName != d.Splits[j].FromName {
			return d.Splits[i].FromName < d.Splits[j].FromName
		}

		return d.Splits[i].FromID < d.Splits[j].FromID
	})

	// Also sort the targets within a Split
	for k := range d.Splits {
		to := d.Splits[k].To
		sort.Slice(to, func(i, j int) bool {
			if to[i].Name != to[j].Name {
				return to[i].Name < to[j].Name
			}

			return to[i].ID < to[j].ID
		})
	}
}

type FeatureAdded struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Reason ChangeReason `json:"reason"`
	Docs   *Docs        `json:"docs,omitempty"`
}

type FeatureRemoved struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Reason ChangeReason `json:"reason"`
}

type FeatureMoved struct {
	FromID   string `json:"fromId"`
	ToID     string `json:"toId"`
	FromName string `json:"fromName"`
	ToName   string `json:"toName"`
}

type FeatureSplit struct {
	FromID   string         `json:"fromId"`
	FromName string         `json:"fromName"`
	To       []FeatureAdded `json:"to"`
}

type FeatureModified struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Docs *Docs  `json:"docs,omitempty"`

	NameChange     *Change[string]                                     `json:"nameChange,omitempty"`
	BaselineChange *Change[BaselineState]                              `json:"baselineChange,omitempty"`
	BrowserChanges map[backend.SupportedBrowsers]*Change[BrowserState] `json:"browserChanges,omitempty"`
	DocsChange     *Change[Docs]                                       `json:"docsChange,omitempty"`
}

func (d FeatureDiffV1) Summarize() workertypes.EventSummary {
	var s workertypes.EventSummary
	s.SchemaVersion = workertypes.VersionEventSummaryV1
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

	return s
}

func (d FeatureDiffV1) HasChanges() bool {
	return d.QueryChanged || len(d.Added) > 0 || len(d.Removed) > 0 ||
		len(d.Modified) > 0 || len(d.Moves) > 0 || len(d.Splits) > 0
}
