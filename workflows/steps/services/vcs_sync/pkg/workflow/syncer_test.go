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

package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/codescan"
	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/google/go-github/v79/github"
)

type mockInstallationStorer struct {
	mu            sync.Mutex
	installations []gcpspanner.VCSInstallation
	listErr       error
	upserted      []gcpspanner.VCSInstallation
	upsertErr     error
}

func (m *mockInstallationStorer) ListVCSInstallations(_ context.Context) ([]gcpspanner.VCSInstallation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.installations, m.listErr
}

func (m *mockInstallationStorer) UpsertVCSInstallation(
	_ context.Context,
	in gcpspanner.VCSInstallation,
) (*string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.upsertErr != nil {
		return nil, m.upsertErr
	}
	m.upserted = append(m.upserted, in)
	id := "upserted-id"

	return &id, nil
}

type mockTokenProvider struct {
	tokenMap map[string]string
	err      error
}

func (m *mockTokenProvider) GetInstallationToken(_ context.Context, installationID string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if tok, ok := m.tokenMap[installationID]; ok {
		return tok, nil
	}

	return "mock-token-" + installationID, nil
}

func (m *mockTokenProvider) GetAppToken() (string, error) {
	if m.err != nil {
		return "", m.err
	}

	return "mock-app-token", nil
}

type mockGitHubRepoLister struct {
	listReposFunc func(ctx context.Context, token string, opts *github.ListOptions) ([]*github.Repository, error)
	listInstFunc  func(ctx context.Context, token string, opts *github.ListOptions) ([]*github.Installation, error)
}

func (m *mockGitHubRepoLister) ListInstallationRepositories(
	ctx context.Context,
	token string,
	opts *github.ListOptions,
) ([]*github.Repository, error) {
	if m.listReposFunc == nil {
		return nil, nil
	}

	return m.listReposFunc(ctx, token, opts)
}

func (m *mockGitHubRepoLister) ListAppInstallations(
	ctx context.Context,
	token string,
	opts *github.ListOptions,
) ([]*github.Installation, error) {
	if m.listInstFunc == nil {
		return nil, nil
	}

	return m.listInstFunc(ctx, token, opts)
}

type mockTaskPublisher struct {
	mu        sync.Mutex
	published []codescantaskv1.CodeScanTaskEvent
	err       error
}

func (m *mockTaskPublisher) PublishCodeScanTask(_ context.Context, task codescantaskv1.CodeScanTaskEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.published = append(m.published, task)

	return nil
}

type mockScanHistoryReader struct {
	mu      sync.Mutex
	history map[string][]gcpspanner.RepositoryScanSummary
	err     error
	calls   [][]string
}

func newMockScanHistoryReader(
	history map[string][]gcpspanner.RepositoryScanSummary,
	err error,
) *mockScanHistoryReader {
	return &mockScanHistoryReader{mu: sync.Mutex{}, history: history, err: err, calls: nil}
}

func (m *mockScanHistoryReader) GetRecentRepositoryScans(
	_ context.Context,
	_ gcpspanner.VCSProvider,
	repoIDs []string,
	perRepoLimit int,
) (map[string][]gcpspanner.RepositoryScanSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, slices.Clone(repoIDs))
	if perRepoLimit != scanHistoryDepth {
		return nil, fmt.Errorf("unexpected per-repository limit %d", perRepoLimit)
	}
	if m.err != nil {
		return nil, m.err
	}

	return m.history, nil
}

//nolint:exhaustruct
func TestVCSSyncProcessor_Process(t *testing.T) {
	t.Parallel()

	testErr := errors.New("simulated error")
	now := time.Now().UTC()

	instGitHub := gcpspanner.VCSInstallation{
		ID:                  "inst-id-1",
		VCSProvider:         gcpspanner.VCSProviderGitHub,
		VCSInstallationID:   "gh-inst-12345",
		AccountLogin:        "GoogleChrome",
		AccountType:         "Organization",
		RepositorySelection: "all",
		Permissions:         gcpspanner.VCSPermissions{GitHub: nil},
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	testCases := []struct {
		name               string
		installations      []gcpspanner.VCSInstallation
		listInstErr        error
		tokenErr           error
		listReposFunc      func(ctx context.Context, token string, opts *github.ListOptions) ([]*github.Repository, error)
		publishErr         error
		expectedPublished  int
		wantErr            bool
		expectCancellation bool
	}{
		{
			name:               "empty installations does nothing",
			installations:      []gcpspanner.VCSInstallation{},
			listInstErr:        nil,
			tokenErr:           nil,
			listReposFunc:      nil,
			publishErr:         nil,
			expectedPublished:  0,
			wantErr:            false,
			expectCancellation: false,
		},
		{
			name:               "listing installations error fails process",
			installations:      nil,
			listInstErr:        testErr,
			tokenErr:           nil,
			listReposFunc:      nil,
			publishErr:         nil,
			expectedPublished:  0,
			wantErr:            true,
			expectCancellation: false,
		},
		{
			name: "unsupported provider is skipped without error",
			installations: []gcpspanner.VCSInstallation{
				{
					ID:                  "inst-id-unsupported",
					VCSProvider:         "gitlab",
					VCSInstallationID:   "gl-123",
					AccountLogin:        "GitLabOrg",
					AccountType:         "Organization",
					RepositorySelection: "all",
					Permissions:         gcpspanner.VCSPermissions{GitHub: nil},
					CreatedAt:           now,
					UpdatedAt:           now,
				},
			},
			listInstErr:        nil,
			tokenErr:           nil,
			listReposFunc:      nil,
			publishErr:         nil,
			expectedPublished:  0,
			wantErr:            false,
			expectCancellation: false,
		},
		{
			name:               "token provider error continues and returns joined error",
			installations:      []gcpspanner.VCSInstallation{instGitHub},
			listInstErr:        nil,
			tokenErr:           testErr,
			listReposFunc:      nil,
			publishErr:         nil,
			expectedPublished:  0,
			wantErr:            true,
			expectCancellation: false,
		},
		{
			name:          "github listing error returns error",
			installations: []gcpspanner.VCSInstallation{instGitHub},
			listInstErr:   nil,
			tokenErr:      nil,
			listReposFunc: func(_ context.Context, _ string, _ *github.ListOptions) ([]*github.Repository, error) {
				return nil, testErr
			},
			publishErr:         nil,
			expectedPublished:  0,
			wantErr:            true,
			expectCancellation: false,
		},
		{
			name:          "successful sync publishes scan tasks for all repositories",
			installations: []gcpspanner.VCSInstallation{instGitHub},
			listInstErr:   nil,
			tokenErr:      nil,
			listReposFunc: func(_ context.Context, _ string, opts *github.ListOptions) ([]*github.Repository, error) {
				if opts.Page > 1 {
					return []*github.Repository{}, nil
				}
				repoID1 := int64(101)
				repoID2 := int64(102)
				fullName1 := "GoogleChrome/webstatus.dev"
				fullName2 := "GoogleChrome/chromium-dashboard"
				branch1 := "main"
				branch2 := "master"

				return []*github.Repository{
					{
						ID:            &repoID1,
						FullName:      &fullName1,
						DefaultBranch: &branch1,
					},
					{
						ID:            &repoID2,
						FullName:      &fullName2,
						DefaultBranch: &branch2,
					},
				}, nil
			},
			publishErr:         nil,
			expectedPublished:  2,
			wantErr:            false,
			expectCancellation: false,
		},
		{
			name:          "404 client error from github is skipped and does not fail process",
			installations: []gcpspanner.VCSInstallation{instGitHub},
			listInstErr:   nil,
			tokenErr:      nil,
			listReposFunc: func(_ context.Context, _ string, _ *github.ListOptions) ([]*github.Repository, error) {
				return nil, &github.ErrorResponse{
					Response: &http.Response{
						StatusCode: http.StatusNotFound,
					},
					Message: "Not Found",
				}
			},
			publishErr:         nil,
			expectedPublished:  0,
			wantErr:            false,
			expectCancellation: false,
		},
		{
			name:               "cancelled context aborts processing",
			installations:      []gcpspanner.VCSInstallation{instGitHub},
			listInstErr:        nil,
			tokenErr:           nil,
			listReposFunc:      nil,
			publishErr:         nil,
			expectedPublished:  0,
			wantErr:            true,
			expectCancellation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			if tc.expectCancellation {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			storer := &mockInstallationStorer{
				mu:            sync.Mutex{},
				installations: tc.installations,
				listErr:       tc.listInstErr,
				upserted:      nil,
				upsertErr:     nil,
			}
			tokenProvider := &mockTokenProvider{
				tokenMap: nil,
				err:      tc.tokenErr,
			}
			repoLister := &mockGitHubRepoLister{
				listReposFunc: tc.listReposFunc,
				listInstFunc:  nil,
			}
			publisher := &mockTaskPublisher{
				mu:        sync.Mutex{},
				published: nil,
				err:       tc.publishErr,
			}

			proc := NewVCSSyncProcessor(storer, tokenProvider, repoLister, publisher,
				newMockScanHistoryReader(nil, nil), ParseSkipPolicy(""))
			err := proc.Process(ctx, NewJobArguments())

			if (err != nil) != tc.wantErr {
				t.Fatalf("Process() error = %v, wantErr %v", err, tc.wantErr)
			}

			if len(publisher.published) != tc.expectedPublished {
				t.Errorf("published tasks count = %d, want %d", len(publisher.published), tc.expectedPublished)
			}

			if tc.name == "successful sync publishes scan tasks for all repositories" {
				if len(publisher.published) > 0 && publisher.published[0].CommitSHA != "" {
					t.Errorf("expected CommitSHA to be empty for scheduled sync, got %s", publisher.published[0].CommitSHA)
				}
			}
		})
	}
}

//nolint:exhaustruct
func TestVCSSyncProcessor_PublishFailuresFailTheRun(t *testing.T) {
	t.Parallel()

	repoID := int64(101)
	fullName := "GoogleChrome/webstatus.dev"
	branch := "main"
	storer := &mockInstallationStorer{
		mu: sync.Mutex{},
		installations: []gcpspanner.VCSInstallation{{
			VCSProvider:       gcpspanner.VCSProviderGitHub,
			VCSInstallationID: "inst-1",
			AccountLogin:      "GoogleChrome",
		}},
		listErr:   nil,
		upserted:  nil,
		upsertErr: nil,
	}
	repoLister := &mockGitHubRepoLister{
		listReposFunc: func(_ context.Context, _ string, opts *github.ListOptions) ([]*github.Repository, error) {
			if opts.Page > 1 {
				return []*github.Repository{}, nil
			}

			return []*github.Repository{{ID: &repoID, FullName: &fullName, DefaultBranch: &branch}}, nil
		},
		listInstFunc: nil,
	}
	publisher := &mockTaskPublisher{mu: sync.Mutex{}, published: nil, err: errors.New("pubsub unavailable")}

	proc := NewVCSSyncProcessor(storer, &mockTokenProvider{tokenMap: nil, err: nil}, repoLister, publisher,
		newMockScanHistoryReader(nil, nil), ParseSkipPolicy(""))
	err := proc.Process(context.Background(), NewJobArguments())
	if !errors.Is(err, ErrScanTaskPublishFailed) {
		t.Fatalf("Process() error = %v, want ErrScanTaskPublishFailed so the execution is marked failed", err)
	}
}

//nolint:exhaustruct
func TestVCSSyncProcessor_ReconcileAppInstallations(t *testing.T) {
	t.Parallel()

	instID := int64(98765)
	login := "OrgFromGitHubAPI"
	accType := "Organization"
	accID := int64(456)
	repoSel := "all"
	now := time.Now().UTC()
	issuesPerm := "write"

	ghInstallations := []*github.Installation{
		{
			ID: &instID,
			Account: &github.User{
				Login:     &login,
				Type:      &accType,
				ID:        &accID,
				AvatarURL: new("https://avatars.github.com/u/456"),
			},
			Permissions: &github.InstallationPermissions{
				Issues: &issuesPerm,
			},
			RepositorySelection: &repoSel,
			CreatedAt:           &github.Timestamp{Time: now},
			UpdatedAt:           &github.Timestamp{Time: now},
		},
	}

	storer := &mockInstallationStorer{
		mu:            sync.Mutex{},
		installations: nil,
		listErr:       nil,
		upserted:      nil,
		upsertErr:     nil,
	}
	tokenProvider := &mockTokenProvider{tokenMap: nil, err: nil}
	repoLister := &mockGitHubRepoLister{
		listReposFunc: nil,
		listInstFunc: func(_ context.Context, _ string, opts *github.ListOptions) ([]*github.Installation, error) {
			if opts.Page > 1 {
				return []*github.Installation{}, nil
			}

			return ghInstallations, nil
		},
	}
	publisher := &mockTaskPublisher{mu: sync.Mutex{}, published: nil, err: nil}

	proc := NewVCSSyncProcessor(storer, tokenProvider, repoLister, publisher,
		newMockScanHistoryReader(nil, nil), ParseSkipPolicy(""))
	err := proc.reconcileGitHubAppInstallations(context.Background())
	if err != nil {
		t.Fatalf("reconcileGitHubAppInstallations() unexpected error: %v", err)
	}

	if len(storer.upserted) != 1 {
		t.Fatalf("expected 1 upserted installation, got %d", len(storer.upserted))
	}
	if storer.upserted[0].VCSInstallationID != "98765" {
		t.Errorf("expected installation ID '98765', got '%s'", storer.upserted[0].VCSInstallationID)
	}
	if storer.upserted[0].AccountLogin != login {
		t.Errorf("expected login '%s', got '%s'", login, storer.upserted[0].AccountLogin)
	}
	if storer.upserted[0].Permissions.GitHub == nil ||
		storer.upserted[0].Permissions.GitHub.Issues == nil ||
		*storer.upserted[0].Permissions.GitHub.Issues != gcpspanner.GitHubAppPermissionLevelWrite {
		t.Errorf("expected issues write permission, got %+v", storer.upserted[0].Permissions.GitHub)
	}
}

//nolint:exhaustruct
func TestToGitHubPermissions(t *testing.T) {
	t.Parallel()

	if perms := toGitHubPermissions(nil); perms != nil {
		t.Errorf("expected nil for nil permissions, got %+v", perms)
	}

	read := "read"
	write := "write"
	invalid := "unknown"

	ghPerms := &github.InstallationPermissions{
		Issues:       &write,
		Contents:     &read,
		Metadata:     &read,
		PullRequests: &write,
		Workflows:    &write,
		Actions:      &invalid,
	}

	result := toGitHubPermissions(ghPerms)
	if result == nil {
		t.Fatalf("expected non-nil result")
	}

	if result.Issues == nil || *result.Issues != gcpspanner.GitHubAppPermissionLevelWrite {
		t.Errorf("expected issues write, got %v", result.Issues)
	}
	if result.Contents == nil || *result.Contents != gcpspanner.GitHubAppPermissionLevelRead {
		t.Errorf("expected contents read, got %v", result.Contents)
	}
	if result.Metadata == nil || *result.Metadata != gcpspanner.GitHubAppPermissionLevelRead {
		t.Errorf("expected metadata read, got %v", result.Metadata)
	}
	if result.PullRequests == nil || *result.PullRequests != gcpspanner.GitHubAppPermissionLevelWrite {
		t.Errorf("expected pull_requests write, got %v", result.PullRequests)
	}
	if result.Workflows == nil || *result.Workflows != gcpspanner.GitHubAppPermissionLevelWrite {
		t.Errorf("expected workflows write, got %v", result.Workflows)
	}
	if result.Actions != nil {
		t.Errorf("expected actions to be nil for unknown permission level, got %v", result.Actions)
	}
}

//nolint:exhaustruct
func testRepo(id int64, fullName string, pushedAt time.Time) *github.Repository {
	repo := &github.Repository{ID: &id, FullName: &fullName, DefaultBranch: new("main")}
	if !pushedAt.IsZero() {
		repo.PushedAt = &github.Timestamp{Time: pushedAt}
	}

	return repo
}

func publishedRepoIDs(publisher *mockTaskPublisher) []string {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()

	ids := make([]string, 0, len(publisher.published))
	for _, task := range publisher.published {
		ids = append(ids, task.VCSRepositoryID)
	}
	slices.Sort(ids)

	return ids
}

//nolint:exhaustruct
func TestVCSSyncProcessor_SkipsRepositoriesThatDoNotNeedAScan(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	branchHeadScan := func(status gcpspanner.ScanStatus, scannedAt time.Time) gcpspanner.RepositoryScanSummary {
		return gcpspanner.RepositoryScanSummary{
			ID:           "scan",
			ScanStatus:   status,
			ScannedAt:    scannedAt,
			Branch:       "main",
			CommitSHA:    "",
			ErrorClass:   nil,
			RulesVersion: new(codescan.RulesVersion),
		}
	}

	storer := &mockInstallationStorer{
		mu: sync.Mutex{},
		installations: []gcpspanner.VCSInstallation{
			{VCSProvider: gcpspanner.VCSProviderGitHub, VCSInstallationID: "101", AccountLogin: "skip-enabled"},
			{VCSProvider: gcpspanner.VCSProviderGitHub, VCSInstallationID: "202", AccountLogin: "skip-disabled"},
		},
	}
	reposByToken := map[string][]*github.Repository{
		"mock-token-101": {
			testRepo(1, "skip-enabled/unchanged", now.Add(-2*time.Hour)),
			testRepo(2, "skip-enabled/pushed", now.Add(-10*time.Minute)),
			testRepo(3, "skip-enabled/never-scanned", now.Add(-time.Hour)),
			testRepo(4, "skip-enabled/failed-recently", now.Add(-3*time.Hour)),
		},
		"mock-token-202": {
			testRepo(5, "skip-disabled/unchanged", now.Add(-2*time.Hour)),
		},
	}
	repoLister := &mockGitHubRepoLister{
		listReposFunc: func(_ context.Context, token string, opts *github.ListOptions) ([]*github.Repository, error) {
			if opts.Page > 1 {
				return nil, nil
			}

			return reposByToken[token], nil
		},
	}
	history := newMockScanHistoryReader(map[string][]gcpspanner.RepositoryScanSummary{
		"1": {branchHeadScan(gcpspanner.ScanStatusSuccess, now.Add(-time.Hour))},
		"2": {branchHeadScan(gcpspanner.ScanStatusSuccess, now.Add(-time.Hour))},
		"4": {branchHeadScan(gcpspanner.ScanStatusFailed, now.Add(-2*time.Minute))},
		"5": {branchHeadScan(gcpspanner.ScanStatusSuccess, now.Add(-time.Hour))},
	}, nil)
	publisher := &mockTaskPublisher{mu: sync.Mutex{}}

	proc := NewVCSSyncProcessor(storer, &mockTokenProvider{}, repoLister, publisher, history, ParseSkipPolicy("101"))
	proc.now = func() time.Time { return now }
	if err := proc.Process(context.Background(), NewJobArguments()); err != nil {
		t.Fatalf("Process() unexpected error: %v", err)
	}

	// 1 is unchanged and 4 is backing off; 5 is unchanged but its installation does not skip.
	if got, want := publishedRepoIDs(publisher), []string{"2", "3", "5"}; !slices.Equal(got, want) {
		t.Errorf("published repositories = %v, want %v", got, want)
	}
	wantCalls := [][]string{{"1", "2", "3", "4"}, {"5"}}
	if !slices.EqualFunc(history.calls, wantCalls, slices.Equal[[]string]) {
		t.Errorf("scan history lookups = %v, want %v", history.calls, wantCalls)
	}
}

//nolint:exhaustruct
func TestVCSSyncProcessor_ScanHistoryErrorPublishesEveryRepository(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	storer := &mockInstallationStorer{
		mu: sync.Mutex{},
		installations: []gcpspanner.VCSInstallation{
			{VCSProvider: gcpspanner.VCSProviderGitHub, VCSInstallationID: "101", AccountLogin: "skip-enabled"},
		},
	}
	repoLister := &mockGitHubRepoLister{
		listReposFunc: func(_ context.Context, _ string, opts *github.ListOptions) ([]*github.Repository, error) {
			if opts.Page > 1 {
				return nil, nil
			}

			return []*github.Repository{
				testRepo(1, "skip-enabled/a", now.Add(-2*time.Hour)),
				testRepo(2, "skip-enabled/b", time.Time{}),
			}, nil
		},
	}
	publisher := &mockTaskPublisher{mu: sync.Mutex{}}

	proc := NewVCSSyncProcessor(storer, &mockTokenProvider{}, repoLister, publisher,
		newMockScanHistoryReader(nil, errors.New("spanner unavailable")), ParseSkipPolicy("*"))
	proc.now = func() time.Time { return now }
	if err := proc.Process(context.Background(), NewJobArguments()); err != nil {
		t.Fatalf("Process() unexpected error: %v", err)
	}

	if got, want := publishedRepoIDs(publisher), []string{"1", "2"}; !slices.Equal(got, want) {
		t.Errorf("published repositories = %v, want %v", got, want)
	}
}

//nolint:exhaustruct
func TestVCSSyncProcessor_LogsRepositoriesThatKeepFailing(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	failures := func(class gcpspanner.ScanErrorClass) []gcpspanner.RepositoryScanSummary {
		scans := make([]gcpspanner.RepositoryScanSummary, 0, 4)
		for i := range 4 {
			scans = append(scans, gcpspanner.RepositoryScanSummary{
				ID:           fmt.Sprintf("scan-%d", i),
				ScanStatus:   gcpspanner.ScanStatusFailed,
				ScannedAt:    now.Add(-time.Duration(i+1) * 10 * time.Minute),
				Branch:       "main",
				CommitSHA:    "",
				ErrorClass:   new(class),
				RulesVersion: new(codescan.RulesVersion),
			})
		}

		return scans
	}

	storer := &mockInstallationStorer{
		mu: sync.Mutex{},
		installations: []gcpspanner.VCSInstallation{
			{VCSProvider: gcpspanner.VCSProviderGitHub, VCSInstallationID: "101", AccountLogin: "owner"},
		},
	}
	repoLister := &mockGitHubRepoLister{
		listReposFunc: func(_ context.Context, _ string, opts *github.ListOptions) ([]*github.Repository, error) {
			if opts.Page > 1 {
				return nil, nil
			}

			return []*github.Repository{
				testRepo(1, "owner/timing-out", now.Add(-5*time.Hour)),
				testRepo(2, "owner/blocked", now.Add(-5*time.Hour)),
			}, nil
		},
	}
	history := newMockScanHistoryReader(map[string][]gcpspanner.RepositoryScanSummary{
		"1": failures(gcpspanner.ScanErrorClassTimeout),
		"2": failures(gcpspanner.ScanErrorClassPermanentClient),
	}, nil)

	var logs bytes.Buffer
	proc := NewVCSSyncProcessor(storer, &mockTokenProvider{}, repoLister, &mockTaskPublisher{mu: sync.Mutex{}},
		history, ParseSkipPolicy(""))
	proc.now = func() time.Time { return now }
	proc.logger = slog.New(slog.NewJSONHandler(&logs, nil))
	if err := proc.Process(context.Background(), NewJobArguments()); err != nil {
		t.Fatalf("Process() unexpected error: %v", err)
	}

	// The "Repository Repeatedly Failing to Scan" alert matches this exact message and reads "repo".
	var warned []string
	for line := range strings.SplitSeq(strings.TrimSpace(logs.String()), "\n") {
		var entry struct {
			Msg                 string `json:"msg"`
			Repo                string `json:"repo"`
			ConsecutiveFailures int    `json:"consecutive_failures"`
			ErrorClass          string `json:"error_class"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("log line is not JSON: %q: %v", line, err)
		}
		if entry.Msg != "repository repeatedly failing to scan" {
			continue
		}
		if entry.ConsecutiveFailures != 4 || entry.ErrorClass != "TIMEOUT" {
			t.Errorf("warning = %+v, want 4 consecutive TIMEOUT failures", entry)
		}
		warned = append(warned, entry.Repo)
	}
	if !slices.Equal(warned, []string{"owner/timing-out"}) {
		t.Errorf("repositories warned about = %v, want only owner/timing-out (permanent failures are excluded)", warned)
	}
}

//nolint:exhaustruct
func TestVCSSyncProcessor_ReadsScanHistoryPerPage(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	repos := make([]*github.Repository, 0, 101)
	for id := int64(1); id <= 101; id++ {
		repos = append(repos, testRepo(id, fmt.Sprintf("owner/repo-%d", id), now.Add(-2*time.Hour)))
	}
	storer := &mockInstallationStorer{
		mu: sync.Mutex{},
		installations: []gcpspanner.VCSInstallation{
			{VCSProvider: gcpspanner.VCSProviderGitHub, VCSInstallationID: "101", AccountLogin: "owner"},
		},
	}
	repoLister := &mockGitHubRepoLister{
		listReposFunc: func(_ context.Context, _ string, opts *github.ListOptions) ([]*github.Repository, error) {
			start := (opts.Page - 1) * opts.PerPage
			if start >= len(repos) {
				return nil, nil
			}

			return repos[start:min(start+opts.PerPage, len(repos))], nil
		},
	}
	// Only the repository on the second page has an up-to-date scan.
	history := newMockScanHistoryReader(map[string][]gcpspanner.RepositoryScanSummary{
		"101": {{
			ID:           "scan",
			ScanStatus:   gcpspanner.ScanStatusSuccess,
			ScannedAt:    now.Add(-time.Hour),
			Branch:       "main",
			CommitSHA:    "",
			ErrorClass:   nil,
			RulesVersion: new(codescan.RulesVersion),
		}},
	}, nil)
	publisher := &mockTaskPublisher{mu: sync.Mutex{}}

	proc := NewVCSSyncProcessor(storer, &mockTokenProvider{}, repoLister, publisher, history, ParseSkipPolicy("*"))
	proc.now = func() time.Time { return now }
	if err := proc.Process(context.Background(), NewJobArguments()); err != nil {
		t.Fatalf("Process() unexpected error: %v", err)
	}

	if len(history.calls) != 2 || len(history.calls[0]) != 100 || !slices.Equal(history.calls[1], []string{"101"}) {
		t.Errorf("scan history lookups = %d calls, want one per page (100 repositories, then [101])",
			len(history.calls))
	}
	published := publishedRepoIDs(publisher)
	if len(published) != 100 || slices.Contains(published, "101") {
		t.Errorf("published %d repositories (contains 101: %v), want the 100 on the first page",
			len(published), slices.Contains(published, "101"))
	}
}
