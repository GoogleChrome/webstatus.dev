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
	"path/filepath"
	"strings"
)

const (
	BrowserslistBaselineWidely = "baseline widely available"
	BrowserslistBaselineNewly  = "baseline newly available"
)

// ParseBrowserslistLine evaluates a single query line from a browserslist configuration.
// It returns the recognized baseline target if exact ("widely" or "newly"),
// a warning message if an unsupported baseline query is detected (no default applied),
// and a boolean indicating whether the line matched a baseline query.
func ParseBrowserslistLine(line, sourceFile string) (target string, warning string, isBaseline bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}

	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "baseline") {
		return "", "", false
	}

	switch lower {
	case BrowserslistBaselineWidely:
		return TargetWidely, "", true
	case BrowserslistBaselineNewly:
		return TargetNewly, "", true
	default:
		warning = fmt.Sprintf(
			"unsupported browserslist baseline query %q in %s: "+
				"webstatus.dev currently only supports %q and %q. "+
				"No project-level default was applied.",
			trimmed, sourceFile, BrowserslistBaselineWidely, BrowserslistBaselineNewly,
		)

		return "", warning, true
	}
}

// IsBrowserslistFile reports whether the filename is a standard browserslist configuration file.
func IsBrowserslistFile(fileName string) bool {
	base := filepath.Base(fileName)
	switch base {
	case ".browserslistrc", ".browserslist", "browserslist", "package.json":
		return true
	default:
		return false
	}
}

// ParseBrowserslistConfig parses content from a .browserslistrc or package.json file.
func ParseBrowserslistConfig(content []byte, fileName string) (target string, warnings []string, err error) {
	base := filepath.Base(fileName)
	if base == "package.json" {
		return parsePackageJSONBrowserslist(content, fileName)
	}

	return parseBrowserslistRC(content, fileName)
}

func parseBrowserslistRC(content []byte, fileName string) (string, []string, error) {
	var target string
	var warnings []string

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}
		t, w, isBaseline := ParseBrowserslistLine(line, fileName)
		if w != "" {
			warnings = append(warnings, w)
		}
		if isBaseline && t != "" && target == "" {
			target = t
		}
	}

	return target, warnings, scanner.Err()
}

func parsePackageJSONBrowserslist(content []byte, fileName string) (string, []string, error) {
	var pkg struct {
		Browserslist any `json:"browserslist"`
	}
	if err := json.Unmarshal(content, &pkg); err != nil {
		return "", nil, err
	}

	if pkg.Browserslist == nil {
		return "", nil, nil
	}

	items := extractBrowserslistItems(pkg.Browserslist)

	var target string
	var warnings []string

	for _, item := range items {
		t, w, isBaseline := ParseBrowserslistLine(item, fileName)
		if w != "" {
			warnings = append(warnings, w)
		}
		if isBaseline && t != "" && target == "" {
			target = t
		}
	}

	return target, warnings, nil
}

func extractBrowserslistItems(raw any) []string {
	switch v := raw.(type) {
	case string:
		return splitQueryString(v)
	case []any:
		return toStringSlice(v)
	case map[string]any:
		if prod, ok := v["production"]; ok {
			return extractBrowserslistItems(prod)
		}
		var all []string
		for _, envVal := range v {
			all = append(all, extractBrowserslistItems(envVal)...)
		}

		return all
	default:
		return nil
	}
}

func splitQueryString(s string) []string {
	var items []string
	for line := range strings.SplitSeq(s, "\n") {
		for item := range strings.SplitSeq(line, ",") {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				items = append(items, trimmed)
			}
		}
	}

	return items
}

func toStringSlice(slice []any) []string {
	var items []string
	for _, item := range slice {
		if str, ok := item.(string); ok {
			trimmed := strings.TrimSpace(str)
			if trimmed != "" {
				items = append(items, trimmed)
			}
		}
	}

	return items
}
