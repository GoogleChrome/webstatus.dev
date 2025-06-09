// Copyright 2023 Google LLC
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
	"io"
	"net/http"

	"github.com/google/go-github/v72/github"
)

var (
	ErrRateLimit             = errors.New("rate limit hit")
	ErrAssetNotFound         = errors.New("asset not found")
	ErrUnableToDownloadAsset = errors.New("unable to download asset")
	ErrUnableToReadAsset     = errors.New("unable to download asset")
	ErrFatalError            = errors.New("fatal error using github")
)

func (c *Client) DownloadFileFromRelease(
	ctx context.Context,
	owner, repo string,
	httpClient *http.Client,
	filePattern string) (io.ReadCloser, error) {
	release, _, err := c.repoClient.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		// nolint: exhaustruct // WONTFIX. This is an external package. Cannot control it.
		if errors.Is(err, &github.RateLimitError{}) {
			return nil, errors.Join(ErrRateLimit, err)
		}

		return nil, errors.Join(ErrFatalError, err)
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == nil {
			continue
		}
		if *asset.Name == filePattern && asset.BrowserDownloadURL != nil {
			downloadURL = *asset.BrowserDownloadURL

			break
		}
	}

	if downloadURL == "" {
		return nil, ErrAssetNotFound
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		downloadURL,
		nil,
	)
	if err != nil {
		// Currently cannot happen. But just in case something changes.
		return nil, errors.Join(ErrFatalError, err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Join(ErrUnableToDownloadAsset, err)
	}

	if resp.StatusCode != http.StatusOK {
		// Clean up by closing since we will not be returning the body
		resp.Body.Close()

		return nil, errors.Join(ErrUnableToDownloadAsset, err)
	}

	return resp.Body, nil
}
