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
	workertypes.BaseSummaryVisitor
	data RSSItemData
}

func newEmptyRSSItemData() RSSItemData {
	return RSSItemData{
		SummaryText:         "",
		Added:               nil,
		Removed:             nil,
		Other:               nil,
		QueryErrors:         nil,
		ResolvedQueryErrors: nil,
		Truncated:           false,
	}
}

func newRSSVisitor(triggers []workertypes.JobTrigger) *rssVisitor {
	return &rssVisitor{
		BaseSummaryVisitor: workertypes.NewBaseSummaryVisitor(triggers),
		data:               newEmptyRSSItemData(),
	}
}

func (v *rssVisitor) VisitV1(summary workertypes.EventSummary) error {
	if err := v.BaseSummaryVisitor.VisitV1(summary); err != nil {
		return err
	}
	v.data.SummaryText = v.Categorized.SummaryText
	v.data.Truncated = v.Categorized.Truncated
	for _, qe := range v.Categorized.QueryErrors {
		v.data.QueryErrors = append(v.data.QueryErrors, qe.Code.Message())
	}
	for _, qe := range v.Categorized.ResolvedQueryErrors {
		v.data.ResolvedQueryErrors = append(v.data.ResolvedQueryErrors, qe.Code.Message())
	}
	for _, h := range v.Categorized.Added {
		v.data.Added = append(v.data.Added, h.FeatureName)
	}
	for _, h := range v.Categorized.Removed {
		v.data.Removed = append(v.data.Removed, h.FeatureName)
	}
	for _, h := range v.Categorized.Other() {
		v.data.Other = append(v.data.Other, fmt.Sprintf("%s (%s)", h.FeatureName, h.Type))
	}

	return nil
}

func (v *rssVisitor) HasContent() bool {
	return v.Categorized.HasContent()
}
