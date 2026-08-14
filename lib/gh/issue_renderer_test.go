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

package gh

import (
	"strings"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
)

func TestRenderIssueTitle(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		featureName string
		trigger     gcpspanner.SubscriptionTrigger
		expected    string
	}{
		{
			name:        "Widely Available Trigger",
			featureName: "CSS Subgrid",
			trigger:     gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToWidely,
			expected:    "🚀 Baseline Update: CSS Subgrid is now Widely Available!",
		},
		{
			name:        "Newly Available Trigger",
			featureName: "View Transitions API",
			trigger:     gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToNewly,
			expected:    "✨ Baseline Update: View Transitions API is now Newly Available!",
		},
		{
			name:        "Default Trigger",
			featureName: "Popover API",
			trigger:     gcpspanner.SubscriptionTriggerBrowserImplementationAnyComplete,
			expected:    "🚀 Web Feature Ready: Popover API",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := RenderIssueTitle(tc.featureName, tc.trigger)
			if actual != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestRenderIssueBody(t *testing.T) {
	t.Parallel()

	params := IssueRenderParams{
		FeatureID:          "css-subgrid",
		FeatureName:        "CSS Subgrid",
		Trigger:            gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToWidely,
		RepositoryFullName: "GoogleChrome/webstatus.dev",
		CommitSHA:          "abcdef1234567890abcdef1234567890abcdef12",
		Occurrences: []gcpspanner.SubscriptionOccurrence{
			{
				FilePath:       "src/components/grid.css",
				LineNumber:     42,
				CommentSnippet: "/* TODO(baseline/subgrid): Remove flexbox fallback */",
			},
			{
				FilePath:       "src/polyfills/subgrid.js",
				LineNumber:     10,
				CommentSnippet: "// Legacy polyfill",
			},
		},
		WebStatusURL: "https://webstatus.dev/features/css-subgrid",
	}

	body := RenderIssueBody(params)

	if !strings.Contains(body, "## 🚀 Baseline Update: CSS Subgrid is now Widely Available!") {
		t.Errorf("missing header in body: %s", body)
	}
	if !strings.Contains(body, "**`css-subgrid`** has achieved **Baseline Widely Available**") {
		t.Errorf("missing status line in body: %s", body)
	}
	if !strings.Contains(body, "[View Feature Status on webstatus.dev](https://webstatus.dev/features/css-subgrid)") {
		t.Errorf("missing webstatus URL link in body: %s", body)
	}
	expectedPermalink := "https://github.com/GoogleChrome/webstatus.dev/blob/" +
		"abcdef1234567890abcdef1234567890abcdef12/src/components/grid.css#L42"
	if !strings.Contains(body, expectedPermalink) {
		t.Errorf("missing permalink %s in body: %s", expectedPermalink, body)
	}
	if !strings.Contains(body, "### 🤖 AI Refactoring Prompt") {
		t.Errorf("missing AI refactoring prompt section in body: %s", body)
	}
}

func TestRenderIssueBodySanitization(t *testing.T) {
	t.Parallel()

	params := IssueRenderParams{
		FeatureID:          "injection<script>",
		FeatureName:        "Feature & Name",
		Trigger:            gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToNewly,
		RepositoryFullName: "org/repo",
		CommitSHA:          "sha123",
		Occurrences: []gcpspanner.SubscriptionOccurrence{
			{
				FilePath:       "../../etc/passwd",
				LineNumber:     1,
				CommentSnippet: "/* <img src=x onerror=alert(1)> */",
			},
		},
		WebStatusURL: "",
	}

	body := RenderIssueBody(params)

	if strings.Contains(body, "<script>") {
		t.Errorf("raw script tag found in body: %s", body)
	}
	if !strings.Contains(body, "injection&lt;script&gt;") {
		t.Errorf("expected HTML-escaped feature ID, got: %s", body)
	}
	if strings.Contains(body, "../../") {
		t.Errorf("path traversal sequence not sanitized: %s", body)
	}
	if !strings.Contains(body, "etc/passwd") {
		t.Errorf("expected sanitized relative path, got: %s", body)
	}
}
