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
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseFileDirectives(t *testing.T) {
	t.Parallel()

	sourceCode := `
// Normal comment
const x = 1;
// TODO(baseline/popover): refactor modal to native popover
// TODO(baseline-newly/view-transitions): progressive enhancement
// TODO(web-feature: subgrid): upgrade layout
/* TODO(baseline-widely/anchor-positioning): anchor tooltip */
<!-- TODO(baseline/dialog): replace polyfill -->
// @webstatus: id:view-transitions
# @webstatus: id:subgrid trigger:newly_available
/* @webstatus: id:anchor-positioning trigger:widely_available */
<!-- @webstatus: id:backdrop-filter -->
// @webstatus: id:compression-streams, id:badging
// @webstatus: trigger:newly_available
`

	directives := ParseFileDirectives([]byte(sourceCode), "src/app.ts", "id:default-feature")

	expected := []Directive{
		{
			TargetQuery: "id:popover",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
			RawSnippet:  "// TODO(baseline/popover): refactor modal to native popover",
			LineNumber:  4,
			FilePath:    "src/app.ts",
		},
		{
			TargetQuery: "id:view-transitions",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToNewly,
			RawSnippet:  "// TODO(baseline-newly/view-transitions): progressive enhancement",
			LineNumber:  5,
			FilePath:    "src/app.ts",
		},
		{
			TargetQuery: "id:subgrid",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
			RawSnippet:  "// TODO(web-feature: subgrid): upgrade layout",
			LineNumber:  6,
			FilePath:    "src/app.ts",
		},
		{
			TargetQuery: "id:anchor-positioning",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
			RawSnippet:  "/* TODO(baseline-widely/anchor-positioning): anchor tooltip */",
			LineNumber:  7,
			FilePath:    "src/app.ts",
		},
		{
			TargetQuery: "id:dialog",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
			RawSnippet:  "<!-- TODO(baseline/dialog): replace polyfill -->",
			LineNumber:  8,
			FilePath:    "src/app.ts",
		},
		{
			TargetQuery: "id:view-transitions",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
			RawSnippet:  "// @webstatus: id:view-transitions",
			LineNumber:  9,
			FilePath:    "src/app.ts",
		},
		{
			TargetQuery: "id:subgrid",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToNewly,
			RawSnippet:  "# @webstatus: id:subgrid trigger:newly_available",
			LineNumber:  10,
			FilePath:    "src/app.ts",
		},
		{
			TargetQuery: "id:anchor-positioning",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
			RawSnippet:  "/* @webstatus: id:anchor-positioning trigger:widely_available */",
			LineNumber:  11,
			FilePath:    "src/app.ts",
		},
		{
			TargetQuery: "id:backdrop-filter",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
			RawSnippet:  "<!-- @webstatus: id:backdrop-filter -->",
			LineNumber:  12,
			FilePath:    "src/app.ts",
		},
		{
			TargetQuery: "id:compression-streams",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
			RawSnippet:  "// @webstatus: id:compression-streams, id:badging",
			LineNumber:  13,
			FilePath:    "src/app.ts",
		},
		{
			TargetQuery: "id:badging",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
			RawSnippet:  "// @webstatus: id:compression-streams, id:badging",
			LineNumber:  13,
			FilePath:    "src/app.ts",
		},
		{
			TargetQuery: "id:default-feature",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToNewly,
			RawSnippet:  "// @webstatus: trigger:newly_available",
			LineNumber:  14,
			FilePath:    "src/app.ts",
		},
	}

	if len(directives) != len(expected) {
		t.Fatalf("got %d directives, want %d", len(directives), len(expected))
	}

	for i := range expected {
		if !reflect.DeepEqual(directives[i], expected[i]) {
			t.Errorf("directive[%d] mismatch:\ngot  %+v\nwant %+v", i, directives[i], expected[i])
		}
	}
}

func TestParseProjectDefaults(t *testing.T) {
	t.Parallel()

	// 1. .baseline.json
	baselineJSON := []byte(`{"target": "group:css-anchor", "trigger": "newly_available"}`)
	target, err := ParseProjectDefaults(baselineJSON, ".baseline.json")
	if err != nil {
		t.Fatalf("unexpected error parsing .baseline.json: %v", err)
	}
	if target != "group:css-anchor" {
		t.Errorf("target = %q, want %q", target, "group:css-anchor")
	}

	// 2. AGENTS.md
	agentsMD := []byte(`# AI Agent Guidelines
<!-- @webstatus: target:widely_available -->
`)
	targetMD, err := ParseProjectDefaults(agentsMD, "AGENTS.md")
	if err != nil {
		t.Fatalf("unexpected error parsing AGENTS.md: %v", err)
	}
	if targetMD != "widely_available" {
		t.Errorf("targetMD = %q, want %q", targetMD, "widely_available")
	}

	// 3. Invalid JSON error
	invalidJSON := []byte(`{invalid-json-content}`)
	_, err = ParseProjectDefaults(invalidJSON, ".baseline.json")
	if err == nil {
		t.Errorf("expected error parsing invalid .baseline.json, got nil")
	}
}

func TestParseFileDirectivesEmptyAndBinary(t *testing.T) {
	t.Parallel()

	// 1. Empty buffer
	emptyDirectives := ParseFileDirectives([]byte{}, "src/empty.ts", "id:default")
	if len(emptyDirectives) != 0 {
		t.Errorf("expected 0 directives for empty buffer, got %d", len(emptyDirectives))
	}

	// 2. Binary bytes (no matching comments)
	binaryData := []byte{0x00, 0xFF, 0xFE, 0x01, 0x7F, 0x80}
	binaryDirectives := ParseFileDirectives(binaryData, "assets/logo.png", "id:default")
	if len(binaryDirectives) != 0 {
		t.Errorf("expected 0 directives for binary data, got %d", len(binaryDirectives))
	}
}

func TestParseFileDirectives_MultiLineComments(t *testing.T) {
	t.Parallel()

	sourceCode := `
/*
 * TODO(baseline/subgrid): Replace nested grid hack
 * with native CSS subgrid once widely available.
 */
const y = 2;
<!--
  TODO(baseline/view-transitions): Multi-line
  HTML block comment transition handler
-->
/*
 * @webstatus: id:anchor-positioning trigger:newly_available
 * Multiline webstatus comment
 */
`

	directives := ParseFileDirectives([]byte(sourceCode), "src/multi.ts", "id:default")
	if len(directives) != 3 {
		t.Fatalf("got %d directives, want 3", len(directives))
	}

	if directives[0].TargetQuery != "id:subgrid" ||
		directives[0].Trigger != SubscriptionTriggerFeatureBaselinePromoteToWidely ||
		directives[0].LineNumber != 3 {
		t.Errorf("unexpected directive[0]: %+v", directives[0])
	}
	if directives[1].TargetQuery != "id:view-transitions" ||
		directives[1].Trigger != SubscriptionTriggerFeatureBaselinePromoteToWidely ||
		directives[1].LineNumber != 8 {
		t.Errorf("unexpected directive[1]: %+v", directives[1])
	}
	if directives[2].TargetQuery != "id:anchor-positioning" ||
		directives[2].Trigger != SubscriptionTriggerFeatureBaselinePromoteToNewly ||
		directives[2].LineNumber != 12 {
		t.Errorf("unexpected directive[2]: %+v", directives[2])
	}
}

type expectedFixtureDirective struct {
	FilePath    string              `json:"file_path"`
	TargetQuery string              `json:"target_query"`
	Trigger     SubscriptionTrigger `json:"trigger"`
	LineNumber  int                 `json:"line_number"`
	RawSnippet  string              `json:"raw_snippet"`
}

type archetypeFileExpectation struct {
	Purpose            string                     `json:"purpose"`
	ExpectedDirectives []expectedFixtureDirective `json:"expectedDirectives"`
}

type archetypeManifest struct {
	Scenario    string                              `json:"scenario"`
	Description string                              `json:"description"`
	Files       map[string]archetypeFileExpectation `json:"files"`
}

func assertDirectiveMatch(t *testing.T, relPath string, idx int, got Directive, want expectedFixtureDirective) {
	t.Helper()
	if got.TargetQuery != want.TargetQuery {
		t.Errorf("%s[%d] TargetQuery = %s, want %s", relPath, idx, got.TargetQuery, want.TargetQuery)
	}
	if got.Trigger != want.Trigger {
		t.Errorf("%s[%d] Trigger = %s, want %s", relPath, idx, got.Trigger, want.Trigger)
	}
	if got.LineNumber != want.LineNumber {
		t.Errorf("%s[%d] LineNumber = %d, want %d", relPath, idx, got.LineNumber, want.LineNumber)
	}
	if got.RawSnippet != want.RawSnippet {
		t.Errorf("%s[%d] RawSnippet = %q, want %q", relPath, idx, got.RawSnippet, want.RawSnippet)
	}
}

func verifyArchetypeDirectives(t *testing.T, arch string) {
	t.Helper()
	archDir := filepath.Join("testdata", "repos", arch)
	expectedPath := filepath.Join(archDir, "expected.json")

	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed reading %s: %v", expectedPath, err)
	}

	var manifest archetypeManifest
	if err := json.Unmarshal(expectedBytes, &manifest); err != nil {
		t.Fatalf("failed parsing %s: %v", expectedPath, err)
	}

	// 1. Verify all manifested files match their expected directives
	for relPath, fileExpectation := range manifest.Files {
		cleanRelPath := filepath.Clean(relPath)
		fullPath := filepath.Join(archDir, cleanRelPath)

		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("failed reading manifested file %s: %v", fullPath, err)
		}

		got := ParseFileDirectives(content, cleanRelPath, "id:default")
		if len(got) != len(fileExpectation.ExpectedDirectives) {
			t.Errorf("%s: got %d directives, want %d", cleanRelPath, len(got), len(fileExpectation.ExpectedDirectives))

			continue
		}

		for i, want := range fileExpectation.ExpectedDirectives {
			assertDirectiveMatch(t, cleanRelPath, i, got[i], want)
		}
	}

	// 2. Ensure every file in the directory is documented in manifest.Files
	err = filepath.WalkDir(archDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}

		relPath, err := filepath.Rel(archDir, path)
		if err != nil {
			return err
		}
		if relPath == "expected.json" {
			return nil
		}

		if _, ok := manifest.Files[relPath]; !ok {
			t.Errorf("unregistered fixture file found on disk: %s in archetype %s", relPath, arch)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("failed walking %s: %v", archDir, err)
	}
}

func TestParseFileDirectives_ArchetypeFixtures(t *testing.T) {
	t.Parallel()

	archetypes := []string{
		"standard-spa",
		"monorepo-workspaces",
		"sfc-components",
	}

	for _, arch := range archetypes {
		t.Run(arch, func(t *testing.T) {
			t.Parallel()
			verifyArchetypeDirectives(t, arch)
		})
	}
}
