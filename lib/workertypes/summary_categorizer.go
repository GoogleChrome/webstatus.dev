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

// CategorizedSummary contains pre-grouped highlights and error slices
// resulting from BaseSummaryVisitor categorization.
type CategorizedSummary struct {
	Text                string
	Truncated           bool
	QueryErrors         []SummaryQueryError
	ResolvedQueryErrors []SummaryQueryError
	Added               []SummaryHighlight
	Removed             []SummaryHighlight
	Changed             []SummaryHighlight
	Moved               []SummaryHighlight
	Split               []SummaryHighlight
	Deleted             []SummaryHighlight
}

// NewEmptyCategorizedSummary returns a zero-initialized CategorizedSummary
// satisfying exhaustruct requirements.
func NewEmptyCategorizedSummary() CategorizedSummary {
	return CategorizedSummary{
		Text:                "",
		Truncated:           false,
		QueryErrors:         nil,
		ResolvedQueryErrors: nil,
		Added:               nil,
		Removed:             nil,
		Changed:             nil,
		Moved:               nil,
		Split:               nil,
		Deleted:             nil,
	}
}

// CategorizedSummaryVisitor defines the strongly-typed contract for consuming
// categorized summary elements. Delivery channels (e.g. RSS, Email, Webhook, Slack)
// implement this interface to receive filtered and promoted categories via double-dispatch.
//
// Documenting & Enforcing a Successfully Tested Renderer:
// Standard delivery channel visitors MUST satisfy the following contract invariants and testing standards:
//
// Runtime Implementation Invariants:
//  1. Nil & Empty Slice Safety: All Visit* methods MUST safely handle nil or empty ([]T{})
//     slices without panicking or dereferencing nil pointers.
//  2. Error Propagation: Rendering failures (e.g., template execution errors) MUST be returned
//     as non-nil errors to allow BaseSummaryVisitor.Dispatch() to fail fast.
//  3. State Isolation: Visitor instances must maintain state isolation across separate dispatch passes.
//
// 5-Part Unit Testing Blueprint (Symmetrical Test Parity Standard):
// Package unit tests for new delivery channels MUST implement the 5 symmetrical test suites:
//  1. Test<Channel>_FeatureCategories: Table-driven tests covering all 6 categories
//     (Added, Removed, Changed, Moved, Split, Deleted).
//  2. Test<Channel>_QueryErrors_RenderMessage: Table-driven tests covering all SummaryQueryErrorCode enums.
//  3. Test<Channel>_TriggerFiltering: Verifying highlight filtering by subscriber triggers.
//  4. Test<Channel>_NilPointerGuards: Verifying zero panics when handling optional diff structs (Moved/Split = nil).
//  5. Test<Channel>_Golden: Output regression testing using .golden snapshot files and cmp.Diff.
type CategorizedSummaryVisitor interface {
	VisitQueryErrors(errors []SummaryQueryError) error
	VisitResolvedQueryErrors(errors []SummaryQueryError) error
	VisitAddedFeatures(features []SummaryHighlight) error
	VisitRemovedFeatures(features []SummaryHighlight) error
	VisitChangedFeatures(features []SummaryHighlight) error
	VisitMovedFeatures(features []SummaryHighlight) error
	VisitSplitFeatures(features []SummaryHighlight) error
	VisitDeletedFeatures(features []SummaryHighlight) error
}

// BaseSummaryVisitor implements SummaryVisitor and centralizes highlight filtering,
// category grouping, promotion logic, and double-dispatching.
// BaseSummaryVisitor is stateful and is NOT safe for concurrent use across goroutines.
// Create a new instance per EventSummary categorization pass.
type BaseSummaryVisitor struct {
	triggers []JobTrigger
	target   CategorizedSummaryVisitor
	Summary  CategorizedSummary
}

// newBaseSummaryVisitor constructs a new BaseSummaryVisitor with the specified triggers
// and optional CategorizedSummaryVisitor dispatch target.
func newBaseSummaryVisitor(triggers []JobTrigger, target CategorizedSummaryVisitor) *BaseSummaryVisitor {
	return &BaseSummaryVisitor{
		triggers: triggers,
		target:   target,
		Summary:  NewEmptyCategorizedSummary(),
	}
}

// VisitV1 processes an EventSummary, filters highlights by triggers, categorizes them,
// applies promotion rules, and dispatches to the target visitor if provided.
func (v *BaseSummaryVisitor) VisitV1(s EventSummary) error {
	v.Summary = NewEmptyCategorizedSummary()
	v.Summary.Text = s.Text
	v.Summary.Truncated = s.Truncated
	v.Summary.QueryErrors = s.QueryErrors
	v.Summary.ResolvedQueryErrors = s.ResolvedQueryErrors

	filtered := FilterHighlights(s.Highlights, v.triggers)
	for _, h := range filtered {
		v.routeHighlight(h)
	}

	if v.target != nil {
		return v.dispatch()
	}

	return nil
}

// Dispatch executes double-dispatch on the provided CategorizedSummaryVisitor target.
func (v *BaseSummaryVisitor) Dispatch(target CategorizedSummaryVisitor) error {
	if target == nil {
		return nil
	}
	v.target = target

	return v.dispatch()
}

// routeHighlight assigns a highlight to its respective category in Summary.
// Note: Highlights with Type "Removed" represent features no longer matching query criteria.
// If a Removed highlight also contains active baseline or browser implementation updates
// (h.BaselineChange != nil || len(h.BrowserChanges) > 0), it is promoted to Changed so
// UI renderers display the detailed browser status change rather than a pure query removal.
func (v *BaseSummaryVisitor) routeHighlight(h SummaryHighlight) {
	switch h.Type {
	case SummaryHighlightTypeAdded:
		v.Summary.Added = append(v.Summary.Added, h)
	case SummaryHighlightTypeRemoved:
		if h.BaselineChange != nil || len(h.BrowserChanges) > 0 {
			v.Summary.Changed = append(v.Summary.Changed, h)
		} else {
			v.Summary.Removed = append(v.Summary.Removed, h)
		}
	case SummaryHighlightTypeChanged:
		v.Summary.Changed = append(v.Summary.Changed, h)
	case SummaryHighlightTypeMoved:
		v.Summary.Moved = append(v.Summary.Moved, h)
	case SummaryHighlightTypeSplit:
		v.Summary.Split = append(v.Summary.Split, h)
	case SummaryHighlightTypeDeleted:
		v.Summary.Deleted = append(v.Summary.Deleted, h)
	}
}

func (v *BaseSummaryVisitor) dispatch() error {
	if len(v.Summary.QueryErrors) > 0 {
		if err := v.target.VisitQueryErrors(v.Summary.QueryErrors); err != nil {
			return err
		}
	}
	if len(v.Summary.ResolvedQueryErrors) > 0 {
		if err := v.target.VisitResolvedQueryErrors(v.Summary.ResolvedQueryErrors); err != nil {
			return err
		}
	}
	if len(v.Summary.Added) > 0 {
		if err := v.target.VisitAddedFeatures(v.Summary.Added); err != nil {
			return err
		}
	}
	if len(v.Summary.Removed) > 0 {
		if err := v.target.VisitRemovedFeatures(v.Summary.Removed); err != nil {
			return err
		}
	}
	if len(v.Summary.Changed) > 0 {
		if err := v.target.VisitChangedFeatures(v.Summary.Changed); err != nil {
			return err
		}
	}
	if len(v.Summary.Moved) > 0 {
		if err := v.target.VisitMovedFeatures(v.Summary.Moved); err != nil {
			return err
		}
	}
	if len(v.Summary.Split) > 0 {
		if err := v.target.VisitSplitFeatures(v.Summary.Split); err != nil {
			return err
		}
	}
	if len(v.Summary.Deleted) > 0 {
		if err := v.target.VisitDeletedFeatures(v.Summary.Deleted); err != nil {
			return err
		}
	}

	return nil
}

// HasContent returns true if any query errors or categorized highlights are present.
func (v *BaseSummaryVisitor) HasContent() bool {
	return len(v.Summary.QueryErrors) > 0 ||
		len(v.Summary.ResolvedQueryErrors) > 0 ||
		len(v.Summary.Added) > 0 ||
		len(v.Summary.Removed) > 0 ||
		len(v.Summary.Changed) > 0 ||
		len(v.Summary.Moved) > 0 ||
		len(v.Summary.Split) > 0 ||
		len(v.Summary.Deleted) > 0
}
