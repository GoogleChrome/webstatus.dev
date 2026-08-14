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

package gh

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-github/v79/github"
)

type MockGitClient struct {
	GetTreeFunc func(
		ctx context.Context,
		owner, repo, sha string,
		recursive bool,
	) (*github.Tree, *github.Response, error)
	GetBlobRawFunc func(
		ctx context.Context,
		owner, repo, sha string,
	) ([]byte, *github.Response, error)
}

func (m *MockGitClient) GetTree(
	ctx context.Context,
	owner, repo, sha string,
	recursive bool,
) (*github.Tree, *github.Response, error) {
	if m.GetTreeFunc == nil {
		panic("GetTreeFunc not set")
	}

	return m.GetTreeFunc(ctx, owner, repo, sha, recursive)
}

func (m *MockGitClient) GetBlobRaw(
	ctx context.Context,
	owner, repo, sha string,
) ([]byte, *github.Response, error) {
	if m.GetBlobRawFunc == nil {
		panic("GetBlobRawFunc not set")
	}

	return m.GetBlobRawFunc(ctx, owner, repo, sha)
}

func TestGetCommitTree(t *testing.T) {
	// nolint:exhaustruct
	testTree := &github.Tree{
		SHA: new("tree-sha"),
		// nolint:exhaustruct
		Entries: []*github.TreeEntry{
			{
				Path: new("src/app.ts"),
				Type: new("blob"),
				SHA:  new("blob-sha-1"),
			},
		},
		Truncated: new(false),
	}

	testCases := []struct {
		name        string
		mockFunc    func(ctx context.Context, owner, repo, sha string, recursive bool) (*github.Tree, *github.Response, error)
		expectErr   bool
		expectedSHA string
	}{
		{
			name: "Success",
			mockFunc: func(_ context.Context, _, _, _ string, _ bool) (*github.Tree, *github.Response, error) {
				return testTree, createTestGithubResponse(0), nil
			},
			expectErr:   false,
			expectedSHA: "tree-sha",
		},
		{
			name: "API Error",
			mockFunc: func(_ context.Context, _, _, _ string, _ bool) (*github.Tree, *github.Response, error) {
				// nolint:exhaustruct
				return nil, &github.Response{
					Response: &http.Response{StatusCode: http.StatusNotFound},
				}, errors.New("tree not found")
			},
			expectErr:   true,
			expectedSHA: "",
		},
		{
			name: "Nil Tree",
			mockFunc: func(_ context.Context, _, _, _ string, _ bool) (*github.Tree, *github.Response, error) {
				return nil, createTestGithubResponse(0), nil
			},
			expectErr:   true,
			expectedSHA: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &Client{
				repoClient:   nil,
				gitClient:    &MockGitClient{GetTreeFunc: tc.mockFunc, GetBlobRawFunc: nil},
				issuesClient: nil,
			}

			tree, err := client.GetCommitTree(context.Background(), "owner", "repo", "sha123")
			if tc.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.expectErr && tree.GetSHA() != tc.expectedSHA {
				t.Errorf("expected tree SHA %s, got %s", tc.expectedSHA, tree.GetSHA())
			}
		})
	}
}

func TestGetBlobContent(t *testing.T) {
	normalBlob := []byte("console.log('hello');")
	largeBlob := bytes.Repeat([]byte("a"), maxBlobSizeBytes+1)

	testCases := []struct {
		name      string
		mockFunc  func(ctx context.Context, owner, repo, sha string) ([]byte, *github.Response, error)
		expectErr bool
		errIs     error
	}{
		{
			name: "Success",
			mockFunc: func(_ context.Context, _, _, _ string) ([]byte, *github.Response, error) {
				return normalBlob, createTestGithubResponse(0), nil
			},
			expectErr: false,
			errIs:     nil,
		},
		{
			name: "Blob Too Large",
			mockFunc: func(_ context.Context, _, _, _ string) ([]byte, *github.Response, error) {
				return largeBlob, createTestGithubResponse(0), nil
			},
			expectErr: true,
			errIs:     ErrBlobTooLarge,
		},
		{
			name: "API Error",
			mockFunc: func(_ context.Context, _, _, _ string) ([]byte, *github.Response, error) {
				// nolint:exhaustruct
				return nil, &github.Response{
					Response: &http.Response{StatusCode: http.StatusNotFound},
				}, errors.New("blob missing")
			},
			expectErr: true,
			errIs:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &Client{
				repoClient:   nil,
				gitClient:    &MockGitClient{GetTreeFunc: nil, GetBlobRawFunc: tc.mockFunc},
				issuesClient: nil,
			}

			content, err := client.GetBlobContent(context.Background(), "owner", "repo", "blob-sha")
			if tc.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.errIs != nil && !errors.Is(err, tc.errIs) {
				t.Errorf("expected error %v, got %v", tc.errIs, err)
			}
			if !tc.expectErr && !bytes.Equal(content, normalBlob) {
				t.Errorf("expected content %s, got %s", string(normalBlob), string(content))
			}
		})
	}
}
