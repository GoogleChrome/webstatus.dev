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
		name  string
		query string
		text  string
	}{
		{
			name:  "Search Query",
			query: "baseline_status:newly",
			text:  "Manual Test: Search update for 'baseline_status:newly'",
		},
		{
			name:  "Feature Query",
			query: "id:\"anchor-positioning\"",
			text:  "Manual Test: Feature update for 'Anchor Positioning'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			summary := workertypes.EventSummary{
				SchemaVersion: "v1",
				Text:          tc.text,
				Categories:    workertypes.SummaryCategories{Added: 1},
				Highlights: []workertypes.SummaryHighlight{
					{
						Type:        workertypes.SummaryHighlightTypeAdded,
						FeatureID:   "anchor-positioning",
						FeatureName: "Anchor Positioning",
					},
				},
			}
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

			mgr := &slackManager{
				frontendBaseURL: "http://localhost:5555",
				httpClient:      &http.Client{},
				stateManager:    &noopStateManager{}, // Don't try to write to Spanner
				job:             job,
			}

			err := mgr.Send(context.Background())
			if err != nil {
				t.Errorf("Failed to send Slack message for %s: %v", tc.name, err)
			} else {
				fmt.Printf("Slack message sent successfully for %s!\n", tc.name)
			}
		})
	}
}

type noopStateManager struct{}

func (n *noopStateManager) RecordSuccess(ctx context.Context, channelID string, sentAt time.Time, webhookEventID string) error {
	return nil
}
func (n *noopStateManager) RecordFailure(ctx context.Context, channelID string, err error, failedAt time.Time, isPermanent bool, webhookEventID string) error {
	return nil
}
