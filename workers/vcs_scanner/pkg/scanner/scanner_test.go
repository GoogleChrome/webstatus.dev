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
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
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
	mu    sync.Mutex
	syncs []gcpspanner.RepositoryScanSync
	// syncDeadline is the deadline of the context passed to SyncRepositoryScan.
	syncDeadline    time.Time
	syncHasDeadline bool
	syncErr         error
	applied         bool
	panicMsg        string
	scanLogs        []gcpspanner.CodeSubscriptionScanLog
	logErr          error
}

func newMockSpannerSyncer() *mockSpannerSyncer {
	return &mockSpannerSyncer{
		mu:              sync.Mutex{},
		syncs:           nil,
		syncDeadline:    time.Time{},
		syncHasDeadline: false,
		syncErr:         nil,
		applied:         true,
		panicMsg:        "",
		scanLogs:        nil,
		logErr:          nil,
	}
}

func (m *mockSpannerSyncer) SyncRepositoryScan(
	ctx context.Context,
	in gcpspanner.RepositoryScanSync,
) (bool, error) {
	if m.panicMsg != "" {
		panic(m.panicMsg)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.syncs = append(m.syncs, in)
	m.syncDeadline, m.syncHasDeadline = ctx.Deadline()

	return m.applied, m.syncErr
}

func (m *mockSpannerSyncer) InsertCodeSubscriptionScanLog(
	_ context.Context,
	log gcpspanner.CodeSubscriptionScanLog,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanLogs = append(m.scanLogs, log)

	return m.logErr
}

func createTarGzArchive(files map[string][]byte) []byte {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	// Deterministic order so truncation tests know which files come first.
	sort.Strings(names)

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for _, name := range names {
		content := files[name]
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

func newTestScanner(fetcher TarballFetcher, tp TokenProvider, syncer SpannerSyncer) *Scanner {
	return NewScanner(tp, func(_ string) TarballFetcher { return fetcher }, syncer)
}

func requireSingleFailedLog(t *testing.T, syncer *mockSpannerSyncer, wantClass gcpspanner.ScanErrorClass) {
	t.Helper()

	if len(syncer.scanLogs) != 1 {
		t.Fatalf("expected 1 FAILED scan log, got %d", len(syncer.scanLogs))
	}
	got := syncer.scanLogs[0]
	if got.ScanStatus != gcpspanner.ScanStatusFailed {
		t.Errorf("scan log status = %v, want FAILED", got.ScanStatus)
	}
	if got.ErrorClass == nil || *got.ErrorClass != wantClass {
		t.Errorf("scan log error class = %v, want %v", got.ErrorClass, wantClass)
	}
}

func requireNotTransient(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if errors.Is(err, event.ErrTransientFailure) {
		t.Fatalf("scan failures must be acknowledged, not retried by Pub/Sub; got transient error %v", err)
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
	syncer := newMockSpannerSyncer()

	s := newTestScanner(fetcher, tp, syncer)
	if err := s.ProcessTask(context.Background(), sampleTask()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(syncer.syncs) != 1 {
		t.Fatalf("expected 1 sync, got %d", len(syncer.syncs))
	}
	sync := syncer.syncs[0]
	if len(sync.Subscriptions) != 2 {
		t.Fatalf("expected 2 subscriptions synced, got %d", len(sync.Subscriptions))
	}
	if sync.Obsolescence.Truncated {
		t.Errorf("expected a complete scan")
	}
	if _, ok := sync.Obsolescence.ParsedPaths["src/index.ts"]; !ok {
		t.Errorf("expected src/index.ts in parsed paths, got %v", sync.Obsolescence.ParsedPaths)
	}

	log := sync.ScanLog
	if log.ScanStatus != gcpspanner.ScanStatusSuccess {
		t.Errorf("log.ScanStatus = %v, want SUCCESS", log.ScanStatus)
	}
	if log.FilesScanned != 2 {
		t.Errorf("log.FilesScanned = %d, want 2", log.FilesScanned)
	}
	if log.DirectivesFound != 2 {
		t.Errorf("log.DirectivesFound = %d, want 2", log.DirectivesFound)
	}
	if log.ID == "" || log.ScannedAt.IsZero() {
		t.Errorf("scan log must carry an ID and start time for the fence, got %+v", log)
	}
	if len(syncer.scanLogs) != 0 {
		t.Errorf("a successful scan must not write a separate scan log row, got %d", len(syncer.scanLogs))
	}
}

func TestProcessTaskSyncContextIsDetachedFromScanDeadline(t *testing.T) {
	t.Parallel()

	archiveBytes := createTarGzArchive(map[string][]byte{"src/a.ts": []byte("// TODO(baseline/popover): x\n")})
	syncer := newMockSpannerSyncer()
	s := newTestScanner(&mockTarballFetcher{archiveData: archiveBytes, err: nil}, nil, syncer)

	ctx := t.Context()
	if err := s.ProcessTask(ctx, sampleTask()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	deadline, ok := syncer.syncDeadline, syncer.syncHasDeadline
	if !ok {
		t.Fatal("expected the sync context to have its own deadline")
	}
	if remaining := time.Until(deadline); remaining > DefaultSyncTimeout || remaining <= 0 {
		t.Errorf("sync deadline in %v, want within the %v sync budget", remaining, DefaultSyncTimeout)
	}
}

func TestProcessTaskSupersededScanIsNotAnError(t *testing.T) {
	t.Parallel()

	archiveBytes := createTarGzArchive(map[string][]byte{"src/a.ts": []byte("// TODO(baseline/popover): x\n")})
	syncer := newMockSpannerSyncer()
	syncer.applied = false
	s := newTestScanner(&mockTarballFetcher{archiveData: archiveBytes, err: nil}, nil, syncer)

	if err := s.ProcessTask(context.Background(), sampleTask()); err != nil {
		t.Fatalf("a superseded scan should be acknowledged without error, got %v", err)
	}
	if len(syncer.scanLogs) != 0 {
		t.Fatalf("a superseded scan must not write a FAILED log, got %d", len(syncer.scanLogs))
	}
}

func TestProcessTaskDeletedBranch(t *testing.T) {
	t.Parallel()

	syncer := newMockSpannerSyncer()
	s := newTestScanner(&mockTarballFetcher{archiveData: nil, err: nil}, nil, syncer)

	task := sampleTask()
	task.CommitSHA = DeletedBranchSHA

	if err := s.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("expected nil error on deleted branch, got %v", err)
	}
}

func TestProcessTaskNonDefaultBranch(t *testing.T) {
	t.Parallel()

	syncer := newMockSpannerSyncer()
	s := newTestScanner(&mockTarballFetcher{archiveData: nil, err: nil}, nil, syncer)

	task := sampleTask()
	task.IsDefaultBranch = false

	if err := s.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("expected nil error on non-default branch, got %v", err)
	}
}

// Failures are recorded and acknowledged, never retried through Pub/Sub; vcs-sync republishes
// repositories whose latest scan failed.
func TestProcessTaskFailuresAreRecordedAndAcknowledged(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		fetcherErr error
		tokenErr   error
		syncErr    error
		wantClass  gcpspanner.ScanErrorClass
	}{
		{
			name:       "tarball download network error",
			fetcherErr: errors.New("tarball download failed"),
			tokenErr:   nil,
			syncErr:    nil,
			wantClass:  gcpspanner.ScanErrorClassUnknown,
		},
		{
			name:       "token provider error",
			fetcherErr: nil,
			tokenErr:   errors.New("token provider expired"),
			syncErr:    nil,
			wantClass:  gcpspanner.ScanErrorClassUnknown,
		},
		{
			name:       "tarball 404",
			fetcherErr: gh.NewHTTPErrorResponseForTest(http.StatusNotFound, "Not Found"),
			tokenErr:   nil,
			syncErr:    nil,
			wantClass:  gcpspanner.ScanErrorClassPermanentClient,
		},
		{
			name:       "tarball 503",
			fetcherErr: gh.NewHTTPErrorResponseForTest(http.StatusServiceUnavailable, "Service Unavailable"),
			tokenErr:   nil,
			syncErr:    nil,
			wantClass:  gcpspanner.ScanErrorClassTransientServer,
		},
		{
			name:       "token 401",
			fetcherErr: nil,
			tokenErr:   gh.NewHTTPErrorResponseForTest(http.StatusUnauthorized, "Bad credentials"),
			syncErr:    nil,
			wantClass:  gcpspanner.ScanErrorClassPermanentClient,
		},
		{
			name:       "tarball request deadline",
			fetcherErr: fmt.Errorf("failed to download tarball stream: %w", context.DeadlineExceeded),
			tokenErr:   nil,
			syncErr:    nil,
			wantClass:  gcpspanner.ScanErrorClassTimeout,
		},
		{
			name:       "spanner sync error",
			fetcherErr: nil,
			tokenErr:   nil,
			syncErr:    errors.New("transaction aborted"),
			wantClass:  gcpspanner.ScanErrorClassSpanner,
		},
	}

	archiveBytes := createTarGzArchive(map[string][]byte{"src/a.ts": []byte("// TODO(baseline/popover): x\n")})
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			syncer := newMockSpannerSyncer()
			syncer.syncErr = tc.syncErr
			tp := &mockTokenProvider{token: "token", err: tc.tokenErr}
			s := newTestScanner(&mockTarballFetcher{archiveData: archiveBytes, err: tc.fetcherErr}, tp, syncer)

			err := s.ProcessTask(context.Background(), sampleTask())
			requireNotTransient(t, err)
			requireSingleFailedLog(t, syncer, tc.wantClass)
		})
	}
}

func TestProcessTaskRecoversFromPanic(t *testing.T) {
	t.Parallel()

	archiveBytes := createTarGzArchive(map[string][]byte{"src/a.ts": []byte("// TODO(baseline/popover): x\n")})
	syncer := newMockSpannerSyncer()
	syncer.panicMsg = "boom"
	s := newTestScanner(&mockTarballFetcher{archiveData: archiveBytes, err: nil}, nil, syncer)

	err := s.ProcessTask(context.Background(), sampleTask())
	requireNotTransient(t, err)
	if !errors.Is(err, errScanPanicked) {
		t.Fatalf("expected errScanPanicked, got %v", err)
	}
	requireSingleFailedLog(t, syncer, gcpspanner.ScanErrorClassPanic)
}

func TestProcessTaskTruncationAndObsolescenceScope(t *testing.T) {
	t.Parallel()

	files := map[string][]byte{
		"src/a.ts": []byte("// TODO(baseline/popover): a\n"),
		"src/b.ts": []byte("// TODO(baseline/dialog): b\n"),
		"src/c.ts": []byte("// TODO(baseline/subgrid): c\n"),
	}
	archiveBytes := createTarGzArchive(files)

	t.Run("file count limit", func(t *testing.T) {
		t.Parallel()

		syncer := newMockSpannerSyncer()
		s := newTestScanner(&mockTarballFetcher{archiveData: archiveBytes, err: nil}, nil, syncer)
		s.maxScannedFiles = 2

		if err := s.ProcessTask(context.Background(), sampleTask()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		sync := syncer.syncs[0]
		if sync.ScanLog.ScanStatus != gcpspanner.ScanStatusTruncated {
			t.Errorf("status = %v, want TRUNCATED", sync.ScanLog.ScanStatus)
		}
		if !sync.Obsolescence.Truncated {
			t.Errorf("obsolescence scope must be truncated")
		}
		if _, ok := sync.Obsolescence.ParsedPaths["src/c.ts"]; ok {
			t.Errorf("src/c.ts was past the cutoff and must not be in parsed paths")
		}
		if len(sync.Obsolescence.ParsedPaths) != 2 {
			t.Errorf("parsed paths = %v, want 2 files", sync.Obsolescence.ParsedPaths)
		}
	})

	t.Run("archive size cap", func(t *testing.T) {
		t.Parallel()

		syncer := newMockSpannerSyncer()
		s := newTestScanner(&mockTarballFetcher{archiveData: archiveBytes, err: nil}, nil, syncer)
		// Enough for the first entry (512-byte header + padded content) but not the whole archive.
		s.maxArchiveBytes = 1024

		if err := s.ProcessTask(context.Background(), sampleTask()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		sync := syncer.syncs[0]
		if sync.ScanLog.ScanStatus != gcpspanner.ScanStatusTruncated {
			t.Errorf("status = %v, want TRUNCATED", sync.ScanLog.ScanStatus)
		}
		if !sync.Obsolescence.Truncated {
			t.Errorf("obsolescence scope must be truncated")
		}
		if sync.ScanLog.ErrorMessage == nil || !strings.Contains(*sync.ScanLog.ErrorMessage, string(truncationArchiveSize)) {
			t.Errorf("expected the truncation reason in the scan log, got %v", sync.ScanLog.ErrorMessage)
		}
	})
}

func TestProcessTaskOversizeFileIsUnscanned(t *testing.T) {
	t.Parallel()

	big := bytes.Repeat([]byte("a"), int(MaxBlobSizeBytes)+1)
	archiveBytes := createTarGzArchive(map[string][]byte{
		"src/big.js": big,
		"src/a.ts":   []byte("// TODO(baseline/popover): a\n"),
	})
	syncer := newMockSpannerSyncer()
	s := newTestScanner(&mockTarballFetcher{archiveData: archiveBytes, err: nil}, nil, syncer)

	if err := s.ProcessTask(context.Background(), sampleTask()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sync := syncer.syncs[0]
	if _, ok := sync.Obsolescence.UnscannedPaths["src/big.js"]; !ok {
		t.Errorf("expected src/big.js in unscanned paths, got %v", sync.Obsolescence.UnscannedPaths)
	}
	if sync.ScanLog.ScanStatus != gcpspanner.ScanStatusSuccess {
		t.Errorf("status = %v, want SUCCESS", sync.ScanLog.ScanStatus)
	}
}

func TestBuildSubscriptionsCapIsSortedAndDeterministic(t *testing.T) {
	t.Parallel()

	occurrences := make(map[string][]gcpspanner.SubscriptionOccurrence)
	triggers := make(map[string]map[gcpspanner.SubscriptionTrigger]struct{})
	for i := range MaxSubscriptionsPerRepo + 5 {
		key := fmt.Sprintf("feature-%04d", i)
		occurrences[key] = []gcpspanner.SubscriptionOccurrence{{FilePath: "src/a.ts", LineNumber: 1, CommentSnippet: key}}
		triggers[key] = make(map[gcpspanner.SubscriptionTrigger]struct{})
		triggers[key][gcpspanner.SubscriptionTriggerFeatureBaselinePromoteToWidely] = struct{}{}
	}

	s := newTestScanner(&mockTarballFetcher{archiveData: nil, err: nil}, nil, newMockSpannerSyncer())
	subs, found, capped := s.buildSubscriptions(sampleTask(), gcpspanner.VCSProviderGitHub, occurrences, triggers)

	if !capped || len(subs) != MaxSubscriptionsPerRepo {
		t.Fatalf("expected %d capped subscriptions, got %d (capped=%v)", MaxSubscriptionsPerRepo, len(subs), capped)
	}
	if len(found) != MaxSubscriptionsPerRepo+5 {
		t.Fatalf("found = %d keys, want every key (%d)", len(found), MaxSubscriptionsPerRepo+5)
	}
	if subs[0].TargetQuery != "feature-0000" || subs[len(subs)-1].TargetQuery != "feature-0499" {
		t.Fatalf("expected the first %d keys in sorted order, got %s..%s",
			MaxSubscriptionsPerRepo, subs[0].TargetQuery, subs[len(subs)-1].TargetQuery)
	}
}

type netTimeoutError struct{}

func (netTimeoutError) Error() string   { return "i/o timeout" }
func (netTimeoutError) Timeout() bool   { return true }
func (netTimeoutError) Temporary() bool { return true }

func TestClassifyScanError(t *testing.T) {
	t.Parallel()

	const (
		liveCtx     = false
		canceledCtx = true
	)

	testCases := []struct {
		name   string
		parent bool // true: the handler's parent context is canceled (worker shutting down)
		err    error
		want   gcpspanner.ScanErrorClass
	}{
		{name: "watchdog trip", parent: liveCtx,
			err: fmt.Errorf("%w: %w", errStall, context.Canceled), want: gcpspanner.ScanErrorClassStall},
		{name: "scan deadline", parent: liveCtx,
			err: fmt.Errorf("read: %w", context.DeadlineExceeded), want: gcpspanner.ScanErrorClassTimeout},
		{name: "network timeout", parent: liveCtx,
			err: fmt.Errorf("dial: %w", netTimeoutError{}), want: gcpspanner.ScanErrorClassTimeout},
		{name: "canceled with live parent", parent: liveCtx,
			err: context.Canceled, want: gcpspanner.ScanErrorClassStream},
		{name: "canceled during shutdown", parent: canceledCtx,
			err: context.Canceled, want: gcpspanner.ScanErrorClassCanceled},
		{name: "404", parent: liveCtx,
			err: gh.NewHTTPErrorResponseForTest(http.StatusNotFound, "x"), want: gcpspanner.ScanErrorClassPermanentClient},
		{name: "504", parent: liveCtx,
			err: gh.NewHTTPErrorResponseForTest(http.StatusGatewayTimeout, "x"), want: gcpspanner.ScanErrorClassTransientServer},
		{name: "429", parent: liveCtx,
			err: gh.NewHTTPErrorResponseForTest(http.StatusTooManyRequests, "x"), want: gcpspanner.ScanErrorClassRateLimited},
		{name: "redirect not followed", parent: liveCtx,
			err: fmt.Errorf("link: %w", gh.ErrRedirectNotFollowed), want: gcpspanner.ScanErrorClassRedirectNotFollowed},
		{name: "invalid provider", parent: liveCtx,
			err: fmt.Errorf("x: %w", gcpspanner.ErrUnknownVCSProvider), want: gcpspanner.ScanErrorClassInvalidProvider},
		{name: "spanner", parent: liveCtx,
			err: fmt.Errorf("%w: %w", errSyncFailed, context.DeadlineExceeded), want: gcpspanner.ScanErrorClassSpanner},
		{name: "panic", parent: liveCtx,
			err: fmt.Errorf("%w: boom", errScanPanicked), want: gcpspanner.ScanErrorClassPanic},
		{name: "bad gzip", parent: liveCtx, err: gzip.ErrHeader, want: gcpspanner.ScanErrorClassArchiveCorrupt},
		{name: "unexpected EOF", parent: liveCtx,
			err: fmt.Errorf("read tar entry: %w", io.ErrUnexpectedEOF), want: gcpspanner.ScanErrorClassStream},
		{name: "connection reset", parent: liveCtx,
			err: fmt.Errorf("read: %w", syscall.ECONNRESET), want: gcpspanner.ScanErrorClassStream},
		{name: "h2 stream reset", parent: liveCtx,
			err: errors.New("stream error: stream ID 9; CANCEL; received from peer"), want: gcpspanner.ScanErrorClassStream},
		{name: "unknown", parent: liveCtx, err: errors.New("something else"), want: gcpspanner.ScanErrorClassUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			parent, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.parent {
				cancel()
			}
			if got := classifyScanError(parent, tc.err); got != tc.want {
				t.Errorf("classifyScanError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// dripReader returns one byte per Read after a delay, simulating a slow-drip connection.
type dripReader struct{ delay time.Duration }

func (d dripReader) Read(p []byte) (int, error) {
	time.Sleep(d.delay)
	if len(p) == 0 {
		return 0, nil
	}
	p[0] = 'x'

	return 1, nil
}

// blockingReader blocks until done is closed, simulating a fully stalled connection.
type blockingReader struct{ done <-chan struct{} }

func (b blockingReader) Read(_ []byte) (int, error) {
	<-b.done

	return 0, context.Canceled
}

func TestThroughputWatchdog(t *testing.T) {
	t.Parallel()

	t.Run("slow drip trips the floor", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		w := newThroughputWatchdog(dripReader{delay: 5 * time.Millisecond}, cancel, 100*time.Millisecond, 1024)

		_, err := io.Copy(io.Discard, w)
		if !errors.Is(err, errStall) {
			t.Fatalf("expected errStall, got %v", err)
		}
		if !errors.Is(context.Cause(ctx), errStall) {
			t.Fatalf("expected the request context to be canceled with errStall, got %v", context.Cause(ctx))
		}
		if _, err := w.Read(make([]byte, 1)); !errors.Is(err, errStall) {
			t.Fatalf("reads after tripping must return errStall, got %v", err)
		}
	})

	t.Run("fully stalled read trips via timer", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		w := newThroughputWatchdog(blockingReader{done: ctx.Done()}, cancel, 50*time.Millisecond, 1024)

		done := make(chan error, 1)
		go func() {
			_, err := w.Read(make([]byte, 32))
			done <- err
		}()
		select {
		case err := <-done:
			if !errors.Is(err, errStall) {
				t.Fatalf("expected errStall, got %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("watchdog did not abort a stalled read")
		}
	})

	t.Run("slow consumer does not trip", func(t *testing.T) {
		t.Parallel()

		_, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		data := bytes.Repeat([]byte("y"), 64)
		w := newThroughputWatchdog(bytes.NewReader(data), cancel, 20*time.Millisecond, 1024)

		buf := make([]byte, 8)
		for {
			// Time spent between reads (slow parsing) must not count against the download.
			time.Sleep(10 * time.Millisecond)
			_, err := w.Read(buf)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("unexpected error from a slow consumer: %v", err)
			}
		}
	})
}

func TestArchiveCapReader(t *testing.T) {
	t.Parallel()

	t.Run("exactly at the cap is not truncated", func(t *testing.T) {
		t.Parallel()

		r := &archiveCapReader{r: bytes.NewReader([]byte("abcd")), remaining: 4, read: 0}
		got, err := io.ReadAll(r)
		if err != nil || string(got) != "abcd" {
			t.Fatalf("ReadAll = %q, %v; want abcd, nil", got, err)
		}
	})

	t.Run("beyond the cap returns the sentinel, not EOF", func(t *testing.T) {
		t.Parallel()

		r := &archiveCapReader{r: bytes.NewReader([]byte("abcdef")), remaining: 4, read: 0}
		_, err := io.ReadAll(r)
		if !errors.Is(err, errArchiveCapExceeded) {
			t.Fatalf("expected errArchiveCapExceeded, got %v", err)
		}
	})
}
