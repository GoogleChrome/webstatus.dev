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
	"fmt"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

// rssVisitor implements workertypes.CategorizedSummaryVisitor to prepare RSS feed item payloads.
// It uses double dispatch via BaseSummaryVisitor to populate RSSItemData with pre-filtered,
// categorized highlights (Added, Removed, Changed, Moved, Split, Deleted) and error banners.
type rssVisitor struct {
	triggers []workertypes.JobTrigger
	data     RSSItemData
}

func newEmptyRSSItemData() RSSItemData {
	return RSSItemData{
		SummaryText:         "",
		Added:               nil,
		Removed:             nil,
		Changed:             nil,
		Moved:               nil,
		Split:               nil,
		Deleted:             nil,
		QueryErrors:         nil,
		ResolvedQueryErrors: nil,
		Truncated:           false,
	}
}

func newRSSVisitor(triggers []workertypes.JobTrigger) *rssVisitor {
	return &rssVisitor{
		triggers: triggers,
		data:     newEmptyRSSItemData(),
	}
}

// VisitV1 categorizes summary highlights against triggers and populates RSSItemData.
func (v *rssVisitor) VisitV1(summary workertypes.EventSummary) error {
	v.data = newEmptyRSSItemData()
	v.data.SummaryText = summary.Text
	v.data.Truncated = summary.Truncated

	if err := summary.Accept(v, v.triggers); err != nil {
		return fmt.Errorf("failed to categorize event summary for RSS: %w", err)
	}

	return nil
}

func (v *rssVisitor) HasContent() bool {
	return len(v.data.QueryErrors) > 0 ||
		len(v.data.ResolvedQueryErrors) > 0 ||
		len(v.data.Added) > 0 ||
		len(v.data.Removed) > 0 ||
		len(v.data.Changed) > 0 ||
		len(v.data.Moved) > 0 ||
		len(v.data.Split) > 0 ||
		len(v.data.Deleted) > 0
}

func (v *rssVisitor) VisitQueryErrors(errs []workertypes.SummaryQueryError) error {
	for _, qe := range errs {
		v.data.QueryErrors = append(v.data.QueryErrors, qe.Code.Message())
	}

	return nil
}

func (v *rssVisitor) VisitResolvedQueryErrors(errs []workertypes.SummaryQueryError) error {
	for _, qe := range errs {
		v.data.ResolvedQueryErrors = append(v.data.ResolvedQueryErrors, qe.Code.Message())
	}

	return nil
}

func (v *rssVisitor) VisitAddedFeatures(features []workertypes.SummaryHighlight) error {
	for _, h := range features {
		v.data.Added = append(v.data.Added, h.FeatureName)
	}

	return nil
}

func (v *rssVisitor) VisitRemovedFeatures(features []workertypes.SummaryHighlight) error {
	for _, h := range features {
		v.data.Removed = append(v.data.Removed, h.FeatureName)
	}

	return nil
}

func (v *rssVisitor) VisitChangedFeatures(features []workertypes.SummaryHighlight) error {
	for _, h := range features {
		v.data.Changed = append(v.data.Changed, h.FeatureName)
	}

	return nil
}

func (v *rssVisitor) VisitMovedFeatures(features []workertypes.SummaryHighlight) error {
	for _, h := range features {
		v.data.Moved = append(v.data.Moved, h.FeatureName)
	}

	return nil
}

func (v *rssVisitor) VisitSplitFeatures(features []workertypes.SummaryHighlight) error {
	for _, h := range features {
		v.data.Split = append(v.data.Split, h.FeatureName)
	}

	return nil
}

func (v *rssVisitor) VisitDeletedFeatures(features []workertypes.SummaryHighlight) error {
	for _, h := range features {
		v.data.Deleted = append(v.data.Deleted, h.FeatureName)
	}

	return nil
}
