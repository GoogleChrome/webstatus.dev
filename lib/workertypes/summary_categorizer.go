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

// CategorizedSummary holds the categorized highlights and query errors of an EventSummary
// after filtering against a subscriber's triggers.
type CategorizedSummary struct {
	SummaryText         string
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

// CategorizeEventSummary filters the summary highlights against the given triggers,
// groups them by their change category, and extracts query errors.
func CategorizeEventSummary(summary EventSummary, triggers []JobTrigger) CategorizedSummary {
	filtered := FilterHighlights(summary.Highlights, triggers)
	if len(filtered) != 0 {
		summary.Highlights = filtered
	}

	categorized := CategorizedSummary{
		SummaryText:         summary.Text,
		Truncated:           summary.Truncated,
		QueryErrors:         summary.QueryErrors,
		ResolvedQueryErrors: summary.ResolvedQueryErrors,
		Added:               nil,
		Removed:             nil,
		Changed:             nil,
		Moved:               nil,
		Split:               nil,
		Deleted:             nil,
	}

	for _, h := range summary.Highlights {
		switch h.Type {
		case SummaryHighlightTypeAdded:
			categorized.Added = append(categorized.Added, h)
		case SummaryHighlightTypeRemoved:
			categorized.Removed = append(categorized.Removed, h)
		case SummaryHighlightTypeChanged:
			categorized.Changed = append(categorized.Changed, h)
		case SummaryHighlightTypeMoved:
			categorized.Moved = append(categorized.Moved, h)
		case SummaryHighlightTypeSplit:
			categorized.Split = append(categorized.Split, h)
		case SummaryHighlightTypeDeleted:
			categorized.Deleted = append(categorized.Deleted, h)
		}
	}

	return categorized
}

// HasContent returns true if the summary contains any categorized highlights or query errors.
// This ensures notifications and RSS feeds are not dropped when highlights are empty but
// critical query errors (e.g. invalid query grammar) occurred or recovery was completed.
func (c *CategorizedSummary) HasContent() bool {
	return len(c.QueryErrors) > 0 ||
		len(c.ResolvedQueryErrors) > 0 ||
		len(c.Added) > 0 ||
		len(c.Removed) > 0 ||
		len(c.Changed) > 0 ||
		len(c.Moved) > 0 ||
		len(c.Split) > 0 ||
		len(c.Deleted) > 0
}

// Other returns all non-Added and non-Removed highlights (Changed, Moved, Split, Deleted)
// in a single slice, which is useful for simple renderers like RSS that group updates into "Other".
func (c *CategorizedSummary) Other() []SummaryHighlight {
	total := len(c.Changed) + len(c.Moved) + len(c.Split) + len(c.Deleted)
	if total == 0 {
		return nil
	}

	other := make([]SummaryHighlight, 0, total)
	other = append(other, c.Changed...)
	other = append(other, c.Moved...)
	other = append(other, c.Split...)
	other = append(other, c.Deleted...)

	return other
}

// BaseSummaryVisitor implements SummaryVisitor and populates a CategorizedSummary.
// Concrete visitors (like rssVisitor or templateDataGenerator) can embed this struct,
// pass triggers at construction time, and call BaseSummaryVisitor.VisitV1(summary) inside their VisitV1.
type BaseSummaryVisitor struct {
	Triggers    []JobTrigger
	Categorized CategorizedSummary
}

func newEmptyCategorizedSummary() CategorizedSummary {
	return CategorizedSummary{
		SummaryText:         "",
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

// NewBaseSummaryVisitor creates a new BaseSummaryVisitor configured with the given triggers.
func NewBaseSummaryVisitor(triggers []JobTrigger) BaseSummaryVisitor {
	return BaseSummaryVisitor{
		Triggers:    triggers,
		Categorized: newEmptyCategorizedSummary(),
	}
}

// VisitV1 categorizes the incoming V1 summary and stores the result.
func (b *BaseSummaryVisitor) VisitV1(summary EventSummary) error {
	b.Categorized = CategorizeEventSummary(summary, b.Triggers)

	return nil
}
