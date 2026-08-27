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
	"context"
	"errors"
	"sync"
	"testing"
	"time"

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

			proc := NewVCSSyncProcessor(storer, tokenProvider, repoLister, publisher)
			err := proc.Process(ctx, NewJobArguments())

			if (err != nil) != tc.wantErr {
				t.Fatalf("Process() error = %v, wantErr %v", err, tc.wantErr)
			}

			if len(publisher.published) != tc.expectedPublished {
				t.Errorf("published tasks count = %d, want %d", len(publisher.published), tc.expectedPublished)
			}
		})
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

	ghInstallations := []*github.Installation{
		{
			ID: &instID,
			Account: &github.User{
				Login:     &login,
				Type:      &accType,
				ID:        &accID,
				AvatarURL: new("https://avatars.github.com/u/456"),
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

	proc := NewVCSSyncProcessor(storer, tokenProvider, repoLister, publisher)
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
}
