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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

// nolint:gochecknoglobals  // WONTFIX - used for testing only
var updateGolden = flag.Bool("update", false, "update golden files")

func TestSlackSender_Send(t *testing.T) {
	tests := []slackTestCase{
		{
			name: "successful send with correct query-based payload",
			job: newTestIncomingWebhookDeliveryJob(
				"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
				workertypes.WebhookTypeSlack,
				"group:css",
				[]byte(`{"text":"New feature landed"}`),
			),
			mockResponse: newTestResponse(http.StatusOK, "ok"),
			mockErr:      nil,
			expectedPayload: &SlackPayload{
				Text: "WebStatus.dev Notification: New feature landed\n" +
					"Query: group:css\n" +
					"View Results: https://webstatus.dev/?q=group%3Acss",
				Blocks: nil,
			},
			expectedErr: nil,
		},
		{
			name: "successful send with direct feature link",
			job: newTestIncomingWebhookDeliveryJob(
				"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
				workertypes.WebhookTypeSlack,
				"id:\"anchor-positioning\"",
				[]byte(`{"text":"Test Body"}`),
			),
			mockResponse: newTestResponse(http.StatusOK, "ok"),
			mockErr:      nil,
			expectedPayload: &SlackPayload{
				Text: "WebStatus.dev Notification: Test Body\n" +
					"Query: id:\"anchor-positioning\"\n" +
					"View Results: https://webstatus.dev/features/anchor-positioning",
				Blocks: nil,
			},
			expectedErr: nil,
		},
		{
			name: "network error",
			job: newTestIncomingWebhookDeliveryJob(
				"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
				workertypes.WebhookTypeSlack,
				"",
				[]byte(`{"text":"retry"}`),
			),
			mockResponse:    nil,
			mockErr:         errors.New("network failure"),
			expectedPayload: nil,
			expectedErr:     ErrTransientWebhook,
		},
		{
			name: "permanent error (404)",
			job: newTestIncomingWebhookDeliveryJob(
				"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
				workertypes.WebhookTypeSlack,
				"",
				[]byte(`{"text":"fail"}`),
			),
			mockResponse: newTestResponse(http.StatusNotFound, "not found"),
			mockErr:      nil,
			expectedPayload: &SlackPayload{
				Text: "WebStatus.dev Notification: fail\n" +
					"Query: \n" +
					"View Results: https://webstatus.dev/?q=",
				Blocks: nil,
			},
			expectedErr: ErrPermanentWebhook,
		},
		{
			name: "transient error (500)",
			job: newTestIncomingWebhookDeliveryJob(
				"https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
				workertypes.WebhookTypeSlack,
				"",
				[]byte(`{"text":"retry"}`),
			),
			mockResponse: newTestResponse(http.StatusInternalServerError, "internal error"),
			mockErr:      nil,
			expectedPayload: &SlackPayload{
				Text: "WebStatus.dev Notification: retry\n" +
					"Query: \n" +
					"View Results: https://webstatus.dev/?q=",
				Blocks: nil,
			},
			expectedErr: ErrTransientWebhook,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedBody := tt.runTest(t)
			tt.verifyPayload(t, capturedBody)
		})
	}
}

func (tc *slackTestCase) runTest(t *testing.T) []byte {
	var capturedBody []byte
	mockHTTP := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if tc.mockErr != nil {
				return nil, tc.mockErr
			}
			var err error
			capturedBody, err = io.ReadAll(req.Body)

			return tc.mockResponse, err
		},
	}

	sender, err := newSlackSender("https://webstatus.dev", mockHTTP, tc.job)
	if err != nil {
		if tc.expectedErr != nil && errors.Is(err, tc.expectedErr) {
			return nil
		}
		t.Fatalf("unexpected error creating sender: %v", err)
	}

	err = sender.Send(context.Background())
	if tc.expectedErr != nil {
		if !errors.Is(err, tc.expectedErr) {
			t.Errorf("Send() error = %v, expectedErr %v", err, tc.expectedErr)
		}
	} else if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	return capturedBody
}

func (tc *slackTestCase) verifyPayload(t *testing.T, capturedBody []byte) {
	var actualPayload *SlackPayload
	if len(capturedBody) > 0 {
		actualPayload = new(SlackPayload)
		if err := json.Unmarshal(capturedBody, actualPayload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
	}

	if diff := cmp.Diff(tc.expectedPayload, actualPayload); diff != "" {
		t.Errorf("payload mismatch (-want +got):\n%s", diff)
	}
}

type slackTestCase struct {
	name            string
	job             workertypes.IncomingWebhookDeliveryJob
	mockResponse    *http.Response
	mockErr         error
	expectedPayload *SlackPayload
	expectedErr     error
}

func TestSlackSender_Send_Golden(t *testing.T) {
	newlyDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	widelyDate := time.Date(2025, 12, 27, 0, 0, 0, 0, time.UTC)

	summary := workertypes.EventSummary{
		SchemaVersion: "v1",
		Text:          "11 features changed",
		Categories: workertypes.SummaryCategories{
			Updated:         5,
			Added:           2,
			Removed:         1,
			Moved:           1,
			Split:           1,
			Deleted:         1,
			UpdatedBaseline: 3,
			QueryChanged:    0,
			UpdatedImpl:     0,
			UpdatedRename:   0,
		},
		Truncated:   false,
		QueryErrors: nil,
		Highlights: []workertypes.SummaryHighlight{
			{
				Type:        workertypes.SummaryHighlightTypeChanged,
				FeatureName: "Container queries",
				FeatureID:   "container-queries",
				Docs: &workertypes.Docs{
					MDNDocs: []workertypes.DocLink{
						{
							URL:   "https://developer.mozilla.org/docs/Web/CSS/CSS_Container_Queries",
							Title: nil,
							Slug:  nil,
						},
					},
				},
				NameChange:     nil,
				BrowserChanges: nil,
				Moved:          nil,
				Split:          nil,
				BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
					From: workertypes.BaselineValue{
						Status:   workertypes.BaselineStatusNewly,
						LowDate:  &newlyDate,
						HighDate: nil},
					To: workertypes.BaselineValue{
						Status:   workertypes.BaselineStatusWidely,
						LowDate:  &newlyDate,
						HighDate: &widelyDate},
				},
			},
			{
				Type:           workertypes.SummaryHighlightTypeChanged,
				FeatureName:    "Newly Available Feature",
				FeatureID:      "newly-feature",
				Docs:           nil,
				NameChange:     nil,
				BrowserChanges: nil,
				Moved:          nil,
				Split:          nil,
				BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
					From: workertypes.BaselineValue{
						Status:   workertypes.BaselineStatusLimited,
						LowDate:  nil,
						HighDate: nil,
					},
					To: workertypes.BaselineValue{
						Status:   workertypes.BaselineStatusNewly,
						LowDate:  &newlyDate,
						HighDate: nil,
					},
				},
			},
			{
				Type:        workertypes.SummaryHighlightTypeChanged,
				FeatureName: "Regressed Feature",
				FeatureID:   "regressed-feature",
				Docs:        nil,
				NameChange:  nil,
				Moved:       nil,
				Split:       nil,
				BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
					From: workertypes.BaselineValue{
						Status:   workertypes.BaselineStatusWidely,
						LowDate:  &newlyDate,
						HighDate: &widelyDate},
					To: workertypes.BaselineValue{
						Status:   workertypes.BaselineStatusLimited,
						LowDate:  nil,
						HighDate: nil},
				},
				BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
					workertypes.BrowserChrome: {
						From: workertypes.BrowserValue{
							Status:  workertypes.BrowserStatusAvailable,
							Version: new("120"),
							Date:    nil,
						},
						To: workertypes.BrowserValue{
							Status:  workertypes.BrowserStatusUnavailable,
							Version: nil,
							Date:    nil,
						},
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
				Type:           workertypes.SummaryHighlightTypeChanged,
				FeatureName:    "content-visibility",
				FeatureID:      "content-visibility",
				Docs:           nil,
				BaselineChange: nil,
				NameChange:     nil,
				Moved:          nil,
				Split:          nil,
				BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
					workertypes.BrowserSafariIos: {
						From: workertypes.BrowserValue{
							Status:  workertypes.BrowserStatusUnavailable,
							Version: nil,
							Date:    nil,
						},
						To: workertypes.BrowserValue{
							Status:  workertypes.BrowserStatusAvailable,
							Version: new("17.2"),
							Date:    nil,
						},
					},
					workertypes.BrowserChrome:         nil,
					workertypes.BrowserChromeAndroid:  nil,
					workertypes.BrowserEdge:           nil,
					workertypes.BrowserFirefox:        nil,
					workertypes.BrowserFirefoxAndroid: nil,
					workertypes.BrowserSafari:         nil,
				},
			},
			{
				Type:           workertypes.SummaryHighlightTypeChanged,
				FeatureName:    "another-feature",
				FeatureID:      "another-feature",
				Docs:           nil,
				BaselineChange: nil,
				NameChange:     nil,
				Moved:          nil,
				Split:          nil,
				BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
					workertypes.BrowserChrome: {
						From: workertypes.BrowserValue{
							Status:  workertypes.BrowserStatusUnavailable,
							Version: nil,
							Date:    nil,
						},
						To: workertypes.BrowserValue{
							Status:  workertypes.BrowserStatusAvailable,
							Version: nil,
							Date:    &newlyDate,
						},
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
				Type:           workertypes.SummaryHighlightTypeAdded,
				FeatureName:    "New Feature",
				FeatureID:      "new-feature",
				Docs:           nil,
				BaselineChange: nil,
				BrowserChanges: nil,
				NameChange:     nil,
				Moved:          nil,
				Split:          nil,
			},
			{
				Type:           workertypes.SummaryHighlightTypeAdded,
				FeatureName:    "Another New Feature",
				FeatureID:      "another-new-feature",
				Docs:           nil,
				BaselineChange: nil,
				BrowserChanges: nil,
				NameChange:     nil,
				Moved:          nil,
				Split:          nil,
			},
			{
				Type:           workertypes.SummaryHighlightTypeRemoved,
				FeatureName:    "Removed Feature",
				FeatureID:      "removed-feature",
				Docs:           nil,
				BaselineChange: nil,
				BrowserChanges: nil,
				NameChange:     nil,
				Moved:          nil,
				Split:          nil,
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
					From: workertypes.FeatureRef{
						ID:         "old-name",
						Name:       "Old Name",
						QueryMatch: workertypes.QueryMatchNoMatch},
					To: workertypes.FeatureRef{
						ID:         "new-cool-name",
						Name:       "New Cool Name",
						QueryMatch: workertypes.QueryMatchNoMatch},
				},
			},
			{
				Type:           workertypes.SummaryHighlightTypeSplit,
				FeatureName:    "Feature To Split",
				FeatureID:      "feature-to-split",
				Docs:           nil,
				BaselineChange: nil,
				BrowserChanges: nil,
				NameChange:     nil,
				Moved:          nil,
				Split: &workertypes.SplitChange{
					From: workertypes.FeatureRef{
						ID: "feature-to-split", Name: "Feature To Split", QueryMatch: workertypes.QueryMatchNoMatch},
					To: []workertypes.FeatureRef{
						{ID: "sub-feature-1", Name: "Sub Feature 1", QueryMatch: workertypes.QueryMatchMatch},
						{ID: "sub-feature-2", Name: "Sub Feature 2", QueryMatch: workertypes.QueryMatchNoMatch},
					},
				},
			},
			{
				Type:           workertypes.SummaryHighlightTypeDeleted,
				FeatureName:    "Deleted Feature",
				FeatureID:      "deleted-feature",
				Docs:           nil,
				BaselineChange: nil,
				BrowserChanges: nil,
				NameChange:     nil,
				Moved:          nil,
				Split:          nil,
			},
			{
				Type:        workertypes.SummaryHighlightTypeRemoved,
				FeatureName: "Removed With Details",
				FeatureID:   "removed-details",
				Docs:        nil,
				NameChange:  nil,
				Moved:       nil,
				Split:       nil,
				BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
					From: workertypes.BaselineValue{
						Status:   workertypes.BaselineStatusNewly,
						LowDate:  &newlyDate,
						HighDate: nil,
					},
					To: workertypes.BaselineValue{
						Status:   workertypes.BaselineStatusLimited,
						LowDate:  nil,
						HighDate: nil,
					},
				},
				BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
					workertypes.BrowserChrome: {
						From: workertypes.BrowserValue{
							Status:  workertypes.BrowserStatusAvailable,
							Version: new("110"),
							Date:    nil,
						},
						To: workertypes.BrowserValue{
							Status:  workertypes.BrowserStatusUnavailable,
							Version: nil,
							Date:    nil,
						},
					},
					workertypes.BrowserChromeAndroid:  nil,
					workertypes.BrowserEdge:           nil,
					workertypes.BrowserFirefox:        nil,
					workertypes.BrowserFirefoxAndroid: nil,
					workertypes.BrowserSafari:         nil,
					workertypes.BrowserSafariIos:      nil,
				},
			},
		},
	}
	summaryBytes, _ := json.Marshal(summary)

	job := workertypes.IncomingWebhookDeliveryJob{
		WebhookEventID: "",
		WebhookDeliveryJob: workertypes.WebhookDeliveryJob{
			SubscriptionID: "",
			WebhookType:    workertypes.WebhookTypeSlack,
			ChannelID:      "",
			Triggers:       nil,
			SummaryRaw:     summaryBytes,
			WebhookURL:     "https://hooks.slack.com/services/T00/B00/XXX",
			Metadata: workertypes.DeliveryMetadata{
				EventID:     "",
				SearchID:    "",
				SearchName:  "My CSS Search",
				Query:       "group:css",
				Frequency:   workertypes.FrequencyWeekly,
				GeneratedAt: time.Time{},
			},
		},
	}

	var capturedBody []byte
	mockHTTP := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			var err error
			capturedBody, err = io.ReadAll(req.Body)

			return &http.Response{
				StatusCode:       http.StatusOK,
				Body:             io.NopCloser(bytes.NewBufferString("ok")),
				Status:           "",
				Proto:            "",
				ProtoMajor:       0,
				ProtoMinor:       0,
				Header:           nil,
				ContentLength:    0,
				TransferEncoding: nil,
				Close:            false,
				Uncompressed:     false,
				Trailer:          nil,
				Request:          nil,
				TLS:              nil,
			}, err
		},
	}

	sender, err := newSlackSender("https://webstatus.dev", mockHTTP, job)
	if err != nil {
		t.Fatalf("failed to create sender: %v", err)
	}

	err = sender.Send(context.Background())
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	goldenFile := filepath.Join("testdata", "slack_payload.golden.json")

	if *updateGolden {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, capturedBody, "", "  "); err != nil {
			t.Fatalf("failed to indent JSON: %v", err)
		}
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenFile, prettyJSON.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}

	expected, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	var expectedPayload, actualPayload map[string]any
	if err := json.Unmarshal(expected, &expectedPayload); err != nil {
		t.Fatalf("failed to decode expected: %v", err)
	}
	if err := json.Unmarshal(capturedBody, &actualPayload); err != nil {
		t.Fatalf("failed to decode actual: %v", err)
	}

	if diff := cmp.Diff(expectedPayload, actualPayload); diff != "" {
		t.Errorf("Payload mismatch (-want +got):\n%s", diff)
	}
}

//go:fix inline
func TestSlackPayloadBuilder_VisitV1_Filter(t *testing.T) {
	builder := &slackPayloadBuilder{
		frontendBaseURL: "https://webstatus.dev",
		query:           "group:css",
		resultsURL:      "https://webstatus.dev/features?q=group:css",
		summary: workertypes.EventSummary{
			SchemaVersion: "v1",
			Text:          "",
			Truncated:     false,
			QueryErrors:   nil,
			Highlights:    nil,
			Categories: workertypes.SummaryCategories{
				QueryChanged:    0,
				Added:           0,
				Removed:         0,
				Deleted:         0,
				Moved:           0,
				Split:           0,
				Updated:         0,
				UpdatedImpl:     0,
				UpdatedRename:   0,
				UpdatedBaseline: 0,
			},
		},
		queryErrors:               nil,
		baselineNewlyChanges:      nil,
		baselineWidelyChanges:     nil,
		baselineRegressionChanges: nil,
		allBrowserChanges:         nil,
		addedFeatures:             nil,
		removedFeatures:           nil,
		deletedFeatures:           nil,
		splitFeatures:             nil,
		movedFeatures:             nil,
		triggers: []workertypes.JobTrigger{
			workertypes.BrowserImplementationAnyComplete,
		},
		subscriptionID: "",
	}

	summary := workertypes.EventSummary{
		SchemaVersion: "v1",
		Categories: workertypes.SummaryCategories{
			QueryChanged:    0,
			Added:           0,
			Removed:         0,
			Deleted:         0,
			Moved:           0,
			Split:           0,
			Updated:         0,
			UpdatedImpl:     0,
			UpdatedRename:   0,
			UpdatedBaseline: 0,
		},
		Text:        "Test summary",
		Truncated:   false,
		QueryErrors: nil,
		Highlights: []workertypes.SummaryHighlight{
			{
				Type:        workertypes.SummaryHighlightTypeChanged,
				FeatureName: "Chrome Feature",
				FeatureID:   "chrome-feat",
				Docs:        nil,
				BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
					workertypes.BrowserChrome: {
						From: workertypes.BrowserValue{
							Status:  workertypes.BrowserStatusUnavailable,
							Version: nil,
							Date:    nil,
						},
						To: workertypes.BrowserValue{
							Status:  workertypes.BrowserStatusAvailable,
							Version: nil,
							Date:    nil,
						},
					},
					workertypes.BrowserChromeAndroid:  nil,
					workertypes.BrowserEdge:           nil,
					workertypes.BrowserFirefox:        nil,
					workertypes.BrowserFirefoxAndroid: nil,
					workertypes.BrowserSafari:         nil,
					workertypes.BrowserSafariIos:      nil,
				},
				NameChange: nil, Moved: nil, Split: nil, BaselineChange: nil,
			},
			{
				Type:           workertypes.SummaryHighlightTypeAdded,
				FeatureName:    "Added Feature",
				FeatureID:      "added-feat",
				Docs:           nil,
				BrowserChanges: nil,
				NameChange:     nil,
				Moved:          nil,
				Split:          nil,
				BaselineChange: nil,
			},
		},
	}

	err := builder.VisitV1(summary)
	if err != nil {
		t.Fatalf("VisitV1 failed: %v", err)
	}

	// Chrome feature should be present (1 browser change found)
	if len(builder.allBrowserChanges) != 1 {
		t.Errorf("expected 1 browser change, got %d", len(builder.allBrowserChanges))
	} else if builder.allBrowserChanges[0].FeatureID != "chrome-feat" {
		t.Errorf("expected chrome-feat, got %s", builder.allBrowserChanges[0].FeatureID)
	}
}
