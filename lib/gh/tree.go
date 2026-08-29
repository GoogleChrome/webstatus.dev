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
	"context"
	"errors"
	"fmt"

	"github.com/google/go-github/v79/github"
)

var (
	// ErrBlobTooLarge indicates a file blob exceeds the maximum allowed size.
	ErrBlobTooLarge = errors.New("file blob exceeds maximum size of 1MB")
	// ErrNilTree indicates GitHub returned a nil tree object.
	ErrNilTree = errors.New("received nil tree from git API")
)

const maxBlobSizeBytes = 1024 * 1024 // 1 MB

// GetCommitTree fetches the full recursive git tree for a given commit SHA.
func (c *Client) GetCommitTree(ctx context.Context, owner, repo, sha string) (*github.Tree, error) {
	tree, _, err := c.gitClient.GetTree(ctx, owner, repo, sha, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch git tree for %s/%s@%s: %w", owner, repo, sha, err)
	}
	if tree == nil {
		return nil, ErrNilTree
	}

	return tree, nil
}

// GetBlobContent fetches raw content of a git blob by SHA, enforcing a 1MB limit.
// Note for callers: To prevent unnecessary network I/O and memory allocation,
// callers should pre-filter git tree entries by relevant file extension and size
// before requesting blob contents.
func (c *Client) GetBlobContent(ctx context.Context, owner, repo, sha string) ([]byte, error) {
	blob, _, err := c.gitClient.GetBlobRaw(ctx, owner, repo, sha)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob %s for %s/%s: %w", sha, owner, repo, err)
	}
	if len(blob) > maxBlobSizeBytes {
		return nil, ErrBlobTooLarge
	}

	return blob, nil
}
