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
	"encoding/json"
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

// Directive represents a single parsed @webstatus AST comment directive.
type Directive struct {
	TargetQuery string              `json:"target_query"`
	Trigger     SubscriptionTrigger `json:"trigger"`
	RawSnippet  string              `json:"raw_snippet"`
	LineNumber  int                 `json:"line_number"`
	FilePath    string              `json:"file_path"`
}

// Regex matching @webstatus comment directives.
var webstatusDirectiveRegex = regexp.MustCompile(`(?i)^@webstatus:\s*(.*?)$`)

// Regex matching idiomatic TODO(baseline/<id>) or TODO(baseline/<id>, newly) comments.
var inlineTodoRegex = regexp.MustCompile(
	`(?i)TODO\(` +
		`(?:web-features?:\s*([\w-]+)|baseline/\s*([\w-]+)(?:\s*,\s*(newly|widely))?)\s*\)` +
		`(?::\s*([^,\n]+))?`,
)

// ParseReader reads from an io.Reader and extracts all valid @webstatus directives.
func ParseReader(r io.Reader, filePath string, defaultTarget string) ([]Directive, error) {
	var directives []Directive
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	lineNum := 0

	inCBlock := false
	inHTMLBlock := false

	defaultTrigger := SubscriptionTriggerFeatureBaselinePromoteToWidely
	if defaultTarget == TargetNewly || defaultTarget == string(SubscriptionTriggerFeatureBaselinePromoteToNewly) ||
		defaultTarget == "newly_available" {
		defaultTrigger = SubscriptionTriggerFeatureBaselinePromoteToNewly
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

		if todoDirs := extractTodoDirectives(cleaned, rawLine, lineNum, filePath, defaultTrigger); len(todoDirs) > 0 {
			directives = append(directives, todoDirs...)
		} else if dirs, ok := extractWebstatusDirective(
			cleaned, rawLine, lineNum, filePath, defaultTarget, defaultTrigger,
		); ok {
			directives = append(directives, dirs...)
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

// ParseFileDirectives scans the lines of a source file and extracts all valid @webstatus directives.
func ParseFileDirectives(content []byte, filePath string, defaultTarget string) []Directive {
	directives, _ := ParseReader(bytes.NewReader(content), filePath, defaultTarget)

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

func extractTodoDirectives(
	cleanedLine, rawLine string,
	lineNum int,
	filePath string,
	defaultTrigger SubscriptionTrigger,
) []Directive {
	matches := inlineTodoRegex.FindAllStringSubmatch(cleanedLine, -1)
	if len(matches) == 0 {
		return nil
	}

	var results []Directive
	for _, m := range matches {
		var featureID string
		var mod string

		if m[1] != "" {
			featureID = m[1]
		} else {
			featureID = m[2]
			mod = strings.ToLower(m[3])
		}

		trigger := defaultTrigger
		if trigger == "" {
			trigger = SubscriptionTriggerFeatureBaselinePromoteToWidely
		}
		switch mod {
		case TargetNewly:
			trigger = SubscriptionTriggerFeatureBaselinePromoteToNewly
		case TargetWidely:
			trigger = SubscriptionTriggerFeatureBaselinePromoteToWidely
		}

		results = append(results, Directive{
			TargetQuery: "id:" + featureID,
			Trigger:     trigger,
			RawSnippet:  strings.TrimSpace(rawLine),
			LineNumber:  lineNum,
			FilePath:    filePath,
		})
	}

	return results
}

func extractWebstatusDirective(
	cleanedLine, rawLine string,
	lineNum int,
	filePath, defaultTarget string,
	defaultTrigger SubscriptionTrigger,
) ([]Directive, bool) {
	matches := webstatusDirectiveRegex.FindStringSubmatch(cleanedLine)
	if len(matches) < 2 {
		return nil, false
	}

	annotationBody := matches[1]
	tokens := parseAnnotationTokens(annotationBody)

	var ids []string
	trigger := defaultTrigger

	for _, token := range tokens {
		parts := strings.SplitN(token, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch key {
		case "id":
			val = strings.TrimPrefix(val, "id:")
			ids = append(ids, val)
		case "trigger":
			if strings.EqualFold(val, "newly_available") || strings.EqualFold(val, TargetNewly) {
				trigger = SubscriptionTriggerFeatureBaselinePromoteToNewly
			} else if strings.EqualFold(val, "widely_available") || strings.EqualFold(val, TargetWidely) {
				trigger = SubscriptionTriggerFeatureBaselinePromoteToWidely
			}
		}
	}

	if len(ids) == 0 {
		if defaultTarget != "" {
			cleanID := strings.TrimPrefix(defaultTarget, "id:")
			ids = append(ids, cleanID)
		} else {
			return nil, false
		}
	}

	var results []Directive
	for _, id := range ids {
		targetQuery := id
		if !strings.HasPrefix(id, "id:") && !strings.HasPrefix(id, "group:") {
			targetQuery = "id:" + id
		}
		results = append(results, Directive{
			TargetQuery: targetQuery,
			Trigger:     trigger,
			RawSnippet:  strings.TrimSpace(rawLine),
			LineNumber:  lineNum,
			FilePath:    filePath,
		})
	}

	return results, true
}

func parseAnnotationTokens(body string) []string {
	var tokens []string
	fields := strings.FieldsSeq(body)
	for field := range fields {
		sub := strings.SplitSeq(field, ",")
		for s := range sub {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				tokens = append(tokens, trimmed)
			}
		}
	}

	return tokens
}

// ParseProjectDefaults extracts baseline target from .baseline.json or AGENTS.md.
func ParseProjectDefaults(content []byte, fileName string) (string, error) {
	if fileName == ".baseline.json" {
		var cfg struct {
			Target string `json:"target"`
		}
		if err := json.Unmarshal(content, &cfg); err != nil {
			return "", err
		}

		return cfg.Target, nil
	}

	if fileName == "AGENTS.md" {
		re := regexp.MustCompile(`(?i)(?:@webstatus:\s*target:|baseline\s*target\s*:\s*"?)\s*([\w-]+)"?`)
		matches := re.FindSubmatch(content)
		if len(matches) >= 2 {
			return string(matches[1]), nil
		}
	}

	return "", nil
}
