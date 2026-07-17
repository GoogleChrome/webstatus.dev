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

	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginLive
	summary.Text = "11 features changed"
	summary.Categories = workertypes.SummaryCategories{
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
	}
	highlights := []workertypes.SummaryHighlight{
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
	}
	for _, h := range highlights {
		summary.AddHighlight(h)
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

func TestSlackSender_Send_QueryError_Golden(t *testing.T) {
	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginFallbackPrevious
	summary.Text = "Query failed"
	summary.SetQueryErrors([]workertypes.SummaryQueryError{
		{Code: workertypes.SummaryQueryErrorCodeSavedSearchNotFound},
	})
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

	goldenFile := filepath.Join("testdata", "slack_payload_query_error.golden.json")

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

func TestSlackPayloadBuilder_TriggerFiltering(t *testing.T) {
	builder := &slackPayloadBuilder{
		frontendBaseURL:           "https://webstatus.dev",
		query:                     "group:css",
		resultsURL:                "https://webstatus.dev/features?q=group:css",
		summary:                   workertypes.NewEmptyEventSummary(),
		queryErrors:               nil,
		resolvedQueryErrors:       nil,
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

	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginLive
	summary.Text = "Test summary"
	highlights := []workertypes.SummaryHighlight{
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
	}
	for _, h := range highlights {
		summary.AddHighlight(h)
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

func TestSlackPayloadBuilder_ResolvedQueryError_Golden(t *testing.T) {
	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginLive
	summary.Text = "Search query recovered and tracking 2 features normally."
	summary.SetResolvedQueryErrors([]workertypes.SummaryQueryError{
		{Code: workertypes.SummaryQueryErrorCodeQueryGrammar},
	})

	builder := newSlackPayloadBuilder("http://localhost:5555", "group:css",
		"http://localhost:5555/features?q=group:css", "sub-123", nil)
	if err := builder.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 failed: %v", err)
	}

	payload := builder.buildPayload("My CSS Search")

	goldenFile := filepath.Join("testdata", "slack_payload_resolved_query_error.golden.json")

	payloadBytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if *updateGolden {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenFile, payloadBytes, 0600); err != nil {
			t.Fatal(err)
		}
	}

	expected, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	if diff := cmp.Diff(string(expected), string(payloadBytes)); diff != "" {
		t.Errorf("Payload mismatch (-want +got):\n%s", diff)
	}
}

func TestSlackPayloadBuilder_CombinedErrorsAndFeatures_Golden(t *testing.T) {
	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginLive
	summary.Text = "Partial errors alongside feature updates"
	summary.SetQueryErrors([]workertypes.SummaryQueryError{
		{Code: workertypes.SummaryQueryErrorCodeSavedSearchNotFound},
	})
	summary.SetResolvedQueryErrors([]workertypes.SummaryQueryError{
		{Code: workertypes.SummaryQueryErrorCodeQueryGrammar},
	})
	summary.AddHighlight(newTestHighlight(workertypes.SummaryHighlightTypeAdded, "feat-added", "Subgrid"))

	builder := newSlackPayloadBuilder("http://localhost:5555", "group:css",
		"http://localhost:5555/features?q=group:css", "sub-123", nil)
	if err := builder.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 failed: %v", err)
	}

	payload := builder.buildPayload("My CSS Search")

	goldenFile := filepath.Join("testdata", "slack_payload_combined_errors_and_features.golden.json")

	payloadBytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if *updateGolden {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenFile, payloadBytes, 0600); err != nil {
			t.Fatal(err)
		}
	}

	expected, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	if diff := cmp.Diff(string(expected), string(payloadBytes)); diff != "" {
		t.Errorf("Payload mismatch (-want +got):\n%s", diff)
	}
}

func TestSlackPayloadBuilder_FeatureCategories(t *testing.T) {
	testCases := []struct {
		name      string
		highlight workertypes.SummaryHighlight
		wantText  string
	}{
		{
			name: "Added feature",
			highlight: workertypes.SummaryHighlight{
				Type:        workertypes.SummaryHighlightTypeAdded,
				FeatureID:   "f-add",
				FeatureName: "Added Feature",
			},
			wantText: "Added Feature",
		},
		{
			name: "Removed feature",
			highlight: workertypes.SummaryHighlight{
				Type:        workertypes.SummaryHighlightTypeRemoved,
				FeatureID:   "f-rem",
				FeatureName: "Removed Feature",
			},
			wantText: "Removed Feature",
		},
		{
			name: "Changed feature",
			highlight: workertypes.SummaryHighlight{
				Type:        workertypes.SummaryHighlightTypeChanged,
				FeatureID:   "f-chg",
				FeatureName: "Changed Feature",
				BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
					To: workertypes.BaselineValue{Status: workertypes.BaselineStatusNewly},
				},
			},
			wantText: "Changed Feature",
		},
		{
			name: "Moved feature",
			highlight: workertypes.SummaryHighlight{
				Type:        workertypes.SummaryHighlightTypeMoved,
				FeatureID:   "f-mvd",
				FeatureName: "Moved Feature",
				Moved: &workertypes.Change[workertypes.FeatureRef]{
					From: workertypes.FeatureRef{Name: "Old Name"},
					To:   workertypes.FeatureRef{Name: "Moved Feature"},
				},
			},
			wantText: "Moved Feature",
		},
		{
			name: "Split feature",
			highlight: workertypes.SummaryHighlight{
				Type:        workertypes.SummaryHighlightTypeSplit,
				FeatureID:   "f-splt",
				FeatureName: "Split Feature",
				Split: &workertypes.SplitChange{
					From: workertypes.FeatureRef{Name: "Split Feature"},
					To:   []workertypes.FeatureRef{{Name: "Child Feature"}},
				},
			},
			wantText: "Split Feature",
		},
		{
			name: "Deleted feature",
			highlight: workertypes.SummaryHighlight{
				Type:        workertypes.SummaryHighlightTypeDeleted,
				FeatureID:   "f-del",
				FeatureName: "Deleted Feature",
			},
			wantText: "Deleted Feature",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := newSlackPayloadBuilder("https://webstatus.dev", "group:css",
				"https://webstatus.dev/features?q=group:css", "sub-123", nil)

			summary := workertypes.NewEmptyEventSummary()
			summary.AddHighlight(tc.highlight)

			if err := builder.VisitV1(summary); err != nil {
				t.Fatalf("VisitV1 unexpected error: %v", err)
			}

			payload := builder.buildPayload("CSS Search")
			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("failed to marshal payload: %v", err)
			}

			if !bytes.Contains(payloadBytes, []byte(tc.wantText)) {
				t.Errorf("buildPayload payload missing expected text %q; got: %s", tc.wantText, string(payloadBytes))
			}
		})
	}
}

func TestSlackPayloadBuilder_QueryErrors_RenderMessage(t *testing.T) {
	testCases := []struct {
		name        string
		errorCode   workertypes.SummaryQueryErrorCode
		wantMessage string
	}{
		{
			name:        "QueryGrammar error",
			errorCode:   workertypes.SummaryQueryErrorCodeQueryGrammar,
			wantMessage: "Invalid query grammar",
		},
		{
			name:        "SavedSearchNotFound error",
			errorCode:   workertypes.SummaryQueryErrorCodeSavedSearchNotFound,
			wantMessage: "Saved search not found",
		},
		{
			name:        "MaxDepthExceeded error",
			errorCode:   workertypes.SummaryQueryErrorCodeMaxDepthExceeded,
			wantMessage: "Saved search max depth exceeded",
		},
		{
			name:        "InvalidQuery error",
			errorCode:   workertypes.SummaryQueryErrorCodeInvalidQuery,
			wantMessage: "Invalid query",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := newSlackPayloadBuilder("https://webstatus.dev", "group:css",
				"https://webstatus.dev/features?q=group:css", "sub-123", nil)

			summary := workertypes.NewEmptyEventSummary()
			summary.SetQueryErrors([]workertypes.SummaryQueryError{{Code: tc.errorCode}})

			if err := builder.VisitV1(summary); err != nil {
				t.Fatalf("VisitV1 unexpected error: %v", err)
			}

			payload := builder.buildPayload("CSS Search")
			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("failed to marshal payload: %v", err)
			}

			if !bytes.Contains(payloadBytes, []byte(tc.wantMessage)) {
				t.Errorf("buildPayload payload missing expected query error message %q; got: %s", tc.wantMessage, string(payloadBytes))
			}
		})
	}
}

func newTestHighlight(hType workertypes.SummaryHighlightType, id, name string) workertypes.SummaryHighlight {
	return workertypes.SummaryHighlight{
		Type:           hType,
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

func TestSlackPayloadBuilder_CombinedErrorsAndFeatures(t *testing.T) {
	builder := newSlackPayloadBuilder("https://webstatus.dev", "group:css",
		"https://webstatus.dev/features?q=group:css", "sub-123", nil)

	summary := workertypes.NewEmptyEventSummary()
	summary.SetQueryErrors([]workertypes.SummaryQueryError{{Code: workertypes.SummaryQueryErrorCodeSavedSearchNotFound}})
	summary.SetResolvedQueryErrors([]workertypes.SummaryQueryError{{Code: workertypes.SummaryQueryErrorCodeQueryGrammar}})
	summary.AddHighlight(newTestHighlight(workertypes.SummaryHighlightTypeAdded, "feat-added", "Added Feature"))

	if err := builder.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error: %v", err)
	}

	payload := builder.buildPayload("Combined Search")
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	if !bytes.Contains(payloadBytes, []byte("Saved search not found")) {
		t.Errorf("missing query error in payload: %s", string(payloadBytes))
	}
	if !bytes.Contains(payloadBytes, []byte("Query Recovered")) {
		t.Errorf("missing resolved query error in payload: %s", string(payloadBytes))
	}
	if !bytes.Contains(payloadBytes, []byte("Added Feature")) {
		t.Errorf("missing added feature in payload: %s", string(payloadBytes))
	}
}

func TestSlackPayloadBuilder_NilPointerGuards(t *testing.T) {
	builder := newSlackPayloadBuilder(
		"sub-1",
		"chan-1",
		"sub-1",
		"http://localhost:8080",
		nil,
	)

	summary := workertypes.NewEmptyEventSummary()
	summary.AddHighlight(workertypes.SummaryHighlight{
		Type:           workertypes.SummaryHighlightTypeMoved,
		FeatureID:      "feat-moved",
		FeatureName:    "Moved Feature",
		Docs:           nil,
		NameChange:     nil,
		BaselineChange: nil,
		BrowserChanges: nil,
		Moved:          nil, // nil guard check
		Split:          nil,
	})
	summary.AddHighlight(workertypes.SummaryHighlight{
		Type:           workertypes.SummaryHighlightTypeSplit,
		FeatureID:      "feat-split",
		FeatureName:    "Split Feature",
		Docs:           nil,
		NameChange:     nil,
		BaselineChange: nil,
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil, // nil guard check
	})
	summary.AddHighlight(workertypes.SummaryHighlight{
		Type:           workertypes.SummaryHighlightTypeChanged,
		FeatureID:      "feat-browser",
		FeatureName:    "Browser Feature",
		Docs:           nil,
		NameChange:     nil,
		Moved:          nil,
		Split:          nil,
		BaselineChange: nil,
		//nolint:exhaustive // Only testing chrome for nil version guard test
		BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
			"chrome": {
				From: workertypes.BrowserValue{
					Status:  workertypes.BrowserStatusAvailable,
					Version: nil, // nil pointer guard check
					Date:    nil,
				},
				To: workertypes.BrowserValue{
					Status:  workertypes.BrowserStatusUnavailable,
					Version: nil,
					Date:    nil,
				},
			},
		},
	})

	if err := builder.VisitV1(summary); err != nil {
		t.Fatalf("VisitV1 unexpected error: %v", err)
	}

	// Must build payload without panicking
	payload := builder.buildPayload("Nil Check Search")
	if len(payload.Blocks) == 0 {
		t.Fatal("buildPayload returned empty blocks")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	if !bytes.Contains(payloadBytes, []byte("Available → *Unavailable*")) {
		t.Errorf("expected payload to contain 'Available → *Unavailable*' without version, got: %s", string(payloadBytes))
	}
}
