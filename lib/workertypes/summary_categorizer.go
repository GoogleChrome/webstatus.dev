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
// categorized summary elements. Delivery channels implement this interface to receive
// filtered and promoted categories via double-dispatch.
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

// routeHighlight assigns a highlight to its respective category in Summary,
// promoting Removed highlights to Changed when active baseline or browser updates exist.
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
