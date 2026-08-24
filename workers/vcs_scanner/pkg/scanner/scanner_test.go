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

package scanner

import (
	"context"
	"errors"
	"testing"

	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/google/go-github/v79/github"
)

type mockGitFetcher struct {
	treeResp *github.Tree
	treeErr  error
	blobs    map[string][]byte
	blobErr  error
}

func (m *mockGitFetcher) GetCommitTree(
	_ context.Context,
	_, _, _ string,
) (*github.Tree, error) {
	return m.treeResp, m.treeErr
}

func (m *mockGitFetcher) GetBlobContent(
	_ context.Context,
	_, _, sha string,
) ([]byte, error) {
	if m.blobErr != nil {
		return nil, m.blobErr
	}

	return m.blobs[sha], nil
}

type mockSpannerSyncer struct {
	syncedSubs []gcpspanner.CodeSubscriptionInput
	syncErr    error
	scanLogs   []gcpspanner.CodeSubscriptionScanLog
	logErr     error
}

func (m *mockSpannerSyncer) SynchronizeRepositoryCodeSubscriptions(
	_ context.Context,
	_ gcpspanner.VCSProvider,
	_ string,
	desired []gcpspanner.CodeSubscriptionInput,
) error {
	m.syncedSubs = desired

	return m.syncErr
}

func (m *mockSpannerSyncer) InsertCodeSubscriptionScanLog(
	_ context.Context,
	log gcpspanner.CodeSubscriptionScanLog,
) error {
	m.scanLogs = append(m.scanLogs, log)

	return m.logErr
}

func sampleTask() codescantaskv1.CodeScanTaskEvent {
	return codescantaskv1.CodeScanTaskEvent{
		VCSProvider:        "github",
		VCSInstallationID:  "12345",
		VCSRepositoryID:    "67890",
		RepositoryFullName: "GoogleChrome/webstatus.dev",
		CommitSHA:          "abcdef1234567890abcdef1234567890abcdef12",
		Branch:             "refs/heads/main",
		IsDefaultBranch:    true,
		ModifiedFiles:      nil,
	}
}

func TestProcessTask(t *testing.T) {
	t.Parallel()

	tsFileContent := `
// TODO(baseline/css-subgrid): enable subgrid
function enableSubgrid() {}
`
	cssFileContent := `
/* TODO(baseline/view-transitions, newly): anim */
.page { view-transition-name: root; }
`
	fetcher := &mockGitFetcher{
		treeResp: &github.Tree{
			SHA: nil,
			Entries: []*github.TreeEntry{
				{
					SHA:     new("sha_ts"),
					Path:    new("src/index.ts"),
					Mode:    nil,
					Type:    new("blob"),
					Size:    new(100),
					Content: nil,
					URL:     nil,
				},
				{
					SHA:     new("sha_css"),
					Path:    new("styles/main.css"),
					Mode:    nil,
					Type:    new("blob"),
					Size:    new(100),
					Content: nil,
					URL:     nil,
				},
				{
					SHA:     new("sha_png"),
					Path:    new("assets/logo.png"),
					Mode:    nil,
					Type:    new("blob"),
					Size:    new(500),
					Content: nil,
					URL:     nil,
				},
			},
			Truncated: new(false),
		},
		treeErr: nil,
		blobs: map[string][]byte{
			"sha_ts":  []byte(tsFileContent),
			"sha_css": []byte(cssFileContent),
		},
		blobErr: nil,
	}

	syncer := &mockSpannerSyncer{
		syncedSubs: nil,
		syncErr:    nil,
		scanLogs:   nil,
		logErr:     nil,
	}
	s := NewScanner(fetcher, syncer)

	err := s.ProcessTask(context.Background(), sampleTask())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(syncer.syncedSubs) != 2 {
		t.Fatalf("expected 2 subscriptions synced, got %d", len(syncer.syncedSubs))
	}

	if len(syncer.scanLogs) != 1 {
		t.Fatalf("expected 1 scan log, got %d", len(syncer.scanLogs))
	}
	log := syncer.scanLogs[0]
	if log.ScanStatus != gcpspanner.ScanStatusSuccess {
		t.Errorf("log.ScanStatus = %v, want SUCCESS", log.ScanStatus)
	}
	if log.FilesScanned != 2 {
		t.Errorf("log.FilesScanned = %d, want 2", log.FilesScanned)
	}
	if log.DirectivesFound != 2 {
		t.Errorf("log.DirectivesFound = %d, want 2", log.DirectivesFound)
	}
}

func TestProcessTaskDeletedBranch(t *testing.T) {
	t.Parallel()

	fetcher := &mockGitFetcher{
		treeResp: nil,
		treeErr:  nil,
		blobs:    nil,
		blobErr:  nil,
	}
	syncer := &mockSpannerSyncer{
		syncedSubs: nil,
		syncErr:    nil,
		scanLogs:   nil,
		logErr:     nil,
	}
	s := NewScanner(fetcher, syncer)

	task := sampleTask()
	task.CommitSHA = DeletedBranchSHA

	err := s.ProcessTask(context.Background(), task)
	if !errors.Is(err, ErrDeletedBranch) {
		t.Fatalf("expected ErrDeletedBranch, got %v", err)
	}
}

func TestProcessTaskNonDefaultBranch(t *testing.T) {
	t.Parallel()

	fetcher := &mockGitFetcher{
		treeResp: nil,
		treeErr:  nil,
		blobs:    nil,
		blobErr:  nil,
	}
	syncer := &mockSpannerSyncer{
		syncedSubs: nil,
		syncErr:    nil,
		scanLogs:   nil,
		logErr:     nil,
	}
	s := NewScanner(fetcher, syncer)

	task := sampleTask()
	task.IsDefaultBranch = false

	err := s.ProcessTask(context.Background(), task)
	if !errors.Is(err, ErrNonDefaultBranch) {
		t.Fatalf("expected ErrNonDefaultBranch, got %v", err)
	}
}

func TestProcessTaskTreeFetchError(t *testing.T) {
	t.Parallel()

	fetcher := &mockGitFetcher{
		treeResp: nil,
		treeErr:  errors.New("github tree 500 error"),
		blobs:    nil,
		blobErr:  nil,
	}
	syncer := &mockSpannerSyncer{
		syncedSubs: nil,
		syncErr:    nil,
		scanLogs:   nil,
		logErr:     nil,
	}
	s := NewScanner(fetcher, syncer)

	err := s.ProcessTask(context.Background(), sampleTask())
	if err == nil {
		t.Fatalf("expected tree fetch error, got nil")
	}

	if len(syncer.scanLogs) != 1 {
		t.Fatalf("expected 1 scan log recorded for failed scan, got %d", len(syncer.scanLogs))
	}
	if syncer.scanLogs[0].ScanStatus != gcpspanner.ScanStatusFailed {
		t.Errorf("scan log status = %v, want FAILED", syncer.scanLogs[0].ScanStatus)
	}
}
