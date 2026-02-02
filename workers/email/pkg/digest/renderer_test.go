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
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/generic"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/google/go-cmp/cmp"
)

// nolint:gochecknoglobals  // WONTFIX - used for testing only
var updateGolden = flag.Bool("update", false, "update golden files")

func TestRenderDigest_Golden(t *testing.T) {
	newlyDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	widelyDate := time.Date(2025, 12, 27, 0, 0, 0, 0, time.UTC)

	// Setup complex test data to exercise all templates
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
			QueryChanged:    0,
			UpdatedImpl:     0,
			UpdatedRename:   0,
			UpdatedBaseline: 3,
		},
		Truncated: false,
		Highlights: []workertypes.SummaryHighlight{
			{
				// Case 1: Baseline Widely (with multiple docs)
				Type:        workertypes.SummaryHighlightTypeChanged,
				FeatureName: "Container queries",
				FeatureID:   "container-queries",
				Docs: &workertypes.Docs{
					MDNDocs: []workertypes.DocLink{
						{URL: "https://developer.mozilla.org/docs/Web/CSS/CSS_Container_Queries", Title: nil, Slug: nil},
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
							Version: generic.ValuePtr("120"), Date: nil},
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
							Version: generic.ValuePtr("17.2"),
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
				// Case 8: Moved
				Type:        workertypes.SummaryHighlightTypeMoved,
				FeatureName: "New Cool Name",
				FeatureID:   "new-cool-name",
				Moved: &workertypes.Change[workertypes.FeatureRef]{
					From: workertypes.FeatureRef{ID: "old-name", Name: "Old Name"},
					To:   workertypes.FeatureRef{ID: "new-cool-name", Name: "New Cool Name"},
				},
				Docs:           nil,
				NameChange:     nil,
				BaselineChange: nil,
				BrowserChanges: nil,
				Split:          nil,
			},
			{
				// Case 9: Split
				Type:        workertypes.SummaryHighlightTypeSplit,
				FeatureName: "Feature To Split",
				FeatureID:   "feature-to-split",
				Split: &workertypes.SplitChange{
					From: workertypes.FeatureRef{ID: "feature-to-split", Name: "Feature To Split"},
					To: []workertypes.FeatureRef{
						{ID: "sub-feature-1", Name: "Sub Feature 1"},
						{ID: "sub-feature-2", Name: "Sub Feature 2"},
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
							Version: generic.ValuePtr("123"), Date: &newlyDate},
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
		},
	}
	summaryBytes, _ := json.Marshal(summary)

	job := workertypes.IncomingEmailDeliveryJob{
		EmailDeliveryJob: workertypes.EmailDeliveryJob{
			SummaryRaw:     summaryBytes,
			RecipientEmail: "rick@example.com",
			SubscriptionID: "sub-123",
			ChannelID:      "chan-1",
			Metadata: workertypes.DeliveryMetadata{
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

func TestFilterHighlights(t *testing.T) {
	newlyDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	widelyDate := time.Date(2025, 12, 27, 0, 0, 0, 0, time.UTC)
	availableDate := time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC)

	// Reusable highlight definitions
	hNewly := workertypes.SummaryHighlight{
		Type:        workertypes.SummaryHighlightTypeChanged,
		FeatureID:   "h1",
		FeatureName: "Newly Feature",
		BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
			From: workertypes.BaselineValue{Status: workertypes.BaselineStatusLimited, LowDate: nil, HighDate: nil},
			To: workertypes.BaselineValue{Status: workertypes.BaselineStatusNewly, LowDate: &newlyDate,
				HighDate: nil},
		},
		Docs:           nil,
		NameChange:     nil,
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	}
	hWidely := workertypes.SummaryHighlight{
		Type:        workertypes.SummaryHighlightTypeChanged,
		FeatureID:   "h2",
		FeatureName: "Widely Feature",
		BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
			From: workertypes.BaselineValue{Status: workertypes.BaselineStatusNewly, LowDate: &newlyDate,
				HighDate: nil},
			To: workertypes.BaselineValue{Status: workertypes.BaselineStatusWidely, LowDate: &newlyDate,
				HighDate: &widelyDate},
		},
		Docs:           nil,
		NameChange:     nil,
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	}
	hRegression := workertypes.SummaryHighlight{
		Type:        workertypes.SummaryHighlightTypeChanged,
		FeatureID:   "h3",
		FeatureName: "Regression Feature",
		BaselineChange: &workertypes.Change[workertypes.BaselineValue]{
			From: workertypes.BaselineValue{Status: workertypes.BaselineStatusWidely, LowDate: &newlyDate,
				HighDate: &widelyDate},
			To: workertypes.BaselineValue{Status: workertypes.BaselineStatusLimited, LowDate: nil, HighDate: nil},
		},
		Docs:           nil,
		NameChange:     nil,
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	}
	hBrowser := workertypes.SummaryHighlight{
		Type:        workertypes.SummaryHighlightTypeChanged,
		FeatureID:   "h4",
		FeatureName: "Browser Feature",
		BrowserChanges: map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]{
			workertypes.BrowserChrome: {
				From: workertypes.BrowserValue{Status: workertypes.BrowserStatusUnavailable, Version: nil, Date: nil},
				To: workertypes.BrowserValue{Status: workertypes.BrowserStatusAvailable, Version: nil,
					Date: &availableDate},
			},
			workertypes.BrowserEdge:           nil,
			workertypes.BrowserFirefox:        nil,
			workertypes.BrowserSafari:         nil,
			workertypes.BrowserChromeAndroid:  nil,
			workertypes.BrowserFirefoxAndroid: nil,
			workertypes.BrowserSafariIos:      nil,
		},
		BaselineChange: nil,
		Docs:           nil,
		NameChange:     nil,
		Moved:          nil,
		Split:          nil,
	}
	hGenericAdded := workertypes.SummaryHighlight{
		Type:           workertypes.SummaryHighlightTypeAdded,
		FeatureID:      "h5",
		FeatureName:    "Generic Added",
		Docs:           nil,
		NameChange:     nil,
		BaselineChange: nil,
		BrowserChanges: nil,
		Moved:          nil,
		Split:          nil,
	}

	allHighlights := []workertypes.SummaryHighlight{hNewly, hWidely, hRegression, hBrowser, hGenericAdded}

	tests := []struct {
		name     string
		triggers []workertypes.JobTrigger
		// IDs of expected highlights
		wantIDs []string
	}{
		{
			name:     "No Triggers (Default) - Should Return All",
			triggers: nil,
			wantIDs:  []string{"h1", "h2", "h3", "h4", "h5"},
		},
		{
			name:     "Empty Triggers List - Should Return All (Same as nil)",
			triggers: []workertypes.JobTrigger{},
			wantIDs:  []string{"h1", "h2", "h3", "h4", "h5"},
		},
		{
			name:     "Newly Trigger",
			triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToNewly},
			wantIDs:  []string{"h1"},
		},
		{
			name:     "Widely Trigger",
			triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToWidely},
			wantIDs:  []string{"h2"},
		},
		{
			name:     "Regression Trigger",
			triggers: []workertypes.JobTrigger{workertypes.FeatureRegressedToLimited},
			wantIDs:  []string{"h3"},
		},
		{
			name:     "Browser Implementation Trigger",
			triggers: []workertypes.JobTrigger{workertypes.BrowserImplementationAnyComplete},
			wantIDs:  []string{"h4"},
		},
		{
			name: "Multiple Triggers (Newly + Widely)",
			triggers: []workertypes.JobTrigger{
				workertypes.FeaturePromotedToNewly,
				workertypes.FeaturePromotedToWidely,
			},
			wantIDs: []string{"h1", "h2"},
		},
		{
			name:     "No Matches",
			triggers: []workertypes.JobTrigger{workertypes.FeaturePromotedToWidely}, // Only h2 matches
			// Pass in only h1 (Newly)
			wantIDs: nil, // If input is only h1, result is empty.
			// But for this test, we run against 'allHighlights' by default unless customized.
			// Let's customize the logic below to handle "wantIDs subset of allHighlights"
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := allHighlights
			// Special case for "No Matches" test to be clear: pass in only items that definitely don't match
			if tc.name == "No Matches" {
				input = []workertypes.SummaryHighlight{hNewly}
			}

			got := filterHighlights(input, tc.triggers)

			if len(got) != len(tc.wantIDs) {
				t.Errorf("Count mismatch: got %d, want %d", len(got), len(tc.wantIDs))
			}

			for i, h := range got {
				if i < len(tc.wantIDs) && h.FeatureID != tc.wantIDs[i] {
					t.Errorf("Index %d mismatch: got ID %s, want %s", i, h.FeatureID, tc.wantIDs[i])
				}
			}
		})
	}
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
