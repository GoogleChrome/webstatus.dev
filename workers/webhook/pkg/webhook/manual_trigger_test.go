//go:build manual

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

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

// TestManualSlackTrigger is a helper "test" that sends a real Slack message.
// To run this test, fill in a valid slackURL below and run:
// go test -v -tags=manual -run TestManualSlackTrigger ./workers/webhook/pkg/webhook
func TestManualSlackTrigger(t *testing.T) {
	slackURL := "" // <--- FILL THIS IN
	if slackURL == "" {
		t.Skip("slackURL not set, skipping manual trigger test")
	}

	testCases := []struct {
		name    string
		query   string
		text    string
		summary *workertypes.EventSummary
	}{
		{
			name:  "Complete Golden Payload (All Sections)",
			query: "group:css",
			text:  "Manual Test: Golden Payload",
			summary: &workertypes.EventSummary{
				SchemaVersion: "v1",
				Text:          "Golden payload test summary",
				Truncated:     false,
				Highlights: []workertypes.SummaryHighlight{
					{
						Type:           workertypes.SummaryHighlightTypeChanged,
						FeatureName:    "Anchor Positioning",
						FeatureID:      "anchor-positioning",
						Docs:           nil,
						NameChange:     nil,
						Moved:          nil,
						Split:          nil,
						BaselineChange: nil,
						BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
							workertypes.BrowserChrome: {
								From: workertypes.BrowserValue{Status: workertypes.BrowserStatusUnavailable, Version: nil, Date: nil},
								To:   workertypes.BrowserValue{Status: workertypes.BrowserStatusAvailable, Version: new("110"), Date: nil},
							},
							workertypes.BrowserChromeAndroid:  nil,
							workertypes.BrowserEdge:           nil,
							workertypes.BrowserFirefox:        nil,
							workertypes.BrowserFirefoxAndroid: nil,
							workertypes.BrowserSafari:         nil,
							workertypes.BrowserSafariIos:      nil,
						},
					},
					{
						Type:           workertypes.SummaryHighlightTypeMoved,
						FeatureName:    "New Cool Name",
						FeatureID:      "new-cool-name",
						Docs:           nil,
						BaselineChange: nil,
						BrowserChanges: nil,
						NameChange:     nil,
						Split:          nil,
						Moved: &workertypes.Change[workertypes.FeatureRef]{
							From: workertypes.FeatureRef{ID: "old-name", Name: "Old Name", QueryMatch: workertypes.QueryMatchNoMatch},
							To:   workertypes.FeatureRef{ID: "new-cool-name", Name: "New Cool Name", QueryMatch: workertypes.QueryMatchNoMatch},
						},
					},
					{
						Type:        workertypes.SummaryHighlightTypeAdded,
						FeatureID:   "newly-added",
						FeatureName: "Newly Added Feature",
					},
					{
						Type:        workertypes.SummaryHighlightTypeRemoved,
						FeatureID:   "removed-feature",
						FeatureName: "Removed Feature",
					},
					{
						Type:        workertypes.SummaryHighlightTypeDeleted,
						FeatureID:   "deleted-feature",
						FeatureName: "Deleted Feature",
					},
					{
						Type:        workertypes.SummaryHighlightTypeSplit,
						FeatureID:   "split-feature",
						FeatureName: "Split Feature Host",
						Split: &workertypes.SplitChange{
							From: workertypes.FeatureRef{ID: "split-feature", Name: "Split Feature Host", QueryMatch: workertypes.QueryMatchNoMatch},
							To: []workertypes.FeatureRef{
								{ID: "split-sub-1", Name: "Sub Feature 1", QueryMatch: workertypes.QueryMatchMatch},
								{ID: "split-sub-2", Name: "Sub Feature 2", QueryMatch: workertypes.QueryMatchNoMatch},
							},
						},
					},
					{
						Type:        workertypes.SummaryHighlightTypeChanged,
						FeatureID:   "baseline-shift",
						FeatureName: "Feature Moving to Widely Available",
						BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
							From: workertypes.BaselineValue{Status: workertypes.BaselineStatusNewly, LowDate: nil, HighDate: nil},
							To:   workertypes.BaselineValue{Status: workertypes.BaselineStatusWidely, LowDate: nil, HighDate: nil},
						},
					},
				},
				Categories: workertypes.SummaryCategories{
					QueryChanged:    2,
					Added:           1,
					Removed:         1,
					Deleted:         1,
					Moved:           1,
					Split:           1,
					Updated:         0,
					UpdatedImpl:     1,
					UpdatedRename:   0,
					UpdatedBaseline: 1,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			summary := *tc.summary
			summaryRaw, _ := json.Marshal(summary)

			job := workertypes.IncomingWebhookDeliveryJob{
				WebhookDeliveryJob: workertypes.WebhookDeliveryJob{
					WebhookURL: slackURL,
					SummaryRaw: summaryRaw,
					Metadata: workertypes.DeliveryMetadata{
						EventID:     "manual-trigger-event",
						SearchID:    "manual-search-id",
						SearchName:  tc.name,
						Query:       tc.query,
						Frequency:   workertypes.FrequencyImmediate,
						GeneratedAt: time.Now(),
					},
					Triggers:    []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly},
					ChannelID:   "manual-channel-id",
					WebhookType: workertypes.WebhookTypeSlack,
				},
				WebhookEventID: "manual-webhook-event-id",
			}

			mgr, err := newSlackSender("http://localhost:5555", &http.Client{}, job)
			if err != nil {
				t.Fatalf("Failed to create sender: %v", err)
			}

			err = mgr.Send(context.Background())
			if err != nil {
				t.Errorf("Failed to send Slack message for %s: %v", tc.name, err)
			} else {
				fmt.Printf("Slack message sent successfully for %s!\n", tc.name)
			}
		})
	}
}
