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
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
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

// --- Generics ---

// OptionallySet allows distinguishing between "Missing Field" (Schema Cold Start)
// and "Zero Value" (Valid Data).
type OptionallySet[T any] struct {
	Value T
	IsSet bool
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
	GeneratedAt    time.Time `json:"generatedAt"`
	SearchID       string    `json:"searchId"`
	QuerySignature string    `json:"querySignature"`
}

type FeatureListData struct {
	Features map[string]ComparableFeature `json:"features"`
}

// ComparableFeature is the struct we generate the signature from.
type ComparableFeature struct {
	ID             string                                    `json:"id"`
	Name           OptionallySet[string]                     `json:"name"`
	BaselineStatus OptionallySet[backend.BaselineInfoStatus] `json:"baselineStatus"`
	BrowserImpls   BrowserImplementations                    `json:"browserImplementations"`
}

// BrowserImplementations defines the specific browsers we track.
// Using a struct with OptionallySet allows us to add new browsers (e.g. Ladybird)
// in the future without triggering false "Added" alerts on old blobs.
type BrowserImplementations struct {
	Chrome         OptionallySet[string] `json:"chrome"`
	ChromeAndroid  OptionallySet[string] `json:"chrome_android"`
	Edge           OptionallySet[string] `json:"edge"`
	Firefox        OptionallySet[string] `json:"firefox"`
	FirefoxAndroid OptionallySet[string] `json:"firefox_android"`
	Safari         OptionallySet[string] `json:"safari"`
	SafariIos      OptionallySet[string] `json:"safari_ios"`
}

// --- Diff Types (Output of Ingestion / Input of Delivery) ---

type FeatureDiffSnapshot struct {
	Metadata DiffMetadata `json:"metadata"`
	Data     FeatureDiff  `json:"data"`
}

func (d FeatureDiffSnapshot) Kind() string    { return KindFeatureListDiff }
func (d FeatureDiffSnapshot) Version() string { return V1FeatureListDiff }

type DiffMetadata struct {
	GeneratedAt time.Time `json:"generatedAt"`
	EventID     string    `json:"eventId"`
	SearchID    string    `json:"searchId"`
}

type FeatureDiff struct {
	QueryChanged bool              `json:"queryChanged"`
	Added        []FeatureAdded    `json:"added"`
	Removed      []FeatureRemoved  `json:"removed"`
	Modified     []FeatureModified `json:"modified"`
	Moves        []FeatureMoved    `json:"moves"`
	Splits       []FeatureSplit    `json:"splits"`
}

// Sort orders all slices deterministically by Name (primary) and ID (secondary).
// This ensures stable JSON output and organized UI/Email lists.
func (d *FeatureDiff) Sort() {
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

	NameChange     *Change[string]                               `json:"nameChange,omitzero"`
	BaselineChange *Change[backend.BaselineInfoStatus]           `json:"baselineChange,omitzero"`
	BrowserChanges map[backend.SupportedBrowsers]*Change[string] `json:"browserChanges,omitzero"`
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
	Text       string            `json:"text"`
	Categories SummaryCategories `json:"categories,omitzero"`
}

func (d FeatureDiff) Summarize() EventSummary {
	var s EventSummary
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

func (d FeatureDiff) HasChanges() bool {
	return d.QueryChanged || len(d.Added) > 0 || len(d.Removed) > 0 ||
		len(d.Modified) > 0 || len(d.Moves) > 0 || len(d.Splits) > 0
}
