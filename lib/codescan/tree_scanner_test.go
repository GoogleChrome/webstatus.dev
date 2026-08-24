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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type mockBlobReader struct {
	blobs      map[string][]byte
	calledSHAs []string
}

func (m *mockBlobReader) GetBlob(_ context.Context, _, sha string) ([]byte, error) {
	m.calledSHAs = append(m.calledSHAs, sha)
	content, ok := m.blobs[sha]
	if !ok {
		return nil, fmt.Errorf("blob not found: %s", sha)
	}

	return content, nil
}

func TestScanGitTree(t *testing.T) {
	t.Parallel()

	reader := &mockBlobReader{
		blobs: map[string][]byte{
			"sha_file1": []byte("// TODO(baseline/view-transitions): transition\nconst a = 1;\n// TODO(baseline/subgrid): grid"),
			"sha_file2": []byte("/* TODO(baseline/view-transitions): transition */\nconst b = 2;"),
			"sha_file3": []byte("<!-- TODO(baseline/view-transitions, newly): anim -->"),
		},
		calledSHAs: nil,
	}

	treeEntries := []GitTreeEntry{
		{Path: "src/file1.ts", Type: "blob", SHA: "sha_file1", Size: 100},
		{Path: "src/file2.js", Type: "blob", SHA: "sha_file2", Size: 100},
		{Path: "public/index.html", Type: "blob", SHA: "sha_file3", Size: 100},
		{Path: "ignored.png", Type: "blob", SHA: "sha_png", Size: 500},
	}

	res, err := ScanGitTree(
		context.Background(),
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err != nil {
		t.Fatalf("ScanGitTree failed: %v", err)
	}

	if res.ScanStatus != ScanStatusSuccess {
		t.Errorf("res.ScanStatus = %v, want Success", res.ScanStatus)
	}
	if res.FilesScanned != 3 {
		t.Errorf("res.FilesScanned = %d, want 3", res.FilesScanned)
	}
	if res.DirectivesFound != 4 {
		t.Errorf("res.DirectivesFound = %d, want 4", res.DirectivesFound)
	}

	if len(res.Subscriptions) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(res.Subscriptions))
	}

	// Subscriptions are sorted deterministically by TargetQuery:
	// [0] -> "id:subgrid", [1] -> "id:view-transitions"
	subgrid := res.Subscriptions[0]
	if subgrid.TargetQuery != "id:subgrid" {
		t.Errorf("subgrid.TargetQuery = %s, want id:subgrid", subgrid.TargetQuery)
	}
	if len(subgrid.Occurrences) != 1 {
		t.Errorf("subgrid.Occurrences len = %d, want 1", len(subgrid.Occurrences))
	}

	vt := res.Subscriptions[1]
	if vt.TargetQuery != "id:view-transitions" {
		t.Errorf("vt.TargetQuery = %s, want id:view-transitions", vt.TargetQuery)
	}
	if len(vt.Occurrences) != 3 {
		t.Errorf("vt.Occurrences len = %d, want 3", len(vt.Occurrences))
	}
	if len(vt.Triggers) != 2 {
		t.Errorf("vt.Triggers len = %d, want 2 triggers, got %v", len(vt.Triggers), vt.Triggers)
	}
}

func TestScanGitTreeProjectDefaults(t *testing.T) {
	t.Parallel()

	reader := &mockBlobReader{
		blobs: map[string][]byte{
			"sha_config": []byte("baseline newly available\n"),
			"sha_code":   []byte("// TODO(baseline/popover): use native popover"),
		},
		calledSHAs: nil,
	}

	treeEntries := []GitTreeEntry{
		{Path: ".browserslistrc", Type: "blob", SHA: "sha_config", Size: 50},
		{Path: "src/main.ts", Type: "blob", SHA: "sha_code", Size: 100},
	}

	res, err := ScanGitTree(
		context.Background(),
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err != nil {
		t.Fatalf("ScanGitTree failed: %v", err)
	}

	if len(res.Subscriptions) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(res.Subscriptions))
	}

	sub := res.Subscriptions[0]
	if sub.TargetQuery != "id:popover" {
		t.Errorf("sub.TargetQuery = %s, want id:popover", sub.TargetQuery)
	}
	if len(sub.Triggers) != 1 || sub.Triggers[0] != SubscriptionTriggerFeatureBaselinePromoteToNewly {
		t.Errorf("sub.Triggers = %v, want newly trigger", sub.Triggers)
	}
}

func TestScanGitTreeLargeBlobSkipped(t *testing.T) {
	t.Parallel()

	reader := &mockBlobReader{
		blobs: map[string][]byte{
			"sha_large": []byte("// TODO(baseline/dialog): skipped"),
		},
		calledSHAs: nil,
	}

	treeEntries := []GitTreeEntry{
		{Path: "src/huge.ts", Type: "blob", SHA: "sha_large", Size: 2 * 1024 * 1024}, // 2MB > 1MB limit
	}

	res, err := ScanGitTree(
		context.Background(),
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err != nil {
		t.Fatalf("ScanGitTree failed: %v", err)
	}

	if res.FilesScanned != 0 {
		t.Errorf("res.FilesScanned = %d, want 0 (skipped due to size)", res.FilesScanned)
	}
	if len(res.Subscriptions) != 0 {
		t.Errorf("expected 0 subscriptions for skipped file, got %d", len(res.Subscriptions))
	}
}

func TestScanGitTreeMaxScannedFiles(t *testing.T) {
	t.Parallel()

	reader := &mockBlobReader{
		blobs:      map[string][]byte{},
		calledSHAs: nil,
	}

	// Create 10001 entries
	treeEntries := make([]GitTreeEntry, MaxScannedFiles+1)
	for i := range treeEntries {
		treeEntries[i] = GitTreeEntry{
			Path: fmt.Sprintf("src/file%d.ts", i),
			Type: "blob",
			SHA:  fmt.Sprintf("sha_%d", i),
			Size: 50,
		}
		reader.blobs[fmt.Sprintf("sha_%d", i)] = []byte("const x = 1;")
	}

	res, err := ScanGitTree(
		context.Background(),
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err != nil {
		t.Fatalf("ScanGitTree failed: %v", err)
	}

	if res.ScanStatus != ScanStatusTruncated {
		t.Errorf("res.ScanStatus = %v, want ScanStatusTruncated", res.ScanStatus)
	}
	if res.FilesScanned != int64(MaxScannedFiles) {
		t.Errorf("res.FilesScanned = %d, want %d", res.FilesScanned, MaxScannedFiles)
	}
}

func TestScanGitTree_ArchetypeFixtures(t *testing.T) {
	t.Parallel()

	archDir := filepath.Join("testdata", "repos", "standard-spa")
	expectedPath := filepath.Join(archDir, "expected.json")

	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed reading %s: %v", expectedPath, err)
	}

	var manifest archetypeManifest
	if err := json.Unmarshal(expectedBytes, &manifest); err != nil {
		t.Fatalf("failed parsing %s: %v", expectedPath, err)
	}

	totalExpectedDirectives := 0
	for _, fileExp := range manifest.Files {
		totalExpectedDirectives += len(fileExp.ExpectedDirectives)
	}

	files := []string{
		".browserslistrc",
		"index.html",
		"src/app.ts",
		"src/styles.css",
	}

	reader := &mockBlobReader{blobs: make(map[string][]byte), calledSHAs: nil}
	treeEntries := make([]GitTreeEntry, 0, len(files))

	for i, f := range files {
		cleanF := filepath.Clean(f)
		fullPath := filepath.Join(archDir, cleanF)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("failed reading %s: %v", fullPath, err)
		}
		sha := fmt.Sprintf("sha_%d", i)
		reader.blobs[sha] = content
		treeEntries = append(treeEntries, GitTreeEntry{
			Path: cleanF,
			Type: "blob",
			SHA:  sha,
			Size: int64(len(content)),
		})
	}

	res, err := ScanGitTree(
		context.Background(),
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err != nil {
		t.Fatalf("ScanGitTree failed: %v", err)
	}

	if res.ScanStatus != ScanStatusSuccess {
		t.Errorf("res.ScanStatus = %v, want Success", res.ScanStatus)
	}
	if res.FilesScanned != 3 {
		t.Errorf("res.FilesScanned = %d, want 3", res.FilesScanned)
	}
	if res.DirectivesFound != int64(totalExpectedDirectives) {
		t.Errorf("res.DirectivesFound = %d, want %d", res.DirectivesFound, totalExpectedDirectives)
	}
}

func TestIsSupportedWebExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ext  string
		want bool
	}{
		// Web languages & components (allowed)
		{ext: ".js", want: true},
		{ext: ".JS", want: true},
		{ext: ".jsx", want: true},
		{ext: ".ts", want: true},
		{ext: ".TSX", want: true},
		{ext: ".mjs", want: true},
		{ext: ".cjs", want: true},
		{ext: ".css", want: true},
		{ext: ".scss", want: true},
		{ext: ".html", want: true},
		{ext: ".htm", want: true},
		{ext: ".vue", want: true},
		{ext: ".svelte", want: true},
		{ext: ".astro", want: true},

		// Non-web / backend / binary extensions (rejected)
		{ext: ".pas", want: false},
		{ext: ".py", want: false},
		{ext: ".go", want: false},
		{ext: ".rs", want: false},
		{ext: ".c", want: false},
		{ext: ".cpp", want: false},
		{ext: ".java", want: false},
		{ext: ".md", want: false},
		{ext: ".txt", want: false},
		{ext: ".png", want: false},
		{ext: ".exe", want: false},
		{ext: "", want: false},
	}

	for _, tc := range tests {
		got := IsSupportedWebExtension(tc.ext)
		if got != tc.want {
			t.Errorf("IsSupportedWebExtension(%q) = %v, want %v", tc.ext, got, tc.want)
		}
	}

	exts := SupportedWebExtensions()
	if len(exts) != 13 {
		t.Errorf("SupportedWebExtensions() returned %d items, want 13", len(exts))
	}
}

func TestScanGitTree_NonWebFilesSkipped(t *testing.T) {
	t.Parallel()

	reader := &mockBlobReader{
		blobs: map[string][]byte{
			"sha_ts":  []byte("// TODO(baseline/view-transitions): transition\nconst a = 1;"),
			"sha_pas": []byte("{ TODO(baseline/popover): pascal comment }"),
			"sha_py":  []byte("# TODO(baseline/subgrid): python comment"),
			"sha_go":  []byte("// TODO(baseline/dialog): go comment"),
		},
		calledSHAs: nil,
	}

	treeEntries := []GitTreeEntry{
		{Path: "src/app.ts", Type: "blob", SHA: "sha_ts", Size: 100},
		{Path: "legacy/calc.pas", Type: "blob", SHA: "sha_pas", Size: 100},
		{Path: "server/backend.py", Type: "blob", SHA: "sha_py", Size: 100},
		{Path: "cmd/main.go", Type: "blob", SHA: "sha_go", Size: 100},
	}

	res, err := ScanGitTree(
		context.Background(),
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err != nil {
		t.Fatalf("ScanGitTree failed: %v", err)
	}

	if res.FilesScanned != 1 {
		t.Errorf("res.FilesScanned = %d, want 1 (only .ts scanned, non-web files skipped)", res.FilesScanned)
	}
	if res.DirectivesFound != 1 {
		t.Errorf("res.DirectivesFound = %d, want 1", res.DirectivesFound)
	}
	if len(res.Subscriptions) != 1 || res.Subscriptions[0].TargetQuery != "id:view-transitions" {
		t.Errorf("unexpected subscriptions: %+v", res.Subscriptions)
	}
}

func TestScanGitTree_InaccurateTreeSizePostFetchGuard(t *testing.T) {
	t.Parallel()

	// Simulates tree entry advertising Size: 0 (or small size), but reader returns > 1MB blob
	hugeContent := make([]byte, 2*1024*1024) // 2MB
	copy(hugeContent, []byte("// TODO(baseline/popover): huge file\n"))

	reader := &mockBlobReader{
		blobs: map[string][]byte{
			"sha_misleading": hugeContent,
			"sha_normal":     []byte("// TODO(baseline/subgrid): normal file"),
		},
		calledSHAs: nil,
	}

	treeEntries := []GitTreeEntry{
		{Path: "src/huge.ts", Type: "blob", SHA: "sha_misleading", Size: 0}, // 0 size in tree metadata
		{Path: "src/valid.ts", Type: "blob", SHA: "sha_normal", Size: 50},
	}

	res, err := ScanGitTree(
		context.Background(),
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err != nil {
		t.Fatalf("ScanGitTree failed: %v", err)
	}

	// The huge file should have been skipped post-fetch, only valid.ts scanned
	if res.FilesScanned != 1 {
		t.Errorf("res.FilesScanned = %d, want 1", res.FilesScanned)
	}
	if res.DirectivesFound != 1 {
		t.Errorf("res.DirectivesFound = %d, want 1", res.DirectivesFound)
	}
	if len(res.Warnings) != 1 {
		t.Errorf("expected 1 warning for skipped huge blob, got %d: %v", len(res.Warnings), res.Warnings)
	}
}

func TestScanGitTree_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	reader := &mockBlobReader{
		blobs: map[string][]byte{
			"sha1": []byte("// TODO(baseline/popover): text"),
		},
		calledSHAs: nil,
	}

	treeEntries := []GitTreeEntry{
		{Path: "src/file1.ts", Type: "blob", SHA: "sha1", Size: 100},
	}

	_, err := ScanGitTree(
		ctx,
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err == nil {
		t.Fatalf("expected context cancellation error, got nil")
	}
}

func TestScanGitTree_BlobFetchErrorWarning(t *testing.T) {
	t.Parallel()

	reader := &mockBlobReader{
		blobs: map[string][]byte{
			"sha_valid": []byte("// TODO(baseline/popover): valid"),
			// sha_missing is deliberately not in blobs map to simulate 404/network failure
		},
		calledSHAs: nil,
	}

	treeEntries := []GitTreeEntry{
		{Path: "src/missing.ts", Type: "blob", SHA: "sha_missing", Size: 100},
		{Path: "src/valid.ts", Type: "blob", SHA: "sha_valid", Size: 100},
	}

	res, err := ScanGitTree(
		context.Background(),
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err != nil {
		t.Fatalf("ScanGitTree failed: %v", err)
	}

	if res.FilesScanned != 1 {
		t.Errorf("res.FilesScanned = %d, want 1", res.FilesScanned)
	}
	if len(res.Warnings) != 1 {
		t.Errorf("expected 1 warning for missing blob fetch, got %d: %v", len(res.Warnings), res.Warnings)
	}
}

func TestScanGitTree_PreFetchTotalBytesCheck(t *testing.T) {
	t.Parallel()

	reader := &mockBlobReader{
		blobs:      map[string][]byte{},
		calledSHAs: nil,
	}

	blobContent := make([]byte, 600*1024) // 600KB (< 1MB MaxBlobSizeBytes)

	// 175 entries * 600KB = 105MB (> 100MB MaxScanTotalBytes)
	numEntries := 175
	treeEntries := make([]GitTreeEntry, numEntries)
	for i := range treeEntries {
		sha := fmt.Sprintf("sha_%d", i)
		reader.blobs[sha] = blobContent
		treeEntries[i] = GitTreeEntry{
			Path: fmt.Sprintf("src/file%d.ts", i),
			Type: "blob",
			SHA:  sha,
			Size: 600 * 1024,
		}
	}

	res, err := ScanGitTree(
		context.Background(),
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err != nil {
		t.Fatalf("ScanGitTree failed: %v", err)
	}

	if !res.IsTruncated {
		t.Errorf("expected res.IsTruncated = true, got false")
	}
	if res.ScanStatus != ScanStatusTruncated {
		t.Errorf("res.ScanStatus = %v, want ScanStatusTruncated", res.ScanStatus)
	}

	// Assert GetBlob was called for fewer than 175 entries because pre-fetch check stopped the loop
	if len(reader.calledSHAs) >= numEntries {
		t.Errorf("expected GetBlob called for fewer than %d files, got %d", numEntries, len(reader.calledSHAs))
	}
	if len(reader.calledSHAs) != int(res.FilesScanned) {
		t.Errorf("expected calledSHAs (%d) == FilesScanned (%d)", len(reader.calledSHAs), res.FilesScanned)
	}
}

func TestScanGitTree_WarningBounding(t *testing.T) {
	t.Parallel()

	reader := &mockBlobReader{
		blobs:      map[string][]byte{},
		calledSHAs: nil,
	}

	// Create 150 missing entries to generate 150 warnings
	treeEntries := make([]GitTreeEntry, 150)
	for i := range treeEntries {
		treeEntries[i] = GitTreeEntry{
			Path: fmt.Sprintf("src/missing_%d.ts", i),
			Type: "blob",
			SHA:  fmt.Sprintf("sha_missing_%d", i),
			Size: 100,
		}
	}

	res, err := ScanGitTree(
		context.Background(),
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err != nil {
		t.Fatalf("ScanGitTree failed: %v", err)
	}

	// Should be capped at MaxScanWarnings (100) + 1 summary message = 101
	expectedCount := MaxScanWarnings + 1
	if len(res.Warnings) != expectedCount {
		t.Errorf("expected %d warnings (capped + summary), got %d", expectedCount, len(res.Warnings))
	}
}

func TestScanGitTree_MultiTriggerDeduplicationAndSorting(t *testing.T) {
	t.Parallel()

	reader := &mockBlobReader{
		blobs: map[string][]byte{
			"sha_z": []byte("// TODO(baseline/subgrid, widely): widely in z\n" +
				"// TODO(baseline/subgrid, newly): newly in z"),
			"sha_a": []byte("// TODO(baseline/subgrid, newly): newly in a\n" +
				"// TODO(baseline/dialog): dialog in a"),
		},
		calledSHAs: nil,
	}

	treeEntries := []GitTreeEntry{
		{Path: "src/z_last.ts", Type: "blob", SHA: "sha_z", Size: 100},
		{Path: "src/a_first.ts", Type: "blob", SHA: "sha_a", Size: 100},
	}

	res, err := ScanGitTree(
		context.Background(),
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err != nil {
		t.Fatalf("ScanGitTree failed: %v", err)
	}

	if len(res.Subscriptions) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(res.Subscriptions))
	}

	// Subscriptions sorted deterministically: [0] -> "id:dialog", [1] -> "id:subgrid"
	dialogSub := res.Subscriptions[0]
	if dialogSub.TargetQuery != "id:dialog" {
		t.Errorf("expected id:dialog, got %s", dialogSub.TargetQuery)
	}

	subgridSub := res.Subscriptions[1]
	if subgridSub.TargetQuery != "id:subgrid" {
		t.Errorf("expected id:subgrid, got %s", subgridSub.TargetQuery)
	}

	// Triggers should be deduplicated and sorted deterministically: ["newly", "widely"]
	expectedTriggers := []SubscriptionTrigger{
		SubscriptionTriggerFeatureBaselinePromoteToNewly,
		SubscriptionTriggerFeatureBaselinePromoteToWidely,
	}
	if len(subgridSub.Triggers) != 2 {
		t.Fatalf("expected 2 triggers, got %d", len(subgridSub.Triggers))
	}
	for i, want := range expectedTriggers {
		if subgridSub.Triggers[i] != want {
			t.Errorf("trigger[%d] = %s, want %s", i, subgridSub.Triggers[i], want)
		}
	}

	// Occurrences should be sorted deterministically: a_first.ts:1, z_last.ts:1, z_last.ts:2
	if len(subgridSub.Occurrences) != 3 {
		t.Fatalf("expected 3 occurrences, got %d", len(subgridSub.Occurrences))
	}
	if subgridSub.Occurrences[0].FilePath != "src/a_first.ts" {
		t.Errorf("expected first occurrence from a_first.ts, got %s", subgridSub.Occurrences[0].FilePath)
	}
	if subgridSub.Occurrences[1].FilePath != "src/z_last.ts" ||
		subgridSub.Occurrences[1].LineNumber != 1 {
		t.Errorf(
			"expected second occurrence z_last.ts:1, got %s:%d",
			subgridSub.Occurrences[1].FilePath,
			subgridSub.Occurrences[1].LineNumber,
		)
	}
}

func TestScanGitTree_TargetQueryCasingNormalization(t *testing.T) {
	t.Parallel()

	reader := &mockBlobReader{
		blobs: map[string][]byte{
			"sha_1": []byte("// TODO(baseline/SubGrid): uppercase in file 1"),
			"sha_2": []byte("// TODO(baseline/subgrid): lowercase in file 2"),
		},
		calledSHAs: nil,
	}

	treeEntries := []GitTreeEntry{
		{Path: "src/file1.ts", Type: "blob", SHA: "sha_1", Size: 100},
		{Path: "src/file2.ts", Type: "blob", SHA: "sha_2", Size: 100},
	}

	res, err := ScanGitTree(
		context.Background(),
		"github",
		"inst-123",
		"12345",
		"GoogleChrome/webstatus.dev",
		treeEntries,
		reader,
	)
	if err != nil {
		t.Fatalf("ScanGitTree failed: %v", err)
	}

	if len(res.Subscriptions) != 1 {
		t.Fatalf("expected 1 unified subscription after casing normalization, got %d", len(res.Subscriptions))
	}

	if res.Subscriptions[0].TargetQuery != "id:subgrid" {
		t.Errorf("expected normalized id:subgrid, got %s", res.Subscriptions[0].TargetQuery)
	}
	if len(res.Subscriptions[0].Occurrences) != 2 {
		t.Errorf("expected 2 occurrences, got %d", len(res.Subscriptions[0].Occurrences))
	}
}
