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
	"time"

	v1 "github.com/GoogleChrome/webstatus.dev/lib/blobtypes/featurelistdiff/v1"
)

var (
	ErrUnknownSummaryVersion    = errors.New("unknown summary version")
	ErrFailedToSerializeSummary = errors.New("failed to serialize summary")
)

const (
	// VersionEventSummaryV1 defines the schema version for v1 of the EventSummary.
	VersionEventSummaryV1 = "v1"
	// MaxHighlights caps the number of detailed items stored in Spanner (The full highlights are stored in GCS).
	// Spanner's 10MB limit can easily accommodate this.
	// Calculation details:
	// A typical highlight contains:
	// - Feature info (ID, Name): ~50-80 bytes
	// - 2 DocLinks (URL, Title, Slug): ~250 bytes
	// - Changes metadata: ~50 bytes
	// - JSON structure overhead: ~50 bytes
	// Total ≈ 450-500 bytes.
	// 10,000 highlights * 500 bytes = 5MB, which is 50% of the 10MB column limit.
	MaxHighlights = 10000
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
	SchemaVersion string             `json:"schemaVersion"`
	Text          string             `json:"text"`
	Categories    SummaryCategories  `json:"categories,omitzero"`
	Truncated     bool               `json:"truncated"`
	Highlights    []SummaryHighlight `json:"highlights"`
}

// Change represents a value transition from Old to New.
type Change[T any] struct {
	From T `json:"from"`
	To   T `json:"to"`
}

type FeatureRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DocLink struct {
	URL   string  `json:"url"`
	Title *string `json:"title,omitempty"`
	Slug  *string `json:"slug,omitempty"`
}

type BaselineStatus string

const (
	BaselineStatusLimited BaselineStatus = "limited"
	BaselineStatusNewly   BaselineStatus = "newly"
	BaselineStatusWidely  BaselineStatus = "widely"
	BaselineStatusUnknown BaselineStatus = "unknown"
)

type BaselineValue struct {
	Status   BaselineStatus `json:"status"`
	LowDate  *time.Time     `json:"low_date,omitempty"`
	HighDate *time.Time     `json:"high_date,omitempty"`
}

type BrowserStatus string

const (
	BrowserStatusAvailable   BrowserStatus = "available"
	BrowserStatusUnavailable BrowserStatus = "unavailable"
	BrowserStatusUnknown     BrowserStatus = ""
)

type BrowserValue struct {
	Status  BrowserStatus `json:"status"`
	Version *string       `json:"version,omitempty"`
}

type BrowserName string

const (
	BrowserChrome         BrowserName = "chrome"
	BrowserChromeAndroid  BrowserName = "chrome_android"
	BrowserEdge           BrowserName = "edge"
	BrowserFirefox        BrowserName = "firefox"
	BrowserFirefoxAndroid BrowserName = "firefox_android"
	BrowserSafari         BrowserName = "safari"
	BrowserSafariIos      BrowserName = "safari_ios"
)

type SummaryHighlightType string

const (
	SummaryHighlightTypeAdded   SummaryHighlightType = "Added"
	SummaryHighlightTypeRemoved SummaryHighlightType = "Removed"
	SummaryHighlightTypeChanged SummaryHighlightType = "Changed"
	SummaryHighlightTypeMoved   SummaryHighlightType = "Moved"
	SummaryHighlightTypeSplit   SummaryHighlightType = "Split"
)

type SplitChange struct {
	From FeatureRef   `json:"from"`
	To   []FeatureRef `json:"to"`
}

type SummaryHighlight struct {
	Type        SummaryHighlightType `json:"type"`
	FeatureID   string               `json:"feature_id"`
	FeatureName string               `json:"feature_name"`
	DocLinks    []DocLink            `json:"doc_links,omitempty"`

	// Strongly typed change fields to support i18n and avoid interface{}
	NameChange     *Change[string]                      `json:"name_change,omitempty"`
	BaselineChange *Change[BaselineValue]               `json:"baseline_change,omitempty"`
	BrowserChanges map[BrowserName]Change[BrowserValue] `json:"browser_changes,omitempty"`
	Moved          *Change[FeatureRef]                  `json:"moved,omitempty"`
	Split          *SplitChange                         `json:"split,omitempty"`
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

	s.Categories, s.Text = g.calculateCategoriesAndText(d)
	s.Highlights, s.Truncated = g.generateHighlights(d)

	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToSerializeSummary, err)
	}

	return b, nil
}

func (g FeatureDiffV1SummaryGenerator) calculateCategoriesAndText(d v1.FeatureDiff) (SummaryCategories, string) {
	var c SummaryCategories
	var parts []string

	// 1. Populate Counts (Categories)
	if d.QueryChanged {
		parts = append(parts, "Search criteria updated")
		c.QueryChanged = 1
	}
	if len(d.Added) > 0 {
		parts = append(parts, fmt.Sprintf("%d features added", len(d.Added)))
		c.Added = len(d.Added)
	}
	if len(d.Removed) > 0 {
		parts = append(parts, fmt.Sprintf("%d features removed", len(d.Removed)))
		c.Removed = len(d.Removed)
	}
	if len(d.Moves) > 0 {
		parts = append(parts, fmt.Sprintf("%d features moved/renamed", len(d.Moves)))
		c.Moved = len(d.Moves)
	}
	if len(d.Splits) > 0 {
		parts = append(parts, fmt.Sprintf("%d features split", len(d.Splits)))
		c.Split = len(d.Splits)
	}
	if len(d.Modified) > 0 {
		parts = append(parts, fmt.Sprintf("%d features updated", len(d.Modified)))
		c.Updated = len(d.Modified)
		for _, m := range d.Modified {
			if len(m.BrowserChanges) > 0 {
				c.UpdatedImpl++
			}
			if m.NameChange != nil {
				c.UpdatedRename++
			}
			if m.BaselineChange != nil {
				c.UpdatedBaseline++
			}
		}
	}

	text := "No changes detected"
	if len(parts) > 0 {
		text = strings.Join(parts, ", ")
	}

	return c, text
}

func (g FeatureDiffV1SummaryGenerator) generateHighlights(d v1.FeatureDiff) ([]SummaryHighlight, bool) {
	var highlights []SummaryHighlight
	var truncated bool

	if highlights, truncated = g.processModified(highlights, d.Modified); truncated {
		return highlights, true
	}

	if highlights, truncated = g.processAdded(highlights, d.Added); truncated {
		return highlights, true
	}

	if highlights, truncated = g.processRemoved(highlights, d.Removed); truncated {
		return highlights, true
	}

	if highlights, truncated = g.processMoves(highlights, d.Moves); truncated {
		return highlights, true
	}

	if highlights, truncated = g.processSplits(highlights, d.Splits); truncated {
		return highlights, true
	}

	return highlights, false
}

func (g FeatureDiffV1SummaryGenerator) processModified(highlights []SummaryHighlight,
	modified []v1.FeatureModified) ([]SummaryHighlight, bool) {
	for _, m := range modified {
		if len(highlights) >= MaxHighlights {
			return highlights, true
		}

		h := SummaryHighlight{
			Type:           SummaryHighlightTypeChanged,
			FeatureID:      m.ID,
			FeatureName:    m.Name,
			DocLinks:       toDocLinks(m.Docs),
			NameChange:     nil,
			BaselineChange: nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		}

		if m.BaselineChange != nil {
			h.BaselineChange = &Change[BaselineValue]{
				From: toBaselineValue(m.BaselineChange.From),
				To:   toBaselineValue(m.BaselineChange.To),
			}
		}
		if m.NameChange != nil {
			h.NameChange = &Change[string]{
				From: m.NameChange.From,
				To:   m.NameChange.To,
			}
		}

		if len(m.BrowserChanges) > 0 {
			h.BrowserChanges = make(map[BrowserName]Change[BrowserValue])
			for b, c := range m.BrowserChanges {
				if c == nil {
					continue
				}
				var key BrowserName
				switch b {
				case v1.Chrome:
					key = BrowserChrome
				case v1.ChromeAndroid:
					key = BrowserChromeAndroid
				case v1.Edge:
					key = BrowserEdge
				case v1.Firefox:
					key = BrowserFirefox
				case v1.FirefoxAndroid:
					key = BrowserFirefoxAndroid
				case v1.Safari:
					key = BrowserSafari
				case v1.SafariIos:
					key = BrowserSafariIos
				default:
					continue
				}
				h.BrowserChanges[key] = Change[BrowserValue]{
					From: toBrowserValue(c.From),
					To:   toBrowserValue(c.To),
				}
			}
		}

		highlights = append(highlights, h)
	}

	return highlights, false
}

func (g FeatureDiffV1SummaryGenerator) processAdded(highlights []SummaryHighlight,
	added []v1.FeatureAdded) ([]SummaryHighlight, bool) {
	for _, a := range added {
		if len(highlights) >= MaxHighlights {
			return highlights, true
		}
		highlights = append(highlights, SummaryHighlight{
			Type:           SummaryHighlightTypeAdded,
			FeatureID:      a.ID,
			FeatureName:    a.Name,
			DocLinks:       toDocLinks(a.Docs),
			NameChange:     nil,
			BaselineChange: nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		})
	}

	return highlights, false
}

func (g FeatureDiffV1SummaryGenerator) processRemoved(highlights []SummaryHighlight,
	removed []v1.FeatureRemoved) ([]SummaryHighlight, bool) {
	for _, r := range removed {
		if len(highlights) >= MaxHighlights {
			return highlights, true
		}
		highlights = append(highlights, SummaryHighlight{
			Type:           SummaryHighlightTypeRemoved,
			FeatureID:      r.ID,
			FeatureName:    r.Name,
			DocLinks:       nil,
			Moved:          nil,
			Split:          nil,
			BaselineChange: nil,
			NameChange:     nil,
			BrowserChanges: nil,
		})
	}

	return highlights, false
}

func (g FeatureDiffV1SummaryGenerator) processMoves(highlights []SummaryHighlight,
	moves []v1.FeatureMoved) ([]SummaryHighlight, bool) {
	for _, m := range moves {
		if len(highlights) >= MaxHighlights {
			return highlights, true
		}
		highlights = append(highlights, SummaryHighlight{
			Type:        SummaryHighlightTypeMoved,
			FeatureID:   m.ToID, // Use new ID after move
			FeatureName: m.ToName,
			Moved: &Change[FeatureRef]{
				From: FeatureRef{ID: m.FromID, Name: m.FromName},
				To:   FeatureRef{ID: m.ToID, Name: m.ToName},
			},
			BrowserChanges: nil,
			BaselineChange: nil,
			NameChange:     nil,
			DocLinks:       nil,
			Split:          nil,
		})
	}

	return highlights, false
}

func (g FeatureDiffV1SummaryGenerator) processSplits(highlights []SummaryHighlight,
	splits []v1.FeatureSplit) ([]SummaryHighlight, bool) {
	for _, split := range splits {
		if len(highlights) >= MaxHighlights {
			return highlights, true
		}
		var to []FeatureRef
		for _, t := range split.To {
			to = append(to, FeatureRef{ID: t.ID, Name: t.Name})
		}
		highlights = append(highlights, SummaryHighlight{
			Type:        SummaryHighlightTypeSplit,
			FeatureID:   split.FromID,
			FeatureName: split.FromName,
			Split: &SplitChange{
				From: FeatureRef{ID: split.FromID, Name: split.FromName},
				To:   to,
			},
			Moved:          nil,
			BrowserChanges: nil,
			BaselineChange: nil,
			NameChange:     nil,
			DocLinks:       nil,
		})
	}

	return highlights, false
}

func toDocLinks(docs *v1.Docs) []DocLink {
	if docs == nil {
		return nil
	}
	links := make([]DocLink, 0, len(docs.MdnDocs))
	for _, d := range docs.MdnDocs {
		links = append(links, DocLink{
			URL:   d.URL,
			Title: d.Title,
			Slug:  d.Slug,
		})
	}

	return links
}

func toBaselineValue(s v1.BaselineState) BaselineValue {
	val := BaselineValue{
		Status:   BaselineStatusUnknown,
		LowDate:  nil,
		HighDate: nil,
	}
	if s.Status.IsSet {
		switch s.Status.Value {
		case v1.Limited:
			val.Status = BaselineStatusLimited
		case v1.Newly:
			val.Status = BaselineStatusNewly
		case v1.Widely:
			val.Status = BaselineStatusWidely
		}
	}

	if s.LowDate.IsSet {
		val.LowDate = s.LowDate.Value
	}
	if s.HighDate.IsSet {
		val.HighDate = s.HighDate.Value
	}

	return val
}

func toBrowserValue(s v1.BrowserState) BrowserValue {
	val := BrowserValue{
		Status:  BrowserStatusUnknown,
		Version: nil,
	}
	if s.Status.IsSet {
		switch s.Status.Value {
		case v1.Available:
			val.Status = BrowserStatusAvailable
		case v1.Unavailable:
			val.Status = BrowserStatusUnavailable
		}
	}
	if s.Version.IsSet {
		val.Version = s.Version.Value
	}

	return val
}

type Reason string

const (
	ReasonQueryChanged = "QUERY_CHANGED"
	ReasonDataUpdated  = "DATA_UPDATED"
)

type PublishEventRequest struct {
	EventID       string
	StateID       string
	StateBlobPath string
	DiffID        string
	DiffBlobPath  string
	SearchID      string
	Query         string
	Summary       []byte
	Reasons       []Reason
	Frequency     JobFrequency
	GeneratedAt   time.Time
}

type LatestEventInfo struct {
	EventID       string
	StateBlobPath string
}

// JobFrequency defines how often a saved search should be checked.
type JobFrequency string

const (
	FrequencyUnknown   JobFrequency = "UNKNOWN"
	FrequencyImmediate JobFrequency = "IMMEDIATE"
	FrequencyWeekly    JobFrequency = "WEEKLY"
	FrequencyMonthly   JobFrequency = "MONTHLY"
)

type RefreshSearchCommand struct {
	SearchID  string
	Query     string
	Frequency JobFrequency
	Timestamp time.Time
}

type SearchJob struct {
	ID    string
	Query string
}

// EmailSubscriber represents a subscriber using an Email channel.
type EmailSubscriber struct {
	SubscriptionID string
	UserID         string
	Triggers       []string
	EmailAddress   string
}

// SubscriberSet groups subscribers by channel type to avoid runtime type assertions.
type SubscriberSet struct {
	Emails []EmailSubscriber
	// Future channel types (e.g. Webhook) can be added here.
}

// DeliveryMetadata contains the necessary context from the original event
// required for rendering notifications (e.g. generating links), decoupled from the upstream event format.
type DeliveryMetadata struct {
	EventID     string
	SearchID    string
	Query       string
	Frequency   JobFrequency
	GeneratedAt time.Time
}

type DispatchEventMetadata struct {
	EventID     string
	SearchID    string
	Frequency   JobFrequency
	Query       string
	GeneratedAt time.Time
}

// EmailDeliveryJob represents a task to send an email.
type EmailDeliveryJob struct {
	SubscriptionID string
	RecipientEmail string
	// SummaryRaw is the opaque JSON payload describing the event.
	SummaryRaw []byte
	// Metadata contains context for links and tracking.
	Metadata DeliveryMetadata
}
