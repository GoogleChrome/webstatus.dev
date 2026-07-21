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

package digest

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

// nolint:gochecknoglobals  // WONTFIX - used for testing only
var updateGolden = flag.Bool("update", false, "update golden files")

func TestRenderDigest_Golden(t *testing.T) {
	newlyDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	widelyDate := time.Date(2025, 12, 27, 0, 0, 0, 0, time.UTC)

	// Setup complex test data to exercise all templates
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
		QueryChanged:    0,
		UpdatedImpl:     0,
		UpdatedRename:   0,
		UpdatedBaseline: 3,
	}
	highlights := []workertypes.SummaryHighlight{
		{
			// Case 1: Baseline Widely (with multiple docs)
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
					{URL: "https://developer.mozilla.org/docs/Web/CSS/container-queries", Title: nil, Slug: nil},
				},
			},
			BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
				From: workertypes.BaselineValue{Status: workertypes.BaselineStatusNewly, LowDate: &newlyDate,
					HighDate: nil},
				To: workertypes.BaselineValue{Status: workertypes.BaselineStatusWidely, LowDate: &newlyDate,
					HighDate: &widelyDate},
			},
			NameChange:     nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
		{
			// Case 2: Baseline Newly
			Type:        workertypes.SummaryHighlightTypeChanged,
			FeatureName: "Newly Available Feature",
			FeatureID:   "newly-feature",
			BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
				From: workertypes.BaselineValue{Status: workertypes.BaselineStatusLimited, LowDate: nil,
					HighDate: nil},
				To: workertypes.BaselineValue{Status: workertypes.BaselineStatusNewly, LowDate: &newlyDate,
					HighDate: nil},
			},
			Docs:           nil,
			NameChange:     nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
		{
			// Case 3: Baseline Regression
			Type:        workertypes.SummaryHighlightTypeChanged,
			FeatureName: "Regressed Feature",
			FeatureID:   "regressed-feature",
			BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
				From: workertypes.BaselineValue{Status: workertypes.BaselineStatusWidely,
					LowDate: &newlyDate, HighDate: &widelyDate},
				To: workertypes.BaselineValue{Status: workertypes.BaselineStatusLimited,
					LowDate: nil, HighDate: nil},
			},
			BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
				workertypes.BrowserChrome: {
					From: workertypes.BrowserValue{Status: workertypes.BrowserStatusAvailable,
						Version: new("120"), Date: nil},
					To: workertypes.BrowserValue{Status: workertypes.BrowserStatusUnavailable, Version: nil,
						Date: nil},
				},
				workertypes.BrowserFirefox:        nil,
				workertypes.BrowserChromeAndroid:  nil,
				workertypes.BrowserEdge:           nil,
				workertypes.BrowserFirefoxAndroid: nil,
				workertypes.BrowserSafari:         nil,
				workertypes.BrowserSafariIos:      nil,
			},
			Docs:       nil,
			NameChange: nil,
			Moved:      nil,
			Split:      nil,
		},
		{
			// Case 4: Browser Implementation with version
			Type:        workertypes.SummaryHighlightTypeChanged,
			FeatureName: "content-visibility",
			FeatureID:   "content-visibility",
			BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
				workertypes.BrowserSafariIos: {
					From: workertypes.BrowserValue{Status: workertypes.BrowserStatusUnavailable, Version: nil,
						Date: nil},
					To: workertypes.BrowserValue{Status: workertypes.BrowserStatusAvailable,
						Version: new("17.2"),
						// Purposefully set to nil to test that it doesn't crash.
						Date: nil},
				},
				workertypes.BrowserChrome:         nil,
				workertypes.BrowserChromeAndroid:  nil,
				workertypes.BrowserEdge:           nil,
				workertypes.BrowserFirefoxAndroid: nil,
				workertypes.BrowserSafari:         nil,
				workertypes.BrowserFirefox:        nil,
			},
			Docs:           nil,
			NameChange:     nil,
			BaselineChange: nil,
			Moved:          nil,
			Split:          nil,
		},
		{
			// Case 5: Browser Implementation with date
			Type:        workertypes.SummaryHighlightTypeChanged,
			FeatureName: "another-feature",
			FeatureID:   "another-feature",
			BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
				workertypes.BrowserChrome: {
					From: workertypes.BrowserValue{Status: workertypes.BrowserStatusUnavailable,
						Date: nil, Version: nil},
					To: workertypes.BrowserValue{Status: workertypes.BrowserStatusAvailable, Date: &newlyDate,
						// Purposefully set to nil so that we can see that it doesn't crash.
						Version: nil},
				},
				workertypes.BrowserFirefox:        nil,
				workertypes.BrowserChromeAndroid:  nil,
				workertypes.BrowserEdge:           nil,
				workertypes.BrowserFirefoxAndroid: nil,
				workertypes.BrowserSafari:         nil,
				workertypes.BrowserSafariIos:      nil,
			},
			Docs:           nil,
			NameChange:     nil,
			BaselineChange: nil,
			Moved:          nil,
			Split:          nil,
		},
		{
			// Case 6: Added
			Type:           workertypes.SummaryHighlightTypeAdded,
			FeatureName:    "New Feature",
			FeatureID:      "new-feature",
			Docs:           nil,
			NameChange:     nil,
			BaselineChange: nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
		{
			// Case 6b: Another Added
			Type:           workertypes.SummaryHighlightTypeAdded,
			FeatureName:    "Another New Feature",
			FeatureID:      "another-new-feature",
			Docs:           nil,
			NameChange:     nil,
			BaselineChange: nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
		{
			// Case 7: Removed
			Type:           workertypes.SummaryHighlightTypeRemoved,
			FeatureName:    "Removed Feature",
			FeatureID:      "removed-feature",
			Docs:           nil,
			NameChange:     nil,
			BaselineChange: nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
		{
			// Case 8: Moved - No Match
			Type:        workertypes.SummaryHighlightTypeMoved,
			FeatureName: "New Cool Name",
			FeatureID:   "new-cool-name",
			Moved: &workertypes.Change[workertypes.FeatureRef]{
				From: workertypes.FeatureRef{ID: "old-name", Name: "Old Name", QueryMatch: ""},
				To: workertypes.FeatureRef{
					ID:         "new-cool-name",
					Name:       "New Cool Name",
					QueryMatch: workertypes.QueryMatchNoMatch,
				},
			},
			Docs:           nil,
			NameChange:     nil,
			BaselineChange: nil,
			BrowserChanges: nil,
			Split:          nil,
		},
		{
			// Case 9: Split - Partial Match
			Type:        workertypes.SummaryHighlightTypeSplit,
			FeatureName: "Feature To Split",
			FeatureID:   "feature-to-split",
			Split: &workertypes.SplitChange{
				From: workertypes.FeatureRef{ID: "feature-to-split", Name: "Feature To Split", QueryMatch: ""},
				To: []workertypes.FeatureRef{
					{ID: "sub-feature-1", Name: "Sub Feature 1", QueryMatch: workertypes.QueryMatchMatch},
					{ID: "sub-feature-2", Name: "Sub Feature 2", QueryMatch: workertypes.QueryMatchNoMatch},
				},
			},
			Docs:           nil,
			NameChange:     nil,
			BaselineChange: nil,
			BrowserChanges: nil,
			Moved:          nil,
		},
		{
			// Case 10: Browser Implementation with version and date
			Type:        workertypes.SummaryHighlightTypeChanged,
			FeatureName: "new-browser-feature",
			FeatureID:   "new-browser-feature",
			BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
				workertypes.BrowserFirefox: {
					From: workertypes.BrowserValue{Status: workertypes.BrowserStatusUnavailable,
						Version: nil, Date: nil},
					To: workertypes.BrowserValue{Status: workertypes.BrowserStatusAvailable,
						Version: new("123"), Date: &newlyDate},
				},
				workertypes.BrowserChrome:         nil,
				workertypes.BrowserChromeAndroid:  nil,
				workertypes.BrowserEdge:           nil,
				workertypes.BrowserFirefoxAndroid: nil,
				workertypes.BrowserSafari:         nil,
				workertypes.BrowserSafariIos:      nil,
			},
			NameChange:     nil,
			BaselineChange: nil,
			Moved:          nil,
			Split:          nil,
			Docs:           nil,
		},
		{
			// Case 11: Deleted
			Type:           workertypes.SummaryHighlightTypeDeleted,
			FeatureName:    "Deleted Feature",
			FeatureID:      "deleted-feature",
			Docs:           nil,
			NameChange:     nil,
			BaselineChange: nil,
			BrowserChanges: nil,
			Moved:          nil,
			Split:          nil,
		},
		{
			// Case 12: Removed with details (Baseline + Browser changes)
			Type:        workertypes.SummaryHighlightTypeRemoved,
			FeatureName: "Removed With Details",
			FeatureID:   "removed-details",
			Docs:        nil,
			BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
				From: workertypes.BaselineValue{
					Status:  workertypes.BaselineStatusNewly,
					LowDate: &newlyDate, HighDate: nil,
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
				workertypes.BrowserEdge:           nil,
				workertypes.BrowserFirefox:        nil,
				workertypes.BrowserSafari:         nil,
				workertypes.BrowserChromeAndroid:  nil,
				workertypes.BrowserFirefoxAndroid: nil,
				workertypes.BrowserSafariIos:      nil,
			},
			NameChange: nil,
			Moved:      nil,
			Split:      nil,
		},
		{
			// Case 13: Browser implementation grouping (Desktop & Mobile)
			Type:        workertypes.SummaryHighlightTypeChanged,
			FeatureName: "grouped-browser-feature",
			FeatureID:   "grouped-browser-feature",
			BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
				workertypes.BrowserChrome: {
					From: workertypes.BrowserValue{Status: workertypes.BrowserStatusUnknown,
						Version: nil, Date: nil},
					To: workertypes.BrowserValue{Status: workertypes.BrowserStatusAvailable,
						Version: new("148"), Date: &newlyDate},
				},
				workertypes.BrowserChromeAndroid: {
					From: workertypes.BrowserValue{Status: workertypes.BrowserStatusUnknown,
						Version: nil, Date: nil},
					To: workertypes.BrowserValue{Status: workertypes.BrowserStatusAvailable,
						Version: new("148"), Date: &newlyDate},
				},
				workertypes.BrowserEdge:           nil,
				workertypes.BrowserFirefox:        nil,
				workertypes.BrowserFirefoxAndroid: nil,
				workertypes.BrowserSafari:         nil,
				workertypes.BrowserSafariIos:      nil,
			},
			NameChange:     nil,
			BaselineChange: nil,
			Moved:          nil,
			Split:          nil,
			Docs:           nil,
		},
		{
			// Case 14: Browser implementation NOT grouping (Different Versions)
			Type:        workertypes.SummaryHighlightTypeChanged,
			FeatureName: "ungrouped-browser-feature",
			FeatureID:   "ungrouped-browser-feature",
			BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
				workertypes.BrowserChrome:         nil,
				workertypes.BrowserChromeAndroid:  nil,
				workertypes.BrowserEdge:           nil,
				workertypes.BrowserFirefox:        nil,
				workertypes.BrowserFirefoxAndroid: nil,
				workertypes.BrowserSafari: {
					From: workertypes.BrowserValue{Status: workertypes.BrowserStatusUnknown,
						Version: nil, Date: nil},
					To: workertypes.BrowserValue{Status: workertypes.BrowserStatusAvailable,
						Version: new("17.0"), Date: &newlyDate},
				},
				workertypes.BrowserSafariIos: {
					From: workertypes.BrowserValue{Status: workertypes.BrowserStatusUnknown,
						Version: nil, Date: nil},
					To: workertypes.BrowserValue{Status: workertypes.BrowserStatusAvailable,
						Version: new("17.2"), Date: &newlyDate},
				},
			},
			NameChange:     nil,
			BaselineChange: nil,
			Moved:          nil,
			Split:          nil,
			Docs:           nil,
		},
	}
	for _, h := range highlights {
		summary.AddHighlight(h)
	}
	summaryBytes, _ := json.Marshal(summary)

	job := workertypes.IncomingEmailDeliveryJob{
		EmailDeliveryJob: workertypes.EmailDeliveryJob{
			SummaryRaw:     summaryBytes,
			RecipientEmail: "rick@example.com",
			SubscriptionID: "sub-123",
			ChannelID:      "chan-1",
			Metadata: workertypes.DeliveryMetadata{
				SearchName:  "My CSS Search",
				Query:       "group:css",
				Frequency:   workertypes.FrequencyWeekly,
				EventID:     "evt-123",
				SearchID:    "s-1",
				GeneratedAt: time.Now(),
			},
			Triggers: nil,
		},
		EmailEventID: "email-event-id",
	}

	// Initialize Renderer
	renderer, err := NewHTMLRenderer("http://localhost:5555")
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	// Execute
	_, body, err := renderer.RenderDigest(job)
	if err != nil {
		t.Fatalf("RenderDigest failed: %v", err)
	}

	goldenFile := filepath.Join("testdata", "digest.golden.html")

	if *updateGolden {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenFile, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}

	expected, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	if diff := cmp.Diff(string(expected), body); diff != "" {
		t.Errorf("HTML mismatch (-want +got):\n%s", diff)
	}
}

func renderAndVerifyDigestGolden(
	t *testing.T,
	summary workertypes.EventSummary,
	freq workertypes.JobFrequency,
	eventID string,
	goldenFilename string,
) {
	t.Helper()

	summaryRaw, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}

	job := workertypes.IncomingEmailDeliveryJob{
		EmailDeliveryJob: workertypes.EmailDeliveryJob{
			SummaryRaw:     summaryRaw,
			RecipientEmail: "user@example.com",
			SubscriptionID: "sub-123",
			ChannelID:      "chan-1",
			Metadata: workertypes.DeliveryMetadata{
				SearchName:  "My CSS Search",
				Query:       "group:css",
				Frequency:   freq,
				EventID:     eventID,
				SearchID:    "s-1",
				GeneratedAt: time.Now(),
			},
			Triggers: nil,
		},
		EmailEventID: "email-event-id",
	}

	renderer, err := NewHTMLRenderer("http://localhost:5555")
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	_, body, err := renderer.RenderDigest(job)
	if err != nil {
		t.Fatalf("RenderDigest failed: %v", err)
	}

	goldenFile := filepath.Join("testdata", goldenFilename)

	if *updateGolden {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenFile, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}

	expected, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	if diff := cmp.Diff(string(expected), body); diff != "" {
		t.Errorf("HTML mismatch (-want +got):\n%s", diff)
	}
}

func TestRenderDigest_QueryError_Golden(t *testing.T) {
	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginFallbackPrevious
	summary.Text = "Query failed"
	summary.SetQueryErrors([]workertypes.SummaryQueryError{
		{Code: workertypes.SummaryQueryErrorCodeSavedSearchNotFound},
	})

	renderAndVerifyDigestGolden(t, summary, workertypes.FrequencyWeekly, "evt-123", "digest_query_error.golden.html")
}

func TestRenderDigest_InvalidJSON(t *testing.T) {
	var metadata workertypes.DeliveryMetadata

	job := workertypes.IncomingEmailDeliveryJob{
		EmailDeliveryJob: workertypes.EmailDeliveryJob{
			SummaryRaw:     []byte("invalid-json"),
			SubscriptionID: "",
			RecipientEmail: "",
			Metadata:       metadata,
			ChannelID:      "",
			Triggers:       nil,
		},
		EmailEventID: "email-event-id",
	}

	renderer, _ := NewHTMLRenderer("https://test.dev")
	_, _, err := renderer.RenderDigest(job)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestRenderDigest_ResolvedQueryError_Golden(t *testing.T) {
	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginLive
	summary.Text = "Search query recovered and tracking 2 features normally."
	summary.SetResolvedQueryErrors([]workertypes.SummaryQueryError{
		{Code: workertypes.SummaryQueryErrorCodeQueryGrammar},
	})

	renderAndVerifyDigestGolden(
		t,
		summary,
		workertypes.FrequencyImmediate,
		"evt-recovery",
		"digest_resolved_query_error.golden.html",
	)
}

func newTestAddedHighlight(id, name string) workertypes.SummaryHighlight {
	return workertypes.SummaryHighlight{
		Type:           workertypes.SummaryHighlightTypeAdded,
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

func newTestRemovedHighlight(id, name string) workertypes.SummaryHighlight {
	return workertypes.SummaryHighlight{
		Type:           workertypes.SummaryHighlightTypeRemoved,
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

func newTestChangedHighlight(id, name string, status workertypes.BaselineStatus) workertypes.SummaryHighlight {
	return workertypes.SummaryHighlight{
		Type:        workertypes.SummaryHighlightTypeChanged,
		FeatureID:   id,
		FeatureName: name,
		Docs:        nil,
		NameChange:  nil,
		BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
			From: workertypes.BaselineValue{
				Status:   workertypes.BaselineStatusNewly,
				LowDate:  nil,
				HighDate: nil,
			},
			To: workertypes.BaselineValue{
				Status:   status,
				LowDate:  nil,
				HighDate: nil,
			},
		},
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	}
}

func newTestMovedHighlight(id, name, oldName string) workertypes.SummaryHighlight {
	return workertypes.SummaryHighlight{
		Type:           workertypes.SummaryHighlightTypeMoved,
		FeatureID:      id,
		FeatureName:    name,
		Docs:           nil,
		NameChange:     nil,
		BaselineChange: nil,
		BrowserChanges: nil,
		Moved: &workertypes.Change[workertypes.FeatureRef]{
			From: workertypes.FeatureRef{ID: "", Name: oldName, QueryMatch: workertypes.QueryMatchNoMatch},
			To:   workertypes.FeatureRef{ID: id, Name: name, QueryMatch: workertypes.QueryMatchNoMatch},
		},
		Split: nil,
	}
}

func newTestSplitHighlight(id, name, childName string) workertypes.SummaryHighlight {
	return workertypes.SummaryHighlight{
		Type:           workertypes.SummaryHighlightTypeSplit,
		FeatureID:      id,
		FeatureName:    name,
		Docs:           nil,
		NameChange:     nil,
		BaselineChange: nil,
		BrowserChanges: nil,
		Moved:          nil,
		Split: &workertypes.SplitChange{
			From: workertypes.FeatureRef{ID: id, Name: name, QueryMatch: workertypes.QueryMatchNoMatch},
			To:   []workertypes.FeatureRef{{ID: "c-1", Name: childName, QueryMatch: workertypes.QueryMatchNoMatch}},
		},
	}
}

func newTestDeletedHighlight(id, name string) workertypes.SummaryHighlight {
	return workertypes.SummaryHighlight{
		Type:           workertypes.SummaryHighlightTypeDeleted,
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

// TestRenderDigest_CombinedErrorsAndFeatures verifies the golden HTML layout for the edge case
// where an event summary contains both query error banners (active/resolved) AND feature change
// sections simultaneously in the same email notification digest.
func TestRenderDigest_CombinedErrorsAndFeatures(t *testing.T) {
	summary := workertypes.NewEmptyEventSummary()
	summary.SnapshotOrigin = workertypes.OriginLive
	summary.Text = "Partial errors alongside feature updates"
	summary.SetQueryErrors([]workertypes.SummaryQueryError{
		{Code: workertypes.SummaryQueryErrorCodeSavedSearchNotFound},
	})
	summary.SetResolvedQueryErrors([]workertypes.SummaryQueryError{
		{Code: workertypes.SummaryQueryErrorCodeQueryGrammar},
	})
	summary.AddHighlight(newTestAddedHighlight("f-added", "Subgrid"))

	renderAndVerifyDigestGolden(
		t,
		summary,
		workertypes.FrequencyWeekly,
		"evt-combined",
		"digest_combined_errors_and_features.golden.html",
	)
}

func createTestIncomingJob(
	t *testing.T,
	summary workertypes.EventSummary,
	triggers []workertypes.JobTrigger,
	eventID string,
) workertypes.IncomingEmailDeliveryJob {
	raw, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("failed to marshal summary: %v", err)
	}

	return workertypes.IncomingEmailDeliveryJob{
		EmailDeliveryJob: workertypes.EmailDeliveryJob{
			SubscriptionID: "sub-1",
			RecipientEmail: "user@example.com",
			ChannelID:      "chan-1",
			Triggers:       triggers,
			SummaryRaw:     raw,
			Metadata: workertypes.DeliveryMetadata{
				EventID:     eventID,
				SearchID:    "search-1",
				Query:       "feature:a",
				GeneratedAt: time.Time{},
				Frequency:   workertypes.FrequencyWeekly,
				SearchName:  "Search A",
			},
		},
		EmailEventID: eventID,
	}
}

func TestEmailVisitor_FeatureCategories(t *testing.T) {
	renderer, err := NewHTMLRenderer("https://test.dev")
	if err != nil {
		t.Fatalf("NewHTMLRenderer unexpected error: %v", err)
	}

	testCases := []struct {
		name      string
		highlight workertypes.SummaryHighlight
		wantText  string
	}{
		{
			name:      "Added feature",
			highlight: newTestAddedHighlight("f-add", "Added Feature"),
			wantText:  "Added Feature",
		},
		{
			name:      "Removed feature",
			highlight: newTestRemovedHighlight("f-rem", "Removed Feature"),
			wantText:  "Removed Feature",
		},
		{
			name:      "Changed feature",
			highlight: newTestChangedHighlight("f-chg", "Changed Feature", workertypes.BaselineStatusNewly),
			wantText:  "Changed Feature",
		},
		{
			name:      "Moved feature",
			highlight: newTestMovedHighlight("f-mvd", "Moved Feature", "Old Name"),
			wantText:  "Moved Feature",
		},
		{
			name:      "Split feature",
			highlight: newTestSplitHighlight("f-splt", "Split Feature", "Child Feature"),
			wantText:  "Split Feature",
		},
		{
			name:      "Deleted feature",
			highlight: newTestDeletedHighlight("f-del", "Deleted Feature"),
			wantText:  "Deleted Feature",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			summary := workertypes.NewEmptyEventSummary()
			summary.SnapshotOrigin = workertypes.OriginLive
			summary.Text = "Digest Summary"
			summary.AddHighlight(tc.highlight)

			job := createTestIncomingJob(t, summary, nil, "evt-1")

			_, body, err := renderer.RenderDigest(job)
			if err != nil {
				t.Fatalf("RenderDigest unexpected error: %v", err)
			}

			if !bytes.Contains([]byte(body), []byte(tc.wantText)) {
				t.Errorf("rendered body does not contain expected text %q", tc.wantText)
			}
		})
	}
}

func TestEmailVisitor_QueryErrors_RenderMessage(t *testing.T) {
	renderer, err := NewHTMLRenderer("https://test.dev")
	if err != nil {
		t.Fatalf("NewHTMLRenderer unexpected error: %v", err)
	}

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
			summary := workertypes.NewEmptyEventSummary()
			summary.SetQueryErrors([]workertypes.SummaryQueryError{{Code: tc.errorCode}})

			job := createTestIncomingJob(t, summary, nil, "evt-err")

			_, body, err := renderer.RenderDigest(job)
			if err != nil {
				t.Fatalf("RenderDigest unexpected error: %v", err)
			}

			if !bytes.Contains([]byte(body), []byte(tc.wantMessage)) {
				t.Errorf("rendered body missing translated query error message %q", tc.wantMessage)
			}
		})
	}
}

func TestEmailVisitor_QueryErrors(t *testing.T) {
	renderer, err := NewHTMLRenderer("https://test.dev")
	if err != nil {
		t.Fatalf("NewHTMLRenderer unexpected error: %v", err)
	}

	summary := workertypes.NewEmptyEventSummary()
	summary.SetQueryErrors([]workertypes.SummaryQueryError{
		{Code: workertypes.SummaryQueryErrorCodeSavedSearchNotFound},
	})

	job := createTestIncomingJob(t, summary, nil, "evt-err")

	_, body, err := renderer.RenderDigest(job)
	if err != nil {
		t.Fatalf("RenderDigest unexpected error: %v", err)
	}

	if !bytes.Contains([]byte(body), []byte("Saved search not found")) {
		t.Error("rendered body missing translated query error message")
	}
}

func TestEmailVisitor_ResolvedQueryErrors(t *testing.T) {
	renderer, err := NewHTMLRenderer("https://test.dev")
	if err != nil {
		t.Fatalf("NewHTMLRenderer unexpected error: %v", err)
	}

	summary := workertypes.NewEmptyEventSummary()
	summary.SetResolvedQueryErrors([]workertypes.SummaryQueryError{
		{Code: workertypes.SummaryQueryErrorCodeQueryGrammar},
	})

	job := createTestIncomingJob(t, summary, nil, "evt-rec")

	_, body, err := renderer.RenderDigest(job)
	if err != nil {
		t.Fatalf("RenderDigest unexpected error: %v", err)
	}

	if !bytes.Contains([]byte(body), []byte("Invalid query grammar")) {
		t.Error("rendered body missing resolved query recovery message")
	}
}

func TestEmailVisitor_TriggerFiltering(t *testing.T) {
	renderer, err := NewHTMLRenderer("https://test.dev")
	if err != nil {
		t.Fatalf("NewHTMLRenderer unexpected error: %v", err)
	}

	summary := workertypes.NewEmptyEventSummary()
	summary.AddHighlight(newTestChangedHighlight("f-widely", "Widely Available Feature", workertypes.BaselineStatusWidely))

	// Job triggers only ask for FeaturePromotedToNewly
	job := createTestIncomingJob(t, summary, []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly}, "evt-filtered")

	_, body, err := renderer.RenderDigest(job)
	if err != nil {
		t.Fatalf("RenderDigest unexpected error: %v", err)
	}

	if bytes.Contains([]byte(body), []byte("Widely Available Feature")) {
		t.Error("rendered body should not contain feature filtered out by subscriber triggers")
	}
}

func TestEmailVisitor_NilPointerGuards(t *testing.T) {
	renderer, err := NewHTMLRenderer("https://test.dev")
	if err != nil {
		t.Fatalf("NewHTMLRenderer unexpected error: %v", err)
	}

	summary := workertypes.NewEmptyEventSummary()
	summary.AddHighlight(newTestMovedHighlight("f-mvd-nil", "Nil Moved Feature", ""))
	summary.AddHighlight(newTestSplitHighlight("f-splt-nil", "Nil Split Feature", ""))
	summary.AddHighlight(newTestAddedHighlight("f-valid", "Valid Added Feature"))

	job := createTestIncomingJob(t, summary, nil, "evt-nil")

	_, body, err := renderer.RenderDigest(job)
	if err != nil {
		t.Fatalf("RenderDigest unexpected error on nil pointer highlights: %v", err)
	}

	if !bytes.Contains([]byte(body), []byte("Valid Added Feature")) {
		t.Error("rendered body missing Valid Added Feature")
	}
}

func TestEmailVisitor_HTMLEscaping(t *testing.T) {
	renderer, err := NewHTMLRenderer("https://test.dev")
	if err != nil {
		t.Fatalf("NewHTMLRenderer unexpected error: %v", err)
	}

	summary := workertypes.NewEmptyEventSummary()
	summary.AddHighlight(newTestAddedHighlight("f-xss", "<script>alert('xss')</script>"))

	job := createTestIncomingJob(t, summary, nil, "evt-xss")

	_, body, err := renderer.RenderDigest(job)
	if err != nil {
		t.Fatalf("RenderDigest unexpected error: %v", err)
	}

	if bytes.Contains([]byte(body), []byte("<script>")) {
		t.Error("rendered body contains unescaped <script> tag")
	}
	if !bytes.Contains([]byte(body), []byte("&lt;script&gt;")) {
		t.Error("rendered body missing properly HTML-escaped &lt;script&gt;")
	}
}

func TestEmailVisitor_TransportPayloadVerification(t *testing.T) {
	renderer, err := NewHTMLRenderer("https://test.dev")
	if err != nil {
		t.Fatalf("NewHTMLRenderer unexpected error: %v", err)
	}

	summary := workertypes.NewEmptyEventSummary()
	summary.AddHighlight(newTestAddedHighlight("f-transport-1", "Transport Feature One"))
	summary.AddHighlight(newTestAddedHighlight("f-transport-2", "Transport Feature Two"))

	job := createTestIncomingJob(t, summary, nil, "evt-transport")

	subject, body, err := renderer.RenderDigest(job)
	if err != nil {
		t.Fatalf("RenderDigest unexpected error: %v", err)
	}

	expectedSubject := "Weekly digest: Search A"
	if subject != expectedSubject {
		t.Errorf("subject mismatch: got %q, want %q", subject, expectedSubject)
	}

	hasOne := bytes.Contains([]byte(body), []byte("Transport Feature One"))
	hasTwo := bytes.Contains([]byte(body), []byte("Transport Feature Two"))
	if !hasOne || !hasTwo {
		t.Error("rendered HTML body missing transport feature names")
	}
}
