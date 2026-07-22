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

package httpserver

import (
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

func newTestSummaryWithErrors(errCode workertypes.SummaryQueryErrorCode) workertypes.EventSummary {
	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginLive
	summary.Text = "Query grammar failure"
	summary.SetQueryErrors([]workertypes.SummaryQueryError{{Code: errCode}})

	return summary
}

func TestRSSVisitor_QueryErrors(t *testing.T) {
	visitor := newRSSVisitor([]workertypes.JobTrigger{workertypes.FeaturePromotedToNewly})
	summary := newTestSummaryWithErrors(workertypes.SummaryQueryErrorCodeQueryGrammar)

	if err := visitor.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error: %v", err)
	}
	if !visitor.HasContent() {
		t.Error("HasContent() = false, want true when QueryErrors exist")
	}
	if len(visitor.data.QueryErrors) != 1 ||
		visitor.data.QueryErrors[0] != workertypes.SummaryQueryErrorCodeQueryGrammar.Message() {
		t.Errorf("data.QueryErrors = %v, want [%s]",
			visitor.data.QueryErrors, workertypes.SummaryQueryErrorCodeQueryGrammar.Message())
	}
}

func TestRSSVisitor_QueryErrors_RenderMessage(t *testing.T) {
	testCases := []struct {
		name        string
		errorCode   workertypes.SummaryQueryErrorCode
		wantMessage string
	}{
		{
			name:        "QueryGrammar error",
			errorCode:   workertypes.SummaryQueryErrorCodeQueryGrammar,
			wantMessage: "Invalid query grammar",
		},
		{
			name:        "SavedSearchNotFound error",
			errorCode:   workertypes.SummaryQueryErrorCodeSavedSearchNotFound,
			wantMessage: "Saved search not found",
		},
		{
			name:        "MaxDepthExceeded error",
			errorCode:   workertypes.SummaryQueryErrorCodeMaxDepthExceeded,
			wantMessage: "Saved search max depth exceeded",
		},
		{
			name:        "InvalidQuery error",
			errorCode:   workertypes.SummaryQueryErrorCodeInvalidQuery,
			wantMessage: "Invalid query",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			visitor := newRSSVisitor([]workertypes.JobTrigger{workertypes.FeaturePromotedToNewly})
			summary := newTestSummaryWithErrors(tc.errorCode)

			if err := visitor.VisitV1(summary); err != nil {
				t.Fatalf("VisitV1 unexpected error: %v", err)
			}

			if !visitor.HasContent() {
				t.Error("HasContent() = false, want true when QueryErrors exist")
			}

			if len(visitor.data.QueryErrors) != 1 {
				t.Fatalf("data.QueryErrors count = %d, want 1", len(visitor.data.QueryErrors))
			}

			if visitor.data.QueryErrors[0] != tc.wantMessage {
				t.Errorf("visitor.data.QueryErrors[0] = %q, want %q", visitor.data.QueryErrors[0], tc.wantMessage)
			}
		})
	}
}

func TestRSSVisitor_ResolvedQueryErrors(t *testing.T) {
	visitor := newRSSVisitor([]workertypes.JobTrigger{workertypes.FeaturePromotedToNewly})
	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginLive
	summary.Text = "Search query recovered"
	summary.SetResolvedQueryErrors([]workertypes.SummaryQueryError{{Code: workertypes.SummaryQueryErrorCodeQueryGrammar}})

	if err := visitor.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error: %v", err)
	}
	if !visitor.HasContent() {
		t.Error("HasContent() = false, want true when ResolvedQueryErrors exist")
	}
	if len(visitor.data.ResolvedQueryErrors) != 1 ||
		visitor.data.ResolvedQueryErrors[0] != workertypes.SummaryQueryErrorCodeQueryGrammar.Message() {
		t.Errorf("data.ResolvedQueryErrors = %v, want [%s]",
			visitor.data.ResolvedQueryErrors, workertypes.SummaryQueryErrorCodeQueryGrammar.Message())
	}
}

func newTestHighlight(typ workertypes.SummaryHighlightType, id, name string) workertypes.SummaryHighlight {
	return workertypes.SummaryHighlight{
		Type:           typ,
		FeatureID:      id,
		FeatureName:    name,
		Docs:           nil,
		NameChange:     nil,
		BaselineChange: nil,
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	}
}

func TestRSSVisitor_FeatureCategories(t *testing.T) {
	testCases := []struct {
		name      string
		highlight workertypes.SummaryHighlight
		checkFunc func(*testing.T, *rssVisitor)
	}{
		{
			name:      "Added feature",
			highlight: newTestHighlight(workertypes.SummaryHighlightTypeAdded, "f-added", "Added Feature"),
			checkFunc: func(t *testing.T, v *rssVisitor) {
				if len(v.data.Added) != 1 || v.data.Added[0] != "Added Feature" {
					t.Errorf("v.data.Added = %v, want ['Added Feature']", v.data.Added)
				}
			},
		},
		{
			name:      "Removed feature",
			highlight: newTestHighlight(workertypes.SummaryHighlightTypeRemoved, "f-removed", "Removed Feature"),
			checkFunc: func(t *testing.T, v *rssVisitor) {
				if len(v.data.Removed) != 1 || v.data.Removed[0] != "Removed Feature" {
					t.Errorf("v.data.Removed = %v, want ['Removed Feature']", v.data.Removed)
				}
			},
		},
		{
			name:      "Changed feature",
			highlight: newTestHighlight(workertypes.SummaryHighlightTypeChanged, "f-changed", "Changed Feature"),
			checkFunc: func(t *testing.T, v *rssVisitor) {
				if len(v.data.Changed) != 1 || v.data.Changed[0] != "Changed Feature" {
					t.Errorf("v.data.Changed = %v, want ['Changed Feature']", v.data.Changed)
				}
			},
		},
		{
			name:      "Moved feature",
			highlight: newTestHighlight(workertypes.SummaryHighlightTypeMoved, "f-moved", "Moved Feature"),
			checkFunc: func(t *testing.T, v *rssVisitor) {
				if len(v.data.Moved) != 1 || v.data.Moved[0] != "Moved Feature" {
					t.Errorf("v.data.Moved = %v, want ['Moved Feature']", v.data.Moved)
				}
			},
		},
		{
			name:      "Split feature",
			highlight: newTestHighlight(workertypes.SummaryHighlightTypeSplit, "f-split", "Split Feature"),
			checkFunc: func(t *testing.T, v *rssVisitor) {
				if len(v.data.Split) != 1 || v.data.Split[0] != "Split Feature" {
					t.Errorf("v.data.Split = %v, want ['Split Feature']", v.data.Split)
				}
			},
		},
		{
			name:      "Deleted feature",
			highlight: newTestHighlight(workertypes.SummaryHighlightTypeDeleted, "f-deleted", "Deleted Feature"),
			checkFunc: func(t *testing.T, v *rssVisitor) {
				if len(v.data.Deleted) != 1 || v.data.Deleted[0] != "Deleted Feature" {
					t.Errorf("v.data.Deleted = %v, want ['Deleted Feature']", v.data.Deleted)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			visitor := newRSSVisitor(nil)
			summary := workertypes.NewEmptyEventSummary()
			summary.AddHighlight(tc.highlight)

			if err := visitor.VisitV1(summary); err != nil {
				t.Fatalf("VisitV1 unexpected error: %v", err)
			}
			if !visitor.HasContent() {
				t.Error("expected HasContent() to be true")
			}
			tc.checkFunc(t, visitor)
		})
	}
}

func TestRSSVisitor_CombinedErrorsAndFeatures(t *testing.T) {
	visitor := newRSSVisitor(nil)
	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginLive
	summary.Text = "Combined errors and features"
	summary.SetQueryErrors([]workertypes.SummaryQueryError{{Code: workertypes.SummaryQueryErrorCodeSavedSearchNotFound}})
	summary.SetResolvedQueryErrors([]workertypes.SummaryQueryError{{Code: workertypes.SummaryQueryErrorCodeQueryGrammar}})
	summary.AddHighlight(newTestHighlight(workertypes.SummaryHighlightTypeAdded, "f-added", "Subgrid"))

	if err := visitor.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error: %v", err)
	}
	if !visitor.HasContent() {
		t.Error("expected HasContent() to be true")
	}
	if len(visitor.data.QueryErrors) != 1 || len(visitor.data.ResolvedQueryErrors) != 1 || len(visitor.data.Added) != 1 {
		t.Errorf("got QueryErrors=%d, ResolvedQueryErrors=%d, Added=%d; want 1 each",
			len(visitor.data.QueryErrors), len(visitor.data.ResolvedQueryErrors), len(visitor.data.Added))
	}
}

func newTestChangedHighlight(id, name string, status workertypes.BaselineStatus) workertypes.SummaryHighlight {
	return workertypes.SummaryHighlight{
		Type:        workertypes.SummaryHighlightTypeChanged,
		FeatureID:   id,
		FeatureName: name,
		Docs:        nil,
		NameChange:  nil,
		BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
			From: workertypes.BaselineValue{
				Status:   workertypes.BaselineStatusNewly,
				LowDate:  nil,
				HighDate: nil,
			},
			To: workertypes.BaselineValue{
				Status:   status,
				LowDate:  nil,
				HighDate: nil,
			},
		},
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	}
}

func TestRSSVisitor_TriggerFiltering(t *testing.T) {
	// Subscription only wants FeaturePromotedToNewly
	visitor := newRSSVisitor([]workertypes.JobTrigger{workertypes.FeaturePromotedToNewly})
	summary := workertypes.NewEmptyEventSummary()
	summary.AddHighlight(newTestChangedHighlight("f-widely", "Widely Available Feature", workertypes.BaselineStatusWidely))

	if err := visitor.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error: %v", err)
	}
	if visitor.HasContent() {
		t.Error("expected HasContent() to be false when highlight is filtered out by triggers")
	}
}

func TestRSSVisitor_NilPointerGuards(t *testing.T) {
	visitor := newRSSVisitor(nil)
	summary := workertypes.NewEmptyEventSummary()
	summary.AddHighlight(newTestHighlight(
		workertypes.SummaryHighlightTypeMoved,
		"f-moved-nil",
		"Moved Feature With Nil Struct",
	))
	summary.AddHighlight(newTestHighlight(
		workertypes.SummaryHighlightTypeSplit,
		"f-split-nil",
		"Split Feature With Nil Struct",
	))

	if err := visitor.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error on nil pointer highlights: %v", err)
	}
	if !visitor.HasContent() {
		t.Error("expected HasContent() to be true")
	}
	if len(visitor.data.Moved) != 1 || visitor.data.Moved[0] != "Moved Feature With Nil Struct" {
		t.Errorf("visitor.data.Moved = %v, want ['Moved Feature With Nil Struct']", visitor.data.Moved)
	}
	if len(visitor.data.Split) != 1 || visitor.data.Split[0] != "Split Feature With Nil Struct" {
		t.Errorf("visitor.data.Split = %v, want ['Split Feature With Nil Struct']", visitor.data.Split)
	}
}
