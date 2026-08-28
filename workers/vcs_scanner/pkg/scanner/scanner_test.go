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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/google/go-github/v79/github"
)

type mockTarballFetcher struct {
	archiveData []byte
	err         error
}

func (m *mockTarballFetcher) DownloadTarball(_ context.Context, _, _, _ string) (io.ReadCloser, error) {
	if m.err != nil {
		return nil, m.err
	}

	return io.NopCloser(bytes.NewReader(m.archiveData)), nil
}

type mockTokenProvider struct {
	token string
	err   error
}

func (m *mockTokenProvider) GetInstallationToken(_ context.Context, _ string) (string, error) {
	return m.token, m.err
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

func createTarGzArchive(files map[string][]byte) []byte {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		//nolint:exhaustruct
		hdr := &tar.Header{
			Name:     "repo-root-dir/" + name,
			Mode:     0o600,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			panic(err)
		}
		if _, err := tw.Write(content); err != nil {
			panic(err)
		}
	}

	if err := tw.Close(); err != nil {
		panic(err)
	}
	if err := gzw.Close(); err != nil {
		panic(err)
	}

	return buf.Bytes()
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
/* TODO(baseline/view-transitions): anim */
.page { view-transition-name: root; }
`
	archiveBytes := createTarGzArchive(map[string][]byte{
		"src/index.ts":    []byte(tsFileContent),
		"styles/main.css": []byte(cssFileContent),
		"assets/logo.png": []byte("fake-binary-png-data"),
	})

	fetcher := &mockTarballFetcher{archiveData: archiveBytes, err: nil}
	tp := &mockTokenProvider{token: "mock-token", err: nil}
	syncer := &mockSpannerSyncer{
		syncedSubs: nil,
		syncErr:    nil,
		scanLogs:   nil,
		logErr:     nil,
	}

	s := NewScanner(tp, func(_ string) TarballFetcher { return fetcher }, syncer)

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

	fetcher := &mockTarballFetcher{archiveData: nil, err: nil}
	syncer := &mockSpannerSyncer{
		syncedSubs: nil,
		syncErr:    nil,
		scanLogs:   nil,
		logErr:     nil,
	}
	s := NewScanner(nil, func(_ string) TarballFetcher { return fetcher }, syncer)

	task := sampleTask()
	task.CommitSHA = DeletedBranchSHA

	err := s.ProcessTask(context.Background(), task)
	if err != nil {
		t.Fatalf("expected nil error on deleted branch, got %v", err)
	}
}

func TestProcessTaskNonDefaultBranch(t *testing.T) {
	t.Parallel()

	fetcher := &mockTarballFetcher{archiveData: nil, err: nil}
	syncer := &mockSpannerSyncer{
		syncedSubs: nil,
		syncErr:    nil,
		scanLogs:   nil,
		logErr:     nil,
	}
	s := NewScanner(nil, func(_ string) TarballFetcher { return fetcher }, syncer)

	task := sampleTask()
	task.IsDefaultBranch = false

	err := s.ProcessTask(context.Background(), task)
	if err != nil {
		t.Fatalf("expected nil error on non-default branch, got %v", err)
	}
}

func TestProcessTaskTarballDownloadError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("tarball download failed")
	fetcher := &mockTarballFetcher{archiveData: nil, err: expectedErr}
	syncer := &mockSpannerSyncer{
		syncedSubs: nil,
		syncErr:    nil,
		scanLogs:   nil,
		logErr:     nil,
	}
	s := NewScanner(nil, func(_ string) TarballFetcher { return fetcher }, syncer)

	err := s.ProcessTask(context.Background(), sampleTask())
	if err == nil {
		t.Fatalf("expected tarball download error, got nil")
	}
	if !errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("expected ErrTransientFailure, got %v", err)
	}

	if len(syncer.scanLogs) != 1 {
		t.Fatalf("expected 1 scan log recorded for failed scan, got %d", len(syncer.scanLogs))
	}
	if syncer.scanLogs[0].ScanStatus != gcpspanner.ScanStatusFailed {
		t.Errorf("scan log status = %v, want FAILED", syncer.scanLogs[0].ScanStatus)
	}
}

func TestProcessTaskTokenProviderError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("token provider expired")
	tp := &mockTokenProvider{token: "", err: expectedErr}
	fetcher := &mockTarballFetcher{archiveData: nil, err: nil}
	syncer := &mockSpannerSyncer{
		syncedSubs: nil,
		syncErr:    nil,
		scanLogs:   nil,
		logErr:     nil,
	}
	s := NewScanner(tp, func(_ string) TarballFetcher { return fetcher }, syncer)

	err := s.ProcessTask(context.Background(), sampleTask())
	if err == nil {
		t.Fatalf("expected token provider error, got nil")
	}
	if !errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("expected ErrTransientFailure, got %v", err)
	}

	if len(syncer.scanLogs) != 1 {
		t.Fatalf("expected 1 scan log recorded for failed scan, got %d", len(syncer.scanLogs))
	}
	if syncer.scanLogs[0].ScanStatus != gcpspanner.ScanStatusFailed {
		t.Errorf("scan log status = %v, want FAILED", syncer.scanLogs[0].ScanStatus)
	}
}

//nolint:exhaustruct
func TestProcessTaskTarballDownloadClientErrorNotRetried(t *testing.T) {
	t.Parallel()

	clientErr := &github.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
		},
		Message: "Not Found",
	}
	fetcher := &mockTarballFetcher{archiveData: nil, err: clientErr}
	syncer := &mockSpannerSyncer{
		syncedSubs: nil,
		syncErr:    nil,
		scanLogs:   nil,
		logErr:     nil,
	}
	s := NewScanner(nil, func(_ string) TarballFetcher { return fetcher }, syncer)

	err := s.ProcessTask(context.Background(), sampleTask())
	if err == nil {
		t.Fatalf("expected tarball download error, got nil")
	}
	if errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("expected non-transient error for 404 client error, but got ErrTransientFailure")
	}

	if len(syncer.scanLogs) != 1 {
		t.Fatalf("expected 1 scan log recorded for failed scan, got %d", len(syncer.scanLogs))
	}
	if syncer.scanLogs[0].ScanStatus != gcpspanner.ScanStatusFailed {
		t.Errorf("scan log status = %v, want FAILED", syncer.scanLogs[0].ScanStatus)
	}
}

//nolint:exhaustruct
func TestProcessTaskTarballDownloadServerErrorRetried(t *testing.T) {
	t.Parallel()

	serverErr := &github.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusServiceUnavailable,
		},
		Message: "Service Unavailable",
	}
	fetcher := &mockTarballFetcher{archiveData: nil, err: serverErr}
	syncer := &mockSpannerSyncer{
		syncedSubs: nil,
		syncErr:    nil,
		scanLogs:   nil,
		logErr:     nil,
	}
	s := NewScanner(nil, func(_ string) TarballFetcher { return fetcher }, syncer)

	err := s.ProcessTask(context.Background(), sampleTask())
	if err == nil {
		t.Fatalf("expected tarball download error, got nil")
	}
	if !errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("expected ErrTransientFailure for 503 server error, got %v", err)
	}

	if len(syncer.scanLogs) != 1 {
		t.Fatalf("expected 1 scan log recorded for failed scan, got %d", len(syncer.scanLogs))
	}
	if syncer.scanLogs[0].ScanStatus != gcpspanner.ScanStatusFailed {
		t.Errorf("scan log status = %v, want FAILED", syncer.scanLogs[0].ScanStatus)
	}
}

//nolint:exhaustruct
func TestProcessTaskTokenProviderClientErrorNotRetried(t *testing.T) {
	t.Parallel()

	clientErr := &github.ErrorResponse{
		Response: &http.Response{
			StatusCode: http.StatusUnauthorized,
		},
		Message: "Bad credentials",
	}
	tp := &mockTokenProvider{token: "", err: clientErr}
	fetcher := &mockTarballFetcher{archiveData: nil, err: nil}
	syncer := &mockSpannerSyncer{
		syncedSubs: nil,
		syncErr:    nil,
		scanLogs:   nil,
		logErr:     nil,
	}
	s := NewScanner(tp, func(_ string) TarballFetcher { return fetcher }, syncer)

	err := s.ProcessTask(context.Background(), sampleTask())
	if err == nil {
		t.Fatalf("expected token provider error, got nil")
	}
	if errors.Is(err, event.ErrTransientFailure) {
		t.Errorf("expected non-transient error for 401 client error, but got ErrTransientFailure")
	}

	if len(syncer.scanLogs) != 1 {
		t.Fatalf("expected 1 scan log recorded for failed scan, got %d", len(syncer.scanLogs))
	}
	if syncer.scanLogs[0].ScanStatus != gcpspanner.ScanStatusFailed {
		t.Errorf("scan log status = %v, want FAILED", syncer.scanLogs[0].ScanStatus)
	}
}
