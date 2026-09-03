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
	"fmt"
	"html"
	"path/filepath"
	"strings"
)

// IssueOccurrence represents a code location occurrence in a repository.
type IssueOccurrence struct {
	FilePath       string
	LineNumber     int64
	CommentSnippet string
}

// IssueRenderParams contains all parameters needed to render a GitHub issue notification.
type IssueRenderParams struct {
	FeatureID          string
	FeatureName        string
	Trigger            string
	RepositoryFullName string
	CommitSHA          string
	Occurrences        []IssueOccurrence
	WebStatusURL       string
}

const (
	// MaxRenderedOccurrences caps the number of occurrences displayed in the issue body
	// to prevent exceeding GitHub's 65,536 character payload limit.
	MaxRenderedOccurrences = 25
	// MaxSnippetLength caps the length of individual comment snippets.
	MaxSnippetLength = 200
)

func sanitizePath(p string) string {
	cleaned := filepath.Clean(p)
	cleaned = strings.TrimPrefix(cleaned, "/")
	cleaned = strings.ReplaceAll(cleaned, "../", "")

	return cleaned
}

func sanitizeSnippet(s string) string {
	if len(s) > MaxSnippetLength {
		s = s[:MaxSnippetLength] + "..."
	}

	return strings.ReplaceAll(s, "`", "'")
}

// RenderIssueTitle generates a descriptive issue title based on feature and trigger.
func RenderIssueTitle(featureName string, trigger string) string {
	cleanName := strings.ReplaceAll(strings.ReplaceAll(featureName, "\r", " "), "\n", " ")
	switch trigger {
	case "feature.baseline.promote_to_widely":
		return fmt.Sprintf("🚀 Baseline Update: %s is now Widely Available!", cleanName)
	case "feature.baseline.promote_to_newly":
		return fmt.Sprintf("✨ Baseline Update: %s is now Newly Available!", cleanName)
	default:
		return fmt.Sprintf("🚀 Web Feature Ready: %s", cleanName)
	}
}

// RenderIssueBody generates markdown body for a GitHub notification issue.
func RenderIssueBody(params IssueRenderParams) string {
	var sb strings.Builder

	escapedFeatureID := html.EscapeString(params.FeatureID)
	escapedFeatureName := html.EscapeString(params.FeatureName)

	title := RenderIssueTitle(params.FeatureName, params.Trigger)
	fmt.Fprintf(&sb, "## %s\n\n", title)

	statusText := "Baseline Widely Available"
	if params.Trigger == "feature.baseline.promote_to_newly" {
		statusText = "Baseline Newly Available"
	}

	fmt.Fprintf(&sb, "**`%s`** has achieved **%s** across major engines.\n\n",
		escapedFeatureID, statusText)

	if params.WebStatusURL != "" {
		fmt.Fprintf(&sb, "📊 [View Feature Status on webstatus.dev](%s)\n\n", params.WebStatusURL)
	}

	occurrences := params.Occurrences
	truncated := 0
	if len(occurrences) > MaxRenderedOccurrences {
		truncated = len(occurrences) - MaxRenderedOccurrences
		occurrences = occurrences[:MaxRenderedOccurrences]
	}

	sb.WriteString("### 📍 Affected File Locations:\n")
	for _, occ := range occurrences {
		safePath := sanitizePath(occ.FilePath)
		permalink := fmt.Sprintf("https://github.com/%s/blob/%s/%s#L%d",
			params.RepositoryFullName, params.CommitSHA, safePath, occ.LineNumber)
		snippet := sanitizeSnippet(occ.CommentSnippet)
		fmt.Fprintf(&sb, "- [`%s:L%d`](%s): `%s`\n",
			safePath, occ.LineNumber, permalink, snippet)
	}
	if truncated > 0 {
		fmt.Fprintf(&sb, "\n*... and %d more occurrences in this repository.*\n", truncated)
	}

	sb.WriteString("\n### 🤖 AI Refactoring Prompt\n```text\n")
	fmt.Fprintf(&sb, "Refactor code to adopt native %s (%s).\n", escapedFeatureName, escapedFeatureID)
	sb.WriteString("Review affected file locations, clean up legacy fallbacks/shims, " +
		"and remove TODO(baseline/...) directives.\n")
	sb.WriteString("Follow Modern Web Guidance: https://github.com/GoogleChrome/modern-web-guidance\n")
	sb.WriteString("```\n")

	return sb.String()
}
