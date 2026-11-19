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
	"compress/flate"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http/httptrace"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/codescan"
	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/google/uuid"
)

const (
	MaxBlobSizeBytes  int64 = 1024 * 1024       // 1MB per file
	MaxScanTotalBytes int64 = 100 * 1024 * 1024 // 100MB aggregate scan limit
	MaxScannedFiles   int64 = 10_000            // Maximum files to scan per tarball
	// MaxArchiveBytes caps the decompressed size of the whole archive, including files that are
	// not scanned (binaries, images). Larger archives are scanned up to the cap and marked TRUNCATED.
	MaxArchiveBytes         int64 = 500 * 1024 * 1024
	MaxSubscriptionsPerRepo int   = 500
	DeletedBranchSHA              = "0000000000000000000000000000000000000000"

	// DefaultScanPhaseTimeout bounds the installation token, archive link, download, and parsing.
	DefaultScanPhaseTimeout = 3*time.Minute + 30*time.Second
	// DefaultSyncTimeout bounds the Spanner transaction that applies the results. It is detached
	// from the scan-phase deadline so a scan that finishes close to its limit can still commit.
	DefaultSyncTimeout = 30 * time.Second
	// DefaultFailedLogTimeout bounds writing the FAILED scan log row.
	DefaultFailedLogTimeout = 5 * time.Second
	// DefaultStallWindow and DefaultMinBytesPerWindow define the download throughput floor: a
	// download that receives fewer than DefaultMinBytesPerWindow bytes during DefaultStallWindow
	// of time spent waiting on the network is aborted.
	DefaultStallWindow             = 30 * time.Second
	DefaultMinBytesPerWindow int64 = 64 * 1024
)

var (
	// errStall is returned when the throughput watchdog aborts a download.
	errStall = errors.New("archive download stalled: throughput fell below the minimum")
	// errArchiveCapExceeded is returned by the archive size cap; it ends the scan as TRUNCATED.
	errArchiveCapExceeded = errors.New("archive exceeds the maximum decompressed size")
	// errSyncFailed wraps failures while applying scan results in Spanner.
	errSyncFailed = errors.New("failed to sync code subscriptions")
	// errScanPanicked is returned when ProcessTask recovers from a panic.
	errScanPanicked = errors.New("panic while scanning repository")
)

// truncationReason records which deterministic limit ended a scan early. Rescanning the same
// content would hit the same limit, so a truncated scan is treated as complete for scheduling
// purposes, but it must not mark subscriptions in unscanned files as OBSOLETE.
type truncationReason string

const (
	truncationNone          truncationReason = ""
	truncationFileCount     truncationReason = "file_count"
	truncationScanBytes     truncationReason = "scan_bytes"
	truncationArchiveSize   truncationReason = "archive_size"
	truncationSubscriptions truncationReason = "subscriptions"
)

type TarballFetcher interface {
	DownloadTarball(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error)
}

type TokenProvider interface {
	GetInstallationToken(ctx context.Context, installationID string) (string, error)
}

type ClientFactory func(token string) TarballFetcher

type SpannerSyncer interface {
	// SyncRepositoryScan applies the scan results and writes the scan log row atomically.
	// It returns false, without writing anything, when a newer scan has already been applied.
	SyncRepositoryScan(ctx context.Context, in gcpspanner.RepositoryScanSync) (bool, error)
	InsertCodeSubscriptionScanLog(ctx context.Context, scanLog gcpspanner.CodeSubscriptionScanLog) error
}

type Scanner struct {
	tokenProvider TokenProvider
	clientFactory ClientFactory
	spannerSyncer SpannerSyncer

	scanPhaseTimeout  time.Duration
	syncTimeout       time.Duration
	failedLogTimeout  time.Duration
	stallWindow       time.Duration
	minBytesPerWindow int64
	maxArchiveBytes   int64
	maxScannedFiles   int64
	now               func() time.Time
}

func NewScanner(
	tokenProvider TokenProvider,
	clientFactory ClientFactory,
	spannerSyncer SpannerSyncer,
) *Scanner {
	if clientFactory == nil {
		// One archive download client (and transport) for the whole process; it opens a fresh
		// HTTP/1.1 connection per download. See gh.NewArchiveDownloadHTTPClient.
		archiveClient := gh.NewArchiveDownloadHTTPClient()
		clientFactory = func(token string) TarballFetcher {
			return gh.NewClient(token).WithArchiveHTTPClient(archiveClient)
		}
	}

	return &Scanner{
		tokenProvider:     tokenProvider,
		clientFactory:     clientFactory,
		spannerSyncer:     spannerSyncer,
		scanPhaseTimeout:  DefaultScanPhaseTimeout,
		syncTimeout:       DefaultSyncTimeout,
		failedLogTimeout:  DefaultFailedLogTimeout,
		stallWindow:       DefaultStallWindow,
		minBytesPerWindow: DefaultMinBytesPerWindow,
		maxArchiveBytes:   MaxArchiveBytes,
		maxScannedFiles:   MaxScannedFiles,
		now:               time.Now,
	}
}

func splitOwnerRepo(fullName string) (string, string) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return "", fullName
}

func stripTarRootPrefix(path string) string {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return ""
	}

	return parts[1]
}

func mapVCSProvider(provider codescantaskv1.VCSProvider) (gcpspanner.VCSProvider, error) {
	switch provider {
	case codescantaskv1.VCSProviderGitHub:
		return gcpspanner.VCSProviderGitHub, nil
	default:
		return "", fmt.Errorf("%w: unsupported vcs provider '%s'", gcpspanner.ErrUnknownVCSProvider, provider)
	}
}

// classifyScanError maps a scan failure to a ScanErrorClass. Order matters: a watchdog abort
// surfaces as a canceled request, so errStall is checked before context errors, and
// context.Canceled only means "shutting down" when the handler's parent context is done.
func classifyScanError(parentCtx context.Context, err error) gcpspanner.ScanErrorClass {
	var netErr net.Error
	switch {
	case errors.Is(err, errScanPanicked):
		return gcpspanner.ScanErrorClassPanic
	case errors.Is(err, errStall):
		return gcpspanner.ScanErrorClassStall
	case errors.Is(err, errSyncFailed):
		return gcpspanner.ScanErrorClassSpanner
	case errors.Is(err, gcpspanner.ErrUnknownVCSProvider):
		return gcpspanner.ScanErrorClassInvalidProvider
	case gh.IsRateLimitError(err):
		return gcpspanner.ScanErrorClassRateLimited
	case errors.Is(err, gh.ErrRedirectNotFollowed):
		return gcpspanner.ScanErrorClassRedirectNotFollowed
	case gh.IsClientError(err):
		return gcpspanner.ScanErrorClassPermanentClient
	case gh.IsServerError(err):
		return gcpspanner.ScanErrorClassTransientServer
	case errors.Is(err, context.DeadlineExceeded),
		errors.As(err, &netErr) && netErr.Timeout():
		return gcpspanner.ScanErrorClassTimeout
	case errors.Is(err, context.Canceled):
		if parentCtx.Err() != nil {
			return gcpspanner.ScanErrorClassCanceled
		}

		return gcpspanner.ScanErrorClassStream
	case isArchiveFormatError(err):
		return gcpspanner.ScanErrorClassArchiveCorrupt
	case isStreamError(err):
		return gcpspanner.ScanErrorClassStream
	}

	return gcpspanner.ScanErrorClassUnknown
}

func isArchiveFormatError(err error) bool {
	var corrupt flate.CorruptInputError

	return errors.Is(err, gzip.ErrHeader) ||
		errors.Is(err, gzip.ErrChecksum) ||
		errors.Is(err, tar.ErrHeader) ||
		errors.As(err, &corrupt)
}

func isStreamError(err error) bool {
	var opErr *net.OpError
	if errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, net.ErrClosed) ||
		errors.As(err, &opErr) {
		return true
	}

	// HTTP/2 stream resets (e.g. "stream error: stream ID 9; CANCEL") use an unexported type.
	return strings.Contains(err.Error(), "stream error")
}

// throughputWatchdog aborts a download whose throughput drops below minBytes per window of time
// spent blocked reading from the network. Only time blocked inside Read counts, so slow parsing
// downstream never trips it. It catches both a fully stalled stream and a slow drip that a plain
// "no bytes received" timer would miss. On trip it cancels the request context with errStall,
// which closes the connection, and every later Read returns an error wrapping errStall.
type throughputWatchdog struct {
	r        io.Reader
	cancel   context.CancelCauseFunc
	window   time.Duration
	minBytes int64

	mu          sync.Mutex
	windowWait  time.Duration
	windowBytes int64
	totalBytes  int64
	tripped     bool
}

func newThroughputWatchdog(
	r io.Reader,
	cancel context.CancelCauseFunc,
	window time.Duration,
	minBytes int64,
) *throughputWatchdog {
	return &throughputWatchdog{
		r:           r,
		cancel:      cancel,
		window:      window,
		minBytes:    minBytes,
		mu:          sync.Mutex{},
		windowWait:  0,
		windowBytes: 0,
		totalBytes:  0,
		tripped:     false,
	}
}

func (w *throughputWatchdog) Read(p []byte) (int, error) {
	w.mu.Lock()
	if w.tripped {
		w.mu.Unlock()

		return 0, errStall
	}
	remaining := w.window - w.windowWait
	needMore := w.windowBytes < w.minBytes
	w.mu.Unlock()

	// A single Read that stays blocked until the window ends cannot have received enough bytes.
	var timer *time.Timer
	if needMore {
		timer = time.AfterFunc(remaining, w.trip)
	}
	start := time.Now()
	n, err := w.r.Read(p)
	waited := time.Since(start)
	if timer != nil {
		timer.Stop()
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.totalBytes += int64(n)
	if !w.tripped {
		w.windowWait += waited
		w.windowBytes += int64(n)
		if w.windowWait >= w.window {
			if w.windowBytes < w.minBytes {
				w.tripLocked()
			} else {
				w.windowWait, w.windowBytes = 0, 0
			}
		}
	}

	if w.tripped {
		if err != nil {
			return n, fmt.Errorf("%w: %w", errStall, err)
		}

		return n, errStall
	}

	return n, err
}

func (w *throughputWatchdog) trip() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.tripLocked()
}

func (w *throughputWatchdog) tripLocked() {
	if !w.tripped {
		w.tripped = true
		w.cancel(errStall)
	}
}

func (w *throughputWatchdog) bytesRead() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.totalBytes
}

// archiveCapReader caps the decompressed archive size. Unlike io.LimitReader it returns
// errArchiveCapExceeded instead of io.EOF, so a cut at a tar header boundary can't be mistaken
// for the end of the archive.
type archiveCapReader struct {
	r         io.Reader
	remaining int64
	read      int64
}

func (c *archiveCapReader) Read(p []byte) (int, error) {
	if c.remaining <= 0 {
		var probe [1]byte
		if n, err := c.r.Read(probe[:]); n == 0 && errors.Is(err, io.EOF) {
			return 0, io.EOF
		}

		return 0, errArchiveCapExceeded
	}
	if int64(len(p)) > c.remaining {
		p = p[:c.remaining]
	}
	n, err := c.r.Read(p)
	c.remaining -= int64(n)
	c.read += int64(n)

	return n, err
}

type tarballScan struct {
	occurrencesByQuery map[string][]gcpspanner.SubscriptionOccurrence
	triggersByQuery    map[string]map[gcpspanner.SubscriptionTrigger]struct{}
	filesScanned       int64
	// scanBytes is the total size of the web files read so far (MaxScanTotalBytes budget).
	scanBytes int64
	// parsedPaths are files read and parsed completely.
	parsedPaths map[string]struct{}
	// unscannedPaths are web files skipped for size or that failed to parse.
	unscannedPaths   map[string]struct{}
	truncated        truncationReason
	decompressedSize int64
}

func (s *Scanner) recordFailedScanLog(
	ctx context.Context,
	task codescantaskv1.CodeScanTaskEvent,
	scannedAt time.Time,
	class gcpspanner.ScanErrorClass,
	errMsg string,
) {
	vcsProvider, err := mapVCSProvider(task.VCSProvider)
	if err != nil {
		vcsProvider = gcpspanner.VCSProviderGitHub
	}
	scanLog := gcpspanner.CodeSubscriptionScanLog{
		ID:              uuid.NewString(),
		VCSProvider:     vcsProvider,
		VCSRepositoryID: task.VCSRepositoryID,
		CommitSHA:       task.CommitSHA,
		Branch:          task.Branch,
		ScanStatus:      gcpspanner.ScanStatusFailed,
		FilesScanned:    0,
		DirectivesFound: 0,
		ErrorMessage:    &errMsg,
		ErrorClass:      &class,
		RulesVersion:    new(codescan.RulesVersion),
		ScannedAt:       scannedAt,
	}

	// Detached from the task context: the row must be written even when the scan failed because
	// its own deadline expired.
	logCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.failedLogTimeout)
	defer cancel()
	if logErr := s.spannerSyncer.InsertCodeSubscriptionScanLog(logCtx, scanLog); logErr != nil {
		slog.ErrorContext(ctx, "failed to insert failed scan log", "error", logErr, "repo", task.RepositoryFullName)
	}
}

// fail records a FAILED scan and returns an error that is never a transient failure: the message
// is acknowledged, and the next vcs-sync run republishes the repository. Code scan tasks are
// idempotent reconciliation triggers, so redelivering them from Pub/Sub only adds DLQ noise and
// can replay stale commits.
func (s *Scanner) fail(
	ctx context.Context,
	task codescantaskv1.CodeScanTaskEvent,
	scannedAt time.Time,
	msg string,
	err error,
) error {
	class := classifyScanError(ctx, err)
	logArgs := []any{
		"repo", task.RepositoryFullName,
		"commit", task.CommitSHA,
		"branch", task.Branch,
		"error_class", string(class),
		"error", err,
	}
	if class.IsPermanent() {
		slog.WarnContext(ctx, "repository scan failed", logArgs...)
	} else {
		slog.ErrorContext(ctx, "repository scan failed", logArgs...)
	}

	s.recordFailedScanLog(ctx, task, scannedAt, class, fmt.Sprintf("%s: %v", msg, err))

	return fmt.Errorf("%s: %w", msg, err)
}

func newTarballScan() *tarballScan {
	return &tarballScan{
		occurrencesByQuery: make(map[string][]gcpspanner.SubscriptionOccurrence),
		triggersByQuery:    make(map[string]map[gcpspanner.SubscriptionTrigger]struct{}),
		filesScanned:       0,
		scanBytes:          0,
		parsedPaths:        make(map[string]struct{}),
		unscannedPaths:     make(map[string]struct{}),
		truncated:          truncationNone,
		decompressedSize:   0,
	}
}

func (t *tarballScan) addDirectives(directives []codescan.Directive) {
	for _, d := range directives {
		queryKey := strings.ToLower(strings.TrimSpace(d.TargetQuery))
		if _, exists := t.triggersByQuery[queryKey]; !exists {
			t.triggersByQuery[queryKey] = make(map[gcpspanner.SubscriptionTrigger]struct{})
		}
		t.triggersByQuery[queryKey][gcpspanner.SubscriptionTrigger(d.Trigger)] = struct{}{}

		t.occurrencesByQuery[queryKey] = append(t.occurrencesByQuery[queryKey],
			gcpspanner.SubscriptionOccurrence{
				FilePath:       d.FilePath,
				LineNumber:     int64(d.LineNumber),
				CommentSnippet: d.RawSnippet,
			})
	}
}

func (s *Scanner) scanTarballStream(
	ctx context.Context,
	archiveStream io.Reader,
) (*tarballScan, error) {
	gzr, err := gzip.NewReader(archiveStream)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize gzip reader: %w", err)
	}
	defer func() {
		_ = gzr.Close()
	}()

	capped := &archiveCapReader{r: gzr, remaining: s.maxArchiveBytes, read: 0}
	tr := tar.NewReader(capped)
	result := newTarballScan()
	defer func() {
		result.decompressedSize = capped.read
	}()

	for {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		header, tarErr := tr.Next()
		if errors.Is(tarErr, io.EOF) {
			break
		}
		if errors.Is(tarErr, errArchiveCapExceeded) {
			result.truncated = truncationArchiveSize

			break
		}
		if tarErr != nil {
			return nil, fmt.Errorf("failed to read tar entry: %w", tarErr)
		}

		stop, entryErr := s.scanEntry(ctx, tr, header, result)
		if entryErr != nil {
			return nil, entryErr
		}
		if stop {
			break
		}
	}

	return result, nil
}

// scanEntry scans one tar entry into result. It returns stop=true when a deterministic limit
// ends the scan early (result.truncated records which one).
func (s *Scanner) scanEntry(
	ctx context.Context,
	tr *tar.Reader,
	header *tar.Header,
	result *tarballScan,
) (bool, error) {
	if header.Typeflag != tar.TypeReg {
		return false, nil
	}

	cleanPath := stripTarRootPrefix(header.Name)
	if cleanPath == "" || !codescan.IsSupportedWebExtension(filepath.Ext(cleanPath)) {
		return false, nil
	}

	if result.filesScanned >= s.maxScannedFiles {
		slog.WarnContext(ctx, "reached maximum scanned files limit, terminating archive scan",
			"max_files", s.maxScannedFiles)
		result.truncated = truncationFileCount

		return true, nil
	}

	if result.scanBytes+header.Size > MaxScanTotalBytes {
		slog.WarnContext(ctx, "reached maximum scan bytes limit, terminating archive scan",
			"total_bytes", result.scanBytes, "max_bytes", MaxScanTotalBytes)
		result.truncated = truncationScanBytes

		return true, nil
	}

	if header.Size > MaxBlobSizeBytes {
		slog.WarnContext(ctx, "skipping large file in archive", "path", cleanPath, "size", header.Size)
		result.unscannedPaths[cleanPath] = struct{}{}

		return false, nil
	}

	result.scanBytes += header.Size

	directives, parseErr := codescan.ParseReader(
		io.LimitReader(tr, MaxBlobSizeBytes+1),
		cleanPath,
		codescan.SubscriptionTriggerFeatureBaselinePromoteToWidely,
	)
	if errors.Is(parseErr, errArchiveCapExceeded) {
		result.truncated = truncationArchiveSize

		return true, nil
	}
	if parseErr != nil {
		if errors.Is(parseErr, errStall) || ctx.Err() != nil {
			return false, fmt.Errorf("failed to read archive entry %q: %w", cleanPath, parseErr)
		}
		// The file could not be parsed (e.g. a line over the parser's limit). Its directives are
		// unknown, so it must not cause subscriptions to be marked OBSOLETE.
		slog.WarnContext(ctx, "failed to parse directives from file", "path", cleanPath, "error", parseErr)
		result.unscannedPaths[cleanPath] = struct{}{}

		return false, nil
	}

	result.filesScanned++
	result.parsedPaths[cleanPath] = struct{}{}
	result.addDirectives(directives)

	return false, nil
}

// buildSubscriptions converts scan results into subscription inputs. Keys are sorted so that,
// when a repository has more than MaxSubscriptionsPerRepo directives, the same subset is kept on
// every scan. It also returns every key found, so keys beyond the cap are not marked OBSOLETE.
func (s *Scanner) buildSubscriptions(
	task codescantaskv1.CodeScanTaskEvent,
	vcsProvider gcpspanner.VCSProvider,
	occurrencesByQuery map[string][]gcpspanner.SubscriptionOccurrence,
	triggersByQuery map[string]map[gcpspanner.SubscriptionTrigger]struct{},
) ([]gcpspanner.CodeSubscriptionInput, map[string]struct{}, bool) {
	queryKeys := make([]string, 0, len(occurrencesByQuery))
	found := make(map[string]struct{}, len(occurrencesByQuery))
	for queryKey := range occurrencesByQuery {
		queryKeys = append(queryKeys, queryKey)
		found[queryKey] = struct{}{}
	}
	slices.Sort(queryKeys)

	capped := false
	if len(queryKeys) > MaxSubscriptionsPerRepo {
		queryKeys = queryKeys[:MaxSubscriptionsPerRepo]
		capped = true
	}

	subscriptions := make([]gcpspanner.CodeSubscriptionInput, 0, len(queryKeys))
	for _, queryKey := range queryKeys {
		triggersMap := triggersByQuery[queryKey]
		triggers := make([]gcpspanner.SubscriptionTrigger, 0, len(triggersMap))
		for t := range triggersMap {
			triggers = append(triggers, t)
		}
		slices.Sort(triggers)

		subscriptions = append(subscriptions, gcpspanner.CodeSubscriptionInput{
			VCSProvider:        vcsProvider,
			VCSInstallationID:  task.VCSInstallationID,
			VCSRepositoryID:    task.VCSRepositoryID,
			RepositoryFullName: task.RepositoryFullName,
			TargetQuery:        queryKey,
			Triggers:           triggers,
			Occurrences:        occurrencesByQuery[queryKey],
		})
	}

	return subscriptions, found, capped
}

// archiveConnInfo captures which connection served the archive download, for telemetry.
// Only the remote address is recorded, never the URL (it can contain a token).
type archiveConnInfo struct {
	mu         sync.Mutex
	remoteAddr string
	reused     bool
}

func (a *archiveConnInfo) trace() *httptrace.ClientTrace {
	//nolint:exhaustruct // Only the GotConn hook is needed.
	return &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			if info.Conn == nil {
				return
			}
			a.mu.Lock()
			defer a.mu.Unlock()
			// The archive request is the last connection obtained, after the archive link lookup.
			a.remoteAddr = info.Conn.RemoteAddr().String()
			a.reused = info.Reused
		},
	}
}

func (a *archiveConnInfo) snapshot() (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.remoteAddr, a.reused
}

// ProcessTask downloads a repository archive, scans it for directives, and applies the results.
//
// Every failure is acknowledged and recorded as a FAILED scan log row with an ErrorClass; it never
// returns event.ErrTransientFailure. vcs-sync republishes repositories whose latest scan failed.
func (s *Scanner) ProcessTask(ctx context.Context, task codescantaskv1.CodeScanTaskEvent) (err error) {
	// Spanner stores microseconds; truncating keeps the fence comparison exact.
	scannedAt := s.now().UTC().Truncate(time.Microsecond)

	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "recovered panic while scanning repository",
				"repo", task.RepositoryFullName, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			err = s.fail(ctx, task, scannedAt, "scan aborted", fmt.Errorf("%w: %v", errScanPanicked, r))
		}
	}()

	if task.CommitSHA == DeletedBranchSHA {
		slog.InfoContext(ctx, "skipping deleted branch commit", "repo", task.RepositoryFullName)

		return nil
	}
	if !task.IsDefaultBranch {
		slog.InfoContext(ctx, "skipping non-default branch commit", "repo", task.RepositoryFullName, "branch", task.Branch)

		return nil
	}

	vcsProvider, provErr := mapVCSProvider(task.VCSProvider)
	if provErr != nil {
		return s.fail(ctx, task, scannedAt, "invalid vcs provider", provErr)
	}

	scanCtx, cancelScan := context.WithTimeout(ctx, s.scanPhaseTimeout)
	defer cancelScan()

	var token string
	if s.tokenProvider != nil && task.VCSInstallationID != "" {
		t, tokenErr := s.tokenProvider.GetInstallationToken(scanCtx, task.VCSInstallationID)
		if tokenErr != nil {
			return s.fail(ctx, task, scannedAt, "failed to get installation token", tokenErr)
		}
		token = t
	}

	owner, repo := splitOwnerRepo(task.RepositoryFullName)
	fetcher := s.clientFactory(token)
	ref := task.CommitSHA
	if ref == "" {
		ref = task.Branch
	}

	connInfo := &archiveConnInfo{mu: sync.Mutex{}, remoteAddr: "", reused: false}
	downloadCtx, cancelDownload := context.WithCancelCause(httptrace.WithClientTrace(scanCtx, connInfo.trace()))
	defer cancelDownload(nil)

	archiveStream, err := fetcher.DownloadTarball(downloadCtx, owner, repo, ref)
	if err != nil {
		return s.fail(ctx, task, scannedAt, "failed to download tarball", err)
	}
	defer func() {
		_ = archiveStream.Close()
	}()

	watchdog := newThroughputWatchdog(archiveStream, cancelDownload, s.stallWindow, s.minBytesPerWindow)
	scan, scanErr := s.scanTarballStream(downloadCtx, watchdog)
	if scanErr != nil {
		return s.fail(ctx, task, scannedAt, "failed to scan tarball", scanErr)
	}

	subscriptions, found, capped := s.buildSubscriptions(task, vcsProvider, scan.occurrencesByQuery, scan.triggersByQuery)
	truncated := scan.truncated
	if truncated == truncationNone && capped {
		truncated = truncationSubscriptions
	}

	scanStatus := gcpspanner.ScanStatusSuccess
	var statusMsg *string
	if truncated != truncationNone {
		scanStatus = gcpspanner.ScanStatusTruncated
		msg := "truncated: " + string(truncated)
		statusMsg = &msg
	}

	var directivesFound int64
	for _, sub := range subscriptions {
		directivesFound += int64(len(sub.Occurrences))
	}

	// Detached from the scan-phase deadline so a scan that finishes near its limit can commit.
	syncCtx, cancelSync := context.WithTimeout(context.WithoutCancel(ctx), s.syncTimeout)
	defer cancelSync()
	applied, syncErr := s.spannerSyncer.SyncRepositoryScan(syncCtx, gcpspanner.RepositoryScanSync{
		VCSProvider:     vcsProvider,
		VCSRepositoryID: task.VCSRepositoryID,
		Subscriptions:   subscriptions,
		Obsolescence: gcpspanner.ObsolescenceScope{
			// Only stream-level truncation leaves files unread. The subscription cap is a complete
			// scan: every file was parsed, and `found` keeps capped keys from being obsoleted.
			Truncated:      scan.truncated != truncationNone,
			FoundTargets:   found,
			ParsedPaths:    scan.parsedPaths,
			UnscannedPaths: scan.unscannedPaths,
		},
		ScanLog: gcpspanner.CodeSubscriptionScanLog{
			ID:              uuid.NewString(),
			VCSProvider:     vcsProvider,
			VCSRepositoryID: task.VCSRepositoryID,
			CommitSHA:       task.CommitSHA,
			Branch:          task.Branch,
			ScanStatus:      scanStatus,
			FilesScanned:    scan.filesScanned,
			DirectivesFound: directivesFound,
			ErrorMessage:    statusMsg,
			ErrorClass:      nil,
			RulesVersion:    new(codescan.RulesVersion),
			ScannedAt:       scannedAt,
		},
	})
	if syncErr != nil {
		return s.fail(ctx, task, scannedAt, "failed to sync code subscriptions",
			fmt.Errorf("%w: %w", errSyncFailed, syncErr))
	}

	remoteAddr, connReused := connInfo.snapshot()
	logArgs := []any{
		"repo", task.RepositoryFullName,
		"commit", task.CommitSHA,
		"files_scanned", scan.filesScanned,
		"subscriptions_synced", len(subscriptions),
		"directives_found", directivesFound,
		"status", scanStatus,
		"truncated_reason", string(truncated),
		"archive_bytes_compressed", watchdog.bytesRead(),
		"archive_bytes_decompressed", scan.decompressedSize,
		"duration_ms", s.now().Sub(scannedAt).Milliseconds(),
		"remote_addr", remoteAddr,
		"conn_reused", connReused,
	}
	if !applied {
		slog.InfoContext(ctx, "repository scan superseded by a newer scan, not applied",
			append(logArgs, "superseded", true)...)

		return nil
	}

	slog.InfoContext(ctx, "successfully scanned repository", append(logArgs, "superseded", false)...)

	return nil
}
