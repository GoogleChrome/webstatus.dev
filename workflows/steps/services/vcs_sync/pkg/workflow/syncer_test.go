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

type mockInstallationLister struct {
	installations []gcpspanner.VCSInstallation
	err           error
}

func (m *mockInstallationLister) ListVCSInstallations(_ context.Context) ([]gcpspanner.VCSInstallation, error) {
	return m.installations, m.err
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

type mockGitHubRepoLister struct {
	listReposFunc func(ctx context.Context, token string, opts *github.ListOptions) ([]*github.Repository, error)
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

			lister := &mockInstallationLister{
				installations: tc.installations,
				err:           tc.listInstErr,
			}
			tokenProvider := &mockTokenProvider{
				tokenMap: nil,
				err:      tc.tokenErr,
			}
			repoLister := &mockGitHubRepoLister{
				listReposFunc: tc.listReposFunc,
			}
			publisher := &mockTaskPublisher{
				mu:        sync.Mutex{},
				published: nil,
				err:       tc.publishErr,
			}

			proc := NewVCSSyncProcessor(lister, tokenProvider, repoLister, publisher)
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
