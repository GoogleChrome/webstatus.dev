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
	blobs map[string][]byte
}

func (m *mockBlobReader) GetBlob(_ context.Context, _, sha string) ([]byte, error) {
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

	if len(res.Subscriptions) != 3 {
		t.Fatalf("expected 3 subscriptions, got %d", len(res.Subscriptions))
	}
}

func TestScanGitTreeProjectDefaults(t *testing.T) {
	t.Parallel()

	reader := &mockBlobReader{
		blobs: map[string][]byte{
			"sha_config": []byte(`{"target": "2024"}`),
			"sha_code":   []byte("// TODO(baseline/popover): use native popover"),
		},
	}

	treeEntries := []GitTreeEntry{
		{Path: ".baseline.json", Type: "blob", SHA: "sha_config", Size: 50},
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
}

func TestScanGitTreeLargeBlobSkipped(t *testing.T) {
	t.Parallel()

	reader := &mockBlobReader{
		blobs: map[string][]byte{
			"sha_large": []byte("// TODO(baseline/dialog): skipped"),
		},
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
		blobs: map[string][]byte{},
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
		".baseline.json",
		"index.html",
		"src/app.ts",
		"src/styles.css",
	}

	reader := &mockBlobReader{blobs: make(map[string][]byte)}
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
