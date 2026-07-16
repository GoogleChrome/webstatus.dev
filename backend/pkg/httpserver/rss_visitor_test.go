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
	return workertypes.EventSummary{
		SchemaVersion:  workertypes.VersionEventSummaryV1,
		SnapshotOrigin: workertypes.OriginLive,
		Truncated:      false,
		Highlights:     nil,
		Text:           "Query grammar failure",
		QueryErrors: []workertypes.SummaryQueryError{
			{Code: errCode},
		},
		ResolvedQueryErrors: nil,
		Categories: workertypes.SummaryCategories{
			Updated:         0,
			Added:           0,
			Removed:         0,
			Moved:           0,
			Split:           0,
			Deleted:         0,
			UpdatedBaseline: 0,
			QueryChanged:    0,
			UpdatedImpl:     0,
			UpdatedRename:   0,
		},
	}
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
			name:        "QueryGrammar error renders human-readable message",
			errorCode:   workertypes.SummaryQueryErrorCodeQueryGrammar,
			wantMessage: "Invalid query grammar",
		},
		{
			name:        "SavedSearchNotFound error renders human-readable message",
			errorCode:   workertypes.SummaryQueryErrorCodeSavedSearchNotFound,
			wantMessage: "Saved search not found",
		},
		{
			name:        "MaxDepthExceeded error renders human-readable message",
			errorCode:   workertypes.SummaryQueryErrorCodeMaxDepthExceeded,
			wantMessage: "Saved search max depth exceeded",
		},
		{
			name:        "InvalidQuery error renders human-readable message",
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
	summary := workertypes.EventSummary{
		SchemaVersion:  workertypes.VersionEventSummaryV1,
		SnapshotOrigin: workertypes.OriginLive,
		Truncated:      false,
		Highlights:     nil,
		Text:           "Search query recovered",
		QueryErrors:    nil,
		ResolvedQueryErrors: []workertypes.SummaryQueryError{
			{Code: workertypes.SummaryQueryErrorCodeQueryGrammar},
		},
		Categories: workertypes.NewEmptySummaryCategories(),
	}

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
