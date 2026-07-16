// Copyright 2026 Google LLC
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
	"testing"
)

type categorizeTestCase struct {
	name            string
	summary         EventSummary
	triggers        []JobTrigger
	wantHasContent  bool
	wantQueryErrors int
	wantAdded       int
	wantRemoved     int
	wantChanged     int
	wantMoved       int
	wantSplit       int
	wantDeleted     int
	wantOther       int
}

func newTestSummaryCategories() SummaryCategories {
	return *new(SummaryCategories)
}

func newTestBaselineValue(status BaselineStatus) BaselineValue {
	bv := new(BaselineValue)
	bv.Status = status

	return *bv
}

func newTestEventSummary(
	text string,
	truncated bool,
	qErrs []SummaryQueryError,
	highlights []SummaryHighlight,
) EventSummary {
	return EventSummary{
		SchemaVersion:       VersionEventSummaryV1,
		SnapshotOrigin:      OriginLive,
		Text:                text,
		Categories:          newTestSummaryCategories(),
		Truncated:           truncated,
		QueryErrors:         qErrs,
		ResolvedQueryErrors: nil,
		Highlights:          highlights,
	}
}

func newTestSummaryHighlight(
	typ SummaryHighlightType,
	id, name string,
	change *Change[BaselineValue],
) SummaryHighlight {
	return SummaryHighlight{
		Type:           typ,
		FeatureID:      id,
		FeatureName:    name,
		Docs:           nil,
		NameChange:     nil,
		BaselineChange: change,
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	}
}

func runCategorizeTestCase(t *testing.T, tc categorizeTestCase) {
	t.Helper()
	got := CategorizeEventSummary(tc.summary, tc.triggers)
	if got.SummaryText != tc.summary.Text {
		t.Errorf("SummaryText = %q, want %q", got.SummaryText, tc.summary.Text)
	}
	if got.Truncated != tc.summary.Truncated {
		t.Errorf("Truncated = %v, want %v", got.Truncated, tc.summary.Truncated)
	}
	if got.HasContent() != tc.wantHasContent {
		t.Errorf("HasContent() = %v, want %v", got.HasContent(), tc.wantHasContent)
	}
	if len(got.QueryErrors) != tc.wantQueryErrors {
		t.Errorf("len(QueryErrors) = %d, want %d", len(got.QueryErrors), tc.wantQueryErrors)
	}
	if len(got.Added) != tc.wantAdded {
		t.Errorf("len(Added) = %d, want %d", len(got.Added), tc.wantAdded)
	}
	if len(got.Removed) != tc.wantRemoved {
		t.Errorf("len(Removed) = %d, want %d", len(got.Removed), tc.wantRemoved)
	}
	if len(got.Changed) != tc.wantChanged {
		t.Errorf("len(Changed) = %d, want %d", len(got.Changed), tc.wantChanged)
	}
	if len(got.Moved) != tc.wantMoved {
		t.Errorf("len(Moved) = %d, want %d", len(got.Moved), tc.wantMoved)
	}
	if len(got.Split) != tc.wantSplit {
		t.Errorf("len(Split) = %d, want %d", len(got.Split), tc.wantSplit)
	}
	if len(got.Deleted) != tc.wantDeleted {
		t.Errorf("len(Deleted) = %d, want %d", len(got.Deleted), tc.wantDeleted)
	}
	if len(got.Other()) != tc.wantOther {
		t.Errorf("len(Other()) = %d, want %d", len(got.Other()), tc.wantOther)
	}
}

func TestCategorizeEventSummary_Basic(t *testing.T) {
	tests := []categorizeTestCase{
		{
			name: "Empty Summary with Query Errors",
			summary: newTestEventSummary("Error occurred", false, []SummaryQueryError{
				{Code: SummaryQueryErrorCodeQueryGrammar},
			}, nil),
			triggers:        nil,
			wantHasContent:  true,
			wantQueryErrors: 1,
			wantAdded:       0,
			wantRemoved:     0,
			wantChanged:     0,
			wantMoved:       0,
			wantSplit:       0,
			wantDeleted:     0,
			wantOther:       0,
		},
		{
			name: "All Categories Without Triggers",
			summary: newTestEventSummary("Summary text", true, nil, []SummaryHighlight{
				newTestSummaryHighlight(SummaryHighlightTypeAdded, "1", "feat-added", nil),
				newTestSummaryHighlight(SummaryHighlightTypeRemoved, "2", "feat-removed", nil),
				newTestSummaryHighlight(SummaryHighlightTypeChanged, "3", "feat-changed", nil),
				newTestSummaryHighlight(SummaryHighlightTypeMoved, "4", "feat-moved", nil),
				newTestSummaryHighlight(SummaryHighlightTypeSplit, "5", "feat-split", nil),
				newTestSummaryHighlight(SummaryHighlightTypeDeleted, "6", "feat-deleted", nil),
			}),
			triggers:        nil,
			wantHasContent:  true,
			wantQueryErrors: 0,
			wantAdded:       1,
			wantRemoved:     1,
			wantChanged:     1,
			wantMoved:       1,
			wantSplit:       1,
			wantDeleted:     1,
			wantOther:       4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCategorizeTestCase(t, tc)
		})
	}
}

func TestCategorizeEventSummary_Triggers(t *testing.T) {
	tests := []categorizeTestCase{
		{
			name: "Trigger Filtering Promoted To Newly",
			summary: newTestEventSummary("Summary text", false, nil, []SummaryHighlight{
				newTestSummaryHighlight(SummaryHighlightTypeAdded, "1", "feat-added", nil),
				newTestSummaryHighlight(SummaryHighlightTypeChanged, "2", "feat-newly", &Change[BaselineValue]{
					From: newTestBaselineValue(BaselineStatusLimited),
					To:   newTestBaselineValue(BaselineStatusNewly),
				}),
			}),
			triggers: []JobTrigger{
				FeaturePromotedToNewly,
			},
			wantHasContent:  true,
			wantQueryErrors: 0,
			wantAdded:       0,
			wantRemoved:     0,
			wantChanged:     1,
			wantMoved:       0,
			wantSplit:       0,
			wantDeleted:     0,
			wantOther:       1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCategorizeTestCase(t, tc)
		})
	}
}

func TestBaseSummaryVisitor(t *testing.T) {
	visitor := NewBaseSummaryVisitor([]JobTrigger{FeaturePromotedToNewly})
	summary := newTestEventSummary("Visitor summary", false, nil, []SummaryHighlight{
		newTestSummaryHighlight(SummaryHighlightTypeChanged, "feat-1", "Baseline newly promoted", &Change[BaselineValue]{
			From: newTestBaselineValue(BaselineStatusLimited),
			To:   newTestBaselineValue(BaselineStatusNewly),
		}),
	})

	if err := visitor.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error: %v", err)
	}

	if visitor.Categorized.SummaryText != "Visitor summary" {
		t.Errorf("SummaryText = %q, want %q", visitor.Categorized.SummaryText, "Visitor summary")
	}
	if len(visitor.Categorized.Changed) != 1 {
		t.Errorf("len(Changed) = %d, want 1", len(visitor.Categorized.Changed))
	}
}
