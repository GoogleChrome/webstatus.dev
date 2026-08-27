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
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseFileDirectives(t *testing.T) {
	t.Parallel()

	sourceCode := `
// Normal comment
const x = 1;
// TODO(baseline/popover): refactor modal to native popover
// TODO(baseline/view-transitions, newly): progressive enhancement
// TODO(web-feature: subgrid): upgrade layout
/* TODO(baseline/anchor-positioning): anchor tooltip */
<!-- TODO(baseline/dialog): replace polyfill -->
// @webstatus: id:view-transitions
# @webstatus: id:subgrid trigger:newly_available
/* @webstatus: id:anchor-positioning trigger:widely_available */
<!-- @webstatus: id:backdrop-filter -->
// @webstatus: id:compression-streams, id:badging
// @webstatus: trigger:newly_available
`

	directives := ParseFileDirectives(
		[]byte(sourceCode),
		"src/app.ts",
		SubscriptionTriggerFeatureBaselinePromoteToWidely,
		"id:default-feature",
	)

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
			RawSnippet:  "// TODO(baseline/view-transitions, newly): progressive enhancement",
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
			RawSnippet:  "/* TODO(baseline/anchor-positioning): anchor tooltip */",
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
	emptyDirectives := ParseFileDirectives(
		[]byte{},
		"src/empty.ts",
		SubscriptionTriggerFeatureBaselinePromoteToWidely,
		"id:default",
	)
	if len(emptyDirectives) != 0 {
		t.Errorf("expected 0 directives for empty buffer, got %d", len(emptyDirectives))
	}

	// 2. Binary bytes (no matching comments)
	binaryData := []byte{0x00, 0xFF, 0xFE, 0x01, 0x7F, 0x80}
	binaryDirectives := ParseFileDirectives(
		binaryData,
		"assets/logo.png",
		SubscriptionTriggerFeatureBaselinePromoteToWidely,
		"id:default",
	)
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

	directives := ParseFileDirectives(
		[]byte(sourceCode),
		"src/multi.ts",
		SubscriptionTriggerFeatureBaselinePromoteToWidely,
		"id:default",
	)
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

		got := ParseFileDirectives(content, cleanRelPath, SubscriptionTriggerFeatureBaselinePromoteToWidely, "id:default")
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

func TestParseFileDirectives_MultiLineBlocks(t *testing.T) {
	t.Parallel()

	sourceCode := `
/*
 * Multi-line CSS comment
 * TODO(baseline/subgrid): convert grid fallback
 * TODO(baseline/has, newly): remove polyfill
 */
const y = 2;
<!--
  Multi-line HTML comment
  TODO(baseline/popover): use native popover
  TODO(baseline/view-transitions, newly): native transitions
-->
/* Same line multi-directive: TODO(baseline/dialog) TODO(baseline/invokers, newly) */
`

	directives := ParseFileDirectives(
		[]byte(sourceCode),
		"src/multi.css",
		SubscriptionTriggerFeatureBaselinePromoteToWidely,
		"",
	)

	expected := []Directive{
		{
			TargetQuery: "id:subgrid",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
			RawSnippet:  "* TODO(baseline/subgrid): convert grid fallback",
			LineNumber:  4,
			FilePath:    "src/multi.css",
		},
		{
			TargetQuery: "id:has",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToNewly,
			RawSnippet:  "* TODO(baseline/has, newly): remove polyfill",
			LineNumber:  5,
			FilePath:    "src/multi.css",
		},
		{
			TargetQuery: "id:popover",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
			RawSnippet:  "TODO(baseline/popover): use native popover",
			LineNumber:  10,
			FilePath:    "src/multi.css",
		},
		{
			TargetQuery: "id:view-transitions",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToNewly,
			RawSnippet:  "TODO(baseline/view-transitions, newly): native transitions",
			LineNumber:  11,
			FilePath:    "src/multi.css",
		},
		{
			TargetQuery: "id:dialog",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
			RawSnippet:  "/* Same line multi-directive: TODO(baseline/dialog) TODO(baseline/invokers, newly) */",
			LineNumber:  13,
			FilePath:    "src/multi.css",
		},
		{
			TargetQuery: "id:invokers",
			Trigger:     SubscriptionTriggerFeatureBaselinePromoteToNewly,
			RawSnippet:  "/* Same line multi-directive: TODO(baseline/dialog) TODO(baseline/invokers, newly) */",
			LineNumber:  13,
			FilePath:    "src/multi.css",
		},
	}

	if len(directives) != len(expected) {
		t.Fatalf("expected %d directives, got %d", len(expected), len(directives))
	}

	for i, want := range expected {
		if !reflect.DeepEqual(directives[i], want) {
			t.Errorf("[%d] got %+v, want %+v", i, directives[i], want)
		}
	}
}

func TestParseReader(t *testing.T) {
	t.Parallel()

	input := "// TODO(baseline/subgrid): layout upgrade\n"
	r := strings.NewReader(input)

	directives, err := ParseReader(r, "stdin.ts", SubscriptionTriggerFeatureBaselinePromoteToWidely, "")
	if err != nil {
		t.Fatalf("ParseReader failed: %v", err)
	}

	if len(directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(directives))
	}

	if directives[0].TargetQuery != "id:subgrid" {
		t.Errorf("expected id:subgrid, got %s", directives[0].TargetQuery)
	}
}

func TestDirective_JSONSerialization(t *testing.T) {
	t.Parallel()

	d := Directive{
		TargetQuery: "id:popover",
		Trigger:     SubscriptionTriggerFeatureBaselinePromoteToWidely,
		RawSnippet:  "// TODO(baseline/popover)",
		LineNumber:  42,
		FilePath:    "src/app.ts",
	}

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	expectedKeys := []string{"target_query", "trigger", "raw_snippet", "line_number", "file_path"}
	for _, key := range expectedKeys {
		if _, ok := parsed[key]; !ok {
			t.Errorf("missing expected json key %q in serialized output", key)
		}
	}
}

func TestParseFileDirectives_DefaultTargetInheritance(t *testing.T) {
	t.Parallel()

	sourceCode := `
// TODO(baseline/popover): standard comment
// TODO(baseline/dialog, newly): explicit override to newly
// TODO(baseline/subgrid, widely): explicit override to widely
// @webstatus: id:anchor-positioning
`

	// Case 1: defaultTrigger is SubscriptionTriggerFeatureBaselinePromoteToNewly
	dirsNewly := ParseFileDirectives(
		[]byte(sourceCode),
		"src/app.ts",
		SubscriptionTriggerFeatureBaselinePromoteToNewly,
		"",
	)
	if len(dirsNewly) != 4 {
		t.Fatalf("got %d directives, want 4", len(dirsNewly))
	}
	if dirsNewly[0].Trigger != SubscriptionTriggerFeatureBaselinePromoteToNewly {
		t.Errorf("dirsNewly[0].Trigger = %s, want newly", dirsNewly[0].Trigger)
	}
	if dirsNewly[1].Trigger != SubscriptionTriggerFeatureBaselinePromoteToNewly {
		t.Errorf("dirsNewly[1].Trigger = %s, want newly", dirsNewly[1].Trigger)
	}
	if dirsNewly[2].Trigger != SubscriptionTriggerFeatureBaselinePromoteToWidely {
		t.Errorf("dirsNewly[2].Trigger = %s, want widely (explicit override)", dirsNewly[2].Trigger)
	}
	if dirsNewly[3].Trigger != SubscriptionTriggerFeatureBaselinePromoteToNewly {
		t.Errorf("dirsNewly[3].Trigger = %s, want newly (inherited from default)", dirsNewly[3].Trigger)
	}

	// Case 2: defaultTrigger is SubscriptionTriggerFeatureBaselinePromoteToWidely
	dirsWidely := ParseFileDirectives(
		[]byte(sourceCode),
		"src/app.ts",
		SubscriptionTriggerFeatureBaselinePromoteToWidely,
		"",
	)
	if len(dirsWidely) != 4 {
		t.Fatalf("got %d directives, want 4", len(dirsWidely))
	}
	if dirsWidely[0].Trigger != SubscriptionTriggerFeatureBaselinePromoteToWidely {
		t.Errorf("dirsWidely[0].Trigger = %s, want widely", dirsWidely[0].Trigger)
	}
	if dirsWidely[1].Trigger != SubscriptionTriggerFeatureBaselinePromoteToNewly {
		t.Errorf("dirsWidely[1].Trigger = %s, want newly (explicit override)", dirsWidely[1].Trigger)
	}
	if dirsWidely[2].Trigger != SubscriptionTriggerFeatureBaselinePromoteToWidely {
		t.Errorf("dirsWidely[2].Trigger = %s, want widely", dirsWidely[2].Trigger)
	}
	if dirsWidely[3].Trigger != SubscriptionTriggerFeatureBaselinePromoteToWidely {
		t.Errorf("dirsWidely[3].Trigger = %s, want widely (inherited from default)", dirsWidely[3].Trigger)
	}

	// Case 3: Bare @webstatus with defaultTargetQuery fallback
	bareSource := "// @webstatus: trigger:newly_available\n"
	dirsBare := ParseFileDirectives(
		[]byte(bareSource),
		"src/app.ts",
		SubscriptionTriggerFeatureBaselinePromoteToWidely,
		"id:fallback-feature",
	)
	if len(dirsBare) != 1 {
		t.Fatalf("got %d directives, want 1", len(dirsBare))
	}
	if dirsBare[0].TargetQuery != "id:fallback-feature" {
		t.Errorf("dirsBare[0].TargetQuery = %s, want id:fallback-feature", dirsBare[0].TargetQuery)
	}
	if dirsBare[0].Trigger != SubscriptionTriggerFeatureBaselinePromoteToNewly {
		t.Errorf("dirsBare[0].Trigger = %s, want newly_available", dirsBare[0].Trigger)
	}
}

type errReader struct{}

func (e errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("simulated read error")
}

func TestParseReader_ScannerError(t *testing.T) {
	t.Parallel()

	_, err := ParseReader(errReader{}, "test.ts", SubscriptionTriggerFeatureBaselinePromoteToWidely, "")
	if err == nil {
		t.Fatal("expected error from errReader, got nil")
	}
	if !strings.Contains(err.Error(), "simulated read error") {
		t.Errorf("unexpected error message: %v", err)
	}
}
