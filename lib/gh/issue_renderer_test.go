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
)

func TestRenderIssueTitle(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		featureName string
		trigger     string
		expected    string
	}{
		{
			name:        "Widely Available Trigger",
			featureName: "CSS Subgrid",
			trigger:     "feature.baseline.promote_to_widely",
			expected:    "🚀 Baseline Update: CSS Subgrid is now Widely Available!",
		},
		{
			name:        "Newly Available Trigger",
			featureName: "View Transitions API",
			trigger:     "feature.baseline.promote_to_newly",
			expected:    "✨ Baseline Update: View Transitions API is now Newly Available!",
		},
		{
			name:        "Default Trigger",
			featureName: "Popover API",
			trigger:     "any_other_trigger",
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
		Trigger:            "feature.baseline.promote_to_widely",
		RepositoryFullName: "GoogleChrome/webstatus.dev",
		CommitSHA:          "abcdef1234567890abcdef1234567890abcdef12",
		Occurrences: []IssueOccurrence{
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
		Trigger:            "feature.baseline.promote_to_newly",
		RepositoryFullName: "org/repo",
		CommitSHA:          "sha123",
		Occurrences: []IssueOccurrence{
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

func TestRenderIssueBody_TruncationAndSnippetClamping(t *testing.T) {
	t.Parallel()

	// 30 occurrences: should truncate to 25 and show "... and 5 more occurrences"
	occurrences := make([]IssueOccurrence, 0, 30)
	for i := 1; i <= 30; i++ {
		occurrences = append(occurrences, IssueOccurrence{
			FilePath:       "src/file.ts",
			LineNumber:     int64(i),
			CommentSnippet: strings.Repeat("a", 250), // longer than 200 chars
		})
	}

	params := IssueRenderParams{
		FeatureID:          "popover",
		FeatureName:        "Popover API",
		Trigger:            "feature.baseline.promote_to_newly",
		RepositoryFullName: "org/repo",
		CommitSHA:          "sha",
		Occurrences:        occurrences,
		WebStatusURL:       "https://webstatus.dev/features/popover",
	}

	body := RenderIssueBody(params)

	if !strings.Contains(body, "*... and 5 more occurrences in this repository.*") {
		t.Errorf("expected overflow notice for 30 occurrences, got: %s", body)
	}

	// Ensure individual snippet is clamped to 200 chars + "..."
	expectedSnippet := strings.Repeat("a", 200) + "..."
	if !strings.Contains(body, expectedSnippet) {
		t.Errorf("expected clamped snippet in body, got: %s", body)
	}

	// Test backtick sanitization
	backtickParams := IssueRenderParams{
		FeatureID:          "popover",
		FeatureName:        "Popover API",
		Trigger:            "feature.baseline.promote_to_newly",
		RepositoryFullName: "org/repo",
		CommitSHA:          "sha",
		Occurrences: []IssueOccurrence{
			{
				FilePath:       "src/file.ts",
				LineNumber:     1,
				CommentSnippet: "/* TODO: `dialog` fallback */",
			},
		},
		WebStatusURL: "",
	}
	backtickBody := RenderIssueBody(backtickParams)
	if strings.Contains(backtickBody, "`dialog`") {
		t.Errorf("unescaped backtick found inside code span: %s", backtickBody)
	}
	if !strings.Contains(backtickBody, "'dialog'") {
		t.Errorf("expected backtick replaced with single quote: %s", backtickBody)
	}
}

func TestRenderIssueTitle_SanitizeNewlines(t *testing.T) {
	t.Parallel()

	title := RenderIssueTitle("Feature\nWith\rNewlines", "feature.baseline.promote_to_widely")
	if strings.Contains(title, "\n") || strings.Contains(title, "\r") {
		t.Errorf("newlines were not sanitized in issue title: %q", title)
	}
	if !strings.Contains(title, "Feature With Newlines") {
		t.Errorf("expected space-replaced feature name in title: %q", title)
	}
}
