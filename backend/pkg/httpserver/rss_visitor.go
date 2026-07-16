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

type rssVisitor struct {
	triggers []workertypes.JobTrigger
	data     RSSItemData
}

func newEmptyRSSItemData() RSSItemData {
	return RSSItemData{
		SummaryText: "",
		Added:       nil,
		Removed:     nil,
		Other:       nil,
		QueryErrors: nil,
		Truncated:   false,
	}
}

func newRSSVisitor(triggers []workertypes.JobTrigger) *rssVisitor {
	return &rssVisitor{
		triggers: triggers,
		data:     newEmptyRSSItemData(),
	}
}

func (v *rssVisitor) VisitV1(summary workertypes.EventSummary) error {
	v.data = newEmptyRSSItemData()
	v.data.SummaryText = summary.Text
	v.data.Truncated = summary.Truncated
	for _, qe := range summary.QueryErrors {
		v.data.QueryErrors = append(v.data.QueryErrors, qe.Code.Message())
	}

	// 1. Filter highlights against user triggers using shared workertypes helper
	highlights := workertypes.FilterHighlights(summary.Highlights, v.triggers)

	// 2. Populate current RSS categories
	for _, h := range highlights {
		switch h.Type {
		case workertypes.SummaryHighlightTypeAdded:
			v.data.Added = append(v.data.Added, h.FeatureName)
		case workertypes.SummaryHighlightTypeRemoved:
			v.data.Removed = append(v.data.Removed, h.FeatureName)
		case workertypes.SummaryHighlightTypeChanged,
			workertypes.SummaryHighlightTypeMoved,
			workertypes.SummaryHighlightTypeSplit,
			workertypes.SummaryHighlightTypeDeleted:
			v.data.Other = append(v.data.Other, fmt.Sprintf("%s (%s)", h.FeatureName, h.Type))
		}
	}

	return nil
}

func (v *rssVisitor) HasContent() bool {
	return len(v.data.QueryErrors) > 0 ||
		len(v.data.Added) > 0 ||
		len(v.data.Removed) > 0 ||
		len(v.data.Other) > 0
}
