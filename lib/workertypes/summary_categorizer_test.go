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
	"errors"
	"testing"
)

type mockCategorizedVisitor struct {
	queryErrorsVisited         int
	resolvedQueryErrorsVisited int
	addedVisited               int
	removedVisited             int
	changedVisited             int
	movedVisited               int
	splitVisited               int
	deletedVisited             int
	errToReturn                error
}

func newMockCategorizedVisitor() *mockCategorizedVisitor {
	return &mockCategorizedVisitor{
		queryErrorsVisited:         0,
		resolvedQueryErrorsVisited: 0,
		addedVisited:               0,
		removedVisited:             0,
		changedVisited:             0,
		movedVisited:               0,
		splitVisited:               0,
		deletedVisited:             0,
		errToReturn:                nil,
	}
}

func (m *mockCategorizedVisitor) VisitQueryErrors(_ []SummaryQueryError) error {
	m.queryErrorsVisited++

	return m.errToReturn
}

func (m *mockCategorizedVisitor) VisitResolvedQueryErrors(_ []SummaryQueryError) error {
	m.resolvedQueryErrorsVisited++

	return m.errToReturn
}

func (m *mockCategorizedVisitor) VisitAddedFeatures(_ []SummaryHighlight) error {
	m.addedVisited++

	return m.errToReturn
}

func (m *mockCategorizedVisitor) VisitRemovedFeatures(_ []SummaryHighlight) error {
	m.removedVisited++

	return m.errToReturn
}

func (m *mockCategorizedVisitor) VisitChangedFeatures(_ []SummaryHighlight) error {
	m.changedVisited++

	return m.errToReturn
}

func (m *mockCategorizedVisitor) VisitMovedFeatures(_ []SummaryHighlight) error {
	m.movedVisited++

	return m.errToReturn
}

func (m *mockCategorizedVisitor) VisitSplitFeatures(_ []SummaryHighlight) error {
	m.splitVisited++

	return m.errToReturn
}

func (m *mockCategorizedVisitor) VisitDeletedFeatures(_ []SummaryHighlight) error {
	m.deletedVisited++

	return m.errToReturn
}

func TestBaseSummaryVisitor_CategorizationAndPromotion(t *testing.T) {
	summary := NewEmptyEventSummary()
	summary.Text = "Production Check"
	summary.QueryErrors = []SummaryQueryError{
		{Code: SummaryQueryErrorCodeInvalidQuery},
	}
	summary.ResolvedQueryErrors = []SummaryQueryError{
		{Code: SummaryQueryErrorCodeQueryGrammar},
	}
	summary.Highlights = []SummaryHighlight{
		{
			Type:        SummaryHighlightTypeAdded,
			FeatureID:   "f-1",
			FeatureName: "Feature One",
			Docs:        nil,
			NameChange:  nil,
			BaselineChange: &Change[BaselineValue]{
				From: BaselineValue{Status: BaselineStatusLimited, LowDate: nil, HighDate: nil},
				To:   BaselineValue{Status: BaselineStatusWidely, LowDate: nil, HighDate: nil},
			},
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
		{
			Type:        SummaryHighlightTypeRemoved,
			FeatureID:   "f-2",
			FeatureName: "Feature Two (Promoted)",
			Docs:        nil,
			NameChange:  nil,
			BaselineChange: &Change[BaselineValue]{
				From: BaselineValue{Status: BaselineStatusWidely, LowDate: nil, HighDate: nil},
				To:   BaselineValue{Status: BaselineStatusLimited, LowDate: nil, HighDate: nil},
			},
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
		{
			Type:           SummaryHighlightTypeRemoved,
			FeatureID:      "f-3",
			FeatureName:    "Feature Three (Unchanged Removed)",
			Docs:           nil,
			NameChange:     nil,
			BaselineChange: nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
	}

	triggers := []JobTrigger{FeaturePromotedToWidely, FeatureRegressedToLimited}
	mock := newMockCategorizedVisitor()
	visitor := newBaseSummaryVisitor(triggers, mock)

	if err := visitor.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error: %v", err)
	}

	if !visitor.HasContent() {
		t.Error("expected HasContent() to be true")
	}

	if len(visitor.Summary.Added) != 1 || visitor.Summary.Added[0].FeatureID != "f-1" {
		t.Errorf("expected 1 Added feature f-1, got %v", visitor.Summary.Added)
	}
	if len(visitor.Summary.Changed) != 1 || visitor.Summary.Changed[0].FeatureID != "f-2" {
		t.Errorf("expected 1 Changed feature f-2 via promotion, got %v", visitor.Summary.Changed)
	}
	if len(visitor.Summary.Removed) != 0 {
		t.Errorf("expected 0 Removed features (f-3 filtered out by triggers), got %v", visitor.Summary.Removed)
	}

	if mock.queryErrorsVisited != 1 {
		t.Errorf("expected query errors to be visited once, got %d", mock.queryErrorsVisited)
	}
	if mock.resolvedQueryErrorsVisited != 1 {
		t.Errorf("expected resolved query errors to be visited once, got %d", mock.resolvedQueryErrorsVisited)
	}
	if mock.addedVisited != 1 {
		t.Errorf("expected added features to be visited once, got %d", mock.addedVisited)
	}
	if mock.changedVisited != 1 {
		t.Errorf("expected changed features to be visited once, got %d", mock.changedVisited)
	}
}

func TestBaseSummaryVisitor_ErrorDispatchPropagation(t *testing.T) {
	summary := NewEmptyEventSummary()
	summary.QueryErrors = []SummaryQueryError{
		{Code: SummaryQueryErrorCodeInvalidQuery},
	}

	mock := newMockCategorizedVisitor()
	expectedErr := errors.New("dispatch failure")
	mock.errToReturn = expectedErr

	visitor := newBaseSummaryVisitor(nil, mock)
	err := visitor.VisitV1(summary)
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestBaseSummaryVisitor_HasContent_Empty(t *testing.T) {
	summary := NewEmptyEventSummary()
	visitor := newBaseSummaryVisitor(nil, nil)
	if err := visitor.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error: %v", err)
	}
	if visitor.HasContent() {
		t.Error("expected HasContent() to be false on empty summary")
	}
}

func newTestHighlight(typ SummaryHighlightType, id, name string) SummaryHighlight {
	return SummaryHighlight{
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

func TestBaseSummaryVisitor_AllCategoryRoutes(t *testing.T) {
	summary := NewEmptyEventSummary()
	summary.AddHighlight(newTestHighlight(SummaryHighlightTypeMoved, "f-m", "Moved"))
	summary.AddHighlight(newTestHighlight(SummaryHighlightTypeSplit, "f-s", "Split"))
	summary.AddHighlight(newTestHighlight(SummaryHighlightTypeDeleted, "f-d", "Deleted"))

	mock := newMockCategorizedVisitor()
	visitor := newBaseSummaryVisitor(nil, mock)

	if err := visitor.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error: %v", err)
	}

	if len(visitor.Summary.Moved) != 1 || visitor.Summary.Moved[0].FeatureID != "f-m" {
		t.Errorf("Moved routing failed: %v", visitor.Summary.Moved)
	}
	if len(visitor.Summary.Split) != 1 || visitor.Summary.Split[0].FeatureID != "f-s" {
		t.Errorf("Split routing failed: %v", visitor.Summary.Split)
	}
	if len(visitor.Summary.Deleted) != 1 || visitor.Summary.Deleted[0].FeatureID != "f-d" {
		t.Errorf("Deleted routing failed: %v", visitor.Summary.Deleted)
	}

	if mock.movedVisited != 1 || mock.splitVisited != 1 || mock.deletedVisited != 1 {
		t.Errorf("Dispatch failed for Moved/Split/Deleted: moved=%d, split=%d, deleted=%d",
			mock.movedVisited, mock.splitVisited, mock.deletedVisited)
	}
}

func TestBaseSummaryVisitor_AllHighlightsFilteredOut(t *testing.T) {
	summary := NewEmptyEventSummary()
	summary.AddHighlight(newTestHighlight(SummaryHighlightTypeAdded, "f-1", "Unmatched Feature"))

	triggers := []JobTrigger{FeaturePromotedToWidely}

	mock1 := newMockCategorizedVisitor()
	v1 := newBaseSummaryVisitor(triggers, mock1)
	if err := v1.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error: %v", err)
	}
	if v1.HasContent() {
		t.Error("expected HasContent() to be false when all highlights are filtered and no errors exist")
	}
	if mock1.addedVisited != 0 {
		t.Errorf("expected 0 addedVisited when filtered out, got %d", mock1.addedVisited)
	}

	summary.SetQueryErrors([]SummaryQueryError{{Code: SummaryQueryErrorCodeInvalidQuery}})
	mock2 := newMockCategorizedVisitor()
	v2 := newBaseSummaryVisitor(triggers, mock2)
	if err := v2.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error: %v", err)
	}
	if !v2.HasContent() {
		t.Error("expected HasContent() to be true when QueryErrors exist even if highlights are filtered out")
	}
	if mock2.queryErrorsVisited != 1 || mock2.addedVisited != 0 {
		t.Errorf("expected queryErrorsVisited=1 and addedVisited=0, got qe=%d, added=%d",
			mock2.queryErrorsVisited, mock2.addedVisited)
	}
}
