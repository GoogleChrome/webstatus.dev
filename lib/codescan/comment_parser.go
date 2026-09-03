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

package codescan

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// SubscriptionTrigger defines the baseline promotion trigger in the domain.
type SubscriptionTrigger string

const (
	SubscriptionTriggerFeatureBaselinePromoteToWidely SubscriptionTrigger = "feature.baseline.promote_to_widely"
	SubscriptionTriggerFeatureBaselinePromoteToNewly  SubscriptionTrigger = "feature.baseline.promote_to_newly"

	TargetWidely = "widely"
	TargetNewly  = "newly"
)

// Directive represents a single parsed TODO(baseline/<id>) AST comment directive.
type Directive struct {
	TargetQuery string              `json:"target_query"`
	Trigger     SubscriptionTrigger `json:"trigger"`
	RawSnippet  string              `json:"raw_snippet"`
	LineNumber  int                 `json:"line_number"`
	FilePath    string              `json:"file_path"`
}

// Regex matching canonical TODO(baseline/<id>) comments.
var baselineTodoRegex = regexp.MustCompile(`(?i)TODO\(baseline/([\w-]+)\)`)

// ParseReader reads from an io.Reader and extracts all valid TODO(baseline/<id>) directives.
func ParseReader(
	r io.Reader,
	filePath string,
	defaultTrigger SubscriptionTrigger,
) ([]Directive, error) {
	var directives []Directive
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	lineNum := 0

	inCBlock := false
	inHTMLBlock := false

	if defaultTrigger == "" {
		defaultTrigger = SubscriptionTriggerFeatureBaselinePromoteToWidely
	}

	for scanner.Scan() {
		lineNum++
		rawLine := scanner.Text()

		hasOpenC := strings.Contains(rawLine, "/*")
		hasCloseC := strings.Contains(rawLine, "*/")
		hasOpenHTML := strings.Contains(rawLine, "<!--")
		hasCloseHTML := strings.Contains(rawLine, "-->")

		if hasOpenC && !hasCloseC {
			inCBlock = true
		}
		if hasOpenHTML && !hasCloseHTML {
			inHTMLBlock = true
		}

		cleaned := cleanCommentLine(rawLine, inCBlock, inHTMLBlock)

		matches := baselineTodoRegex.FindAllStringSubmatch(cleaned, -1)
		for _, m := range matches {
			featureID := m[1]
			directives = append(directives, Directive{
				TargetQuery: "id:" + featureID,
				Trigger:     defaultTrigger,
				RawSnippet:  strings.TrimSpace(rawLine),
				LineNumber:  lineNum,
				FilePath:    filePath,
			})
		}

		if hasCloseC {
			inCBlock = false
		}
		if hasCloseHTML {
			inHTMLBlock = false
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan %s: %w", filePath, err)
	}

	return directives, nil
}

// ParseFileDirectives scans source bytes and extracts all valid TODO(baseline/<id>) directives.
func ParseFileDirectives(
	content []byte,
	filePath string,
	defaultTrigger SubscriptionTrigger,
) []Directive {
	directives, _ := ParseReader(bytes.NewReader(content), filePath, defaultTrigger)

	return directives
}

func cleanCommentLine(rawLine string, inCBlock, inHTMLBlock bool) string {
	trimmed := strings.TrimSpace(rawLine)

	prefixes := []string{"//", "#", "/*", "<!--"}
	for _, p := range prefixes {
		if after, ok := strings.CutPrefix(trimmed, p); ok {
			trimmed = after

			break
		}
	}

	if inCBlock && strings.HasPrefix(trimmed, "*") {
		trimmed = strings.TrimPrefix(trimmed, "*")
	} else if inHTMLBlock && strings.HasPrefix(trimmed, "-->") {
		return ""
	}

	trimmed = strings.TrimSuffix(trimmed, "*/")
	trimmed = strings.TrimSuffix(trimmed, "-->")

	return strings.TrimSpace(trimmed)
}
