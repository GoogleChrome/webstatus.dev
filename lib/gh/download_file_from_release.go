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
	"log/slog"
	"net/http"
	"strings"

	"github.com/GoogleChrome/webstatus.dev/lib/fetchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/httputils"
	"github.com/google/go-github/v74/github"
	"golang.org/x/mod/semver"
)

var (
	ErrRateLimit             = errors.New("rate limit hit")
	ErrAssetNotFound         = errors.New("asset not found")
	ErrUnableToDownloadAsset = errors.New("unable to download asset")
	ErrUnableToReadAsset     = errors.New("unable to download asset")
	ErrFatalError            = errors.New("fatal error using github")
)

// ReleaseFile represents a file in a given Github release.
type ReleaseFile struct {
	Contents io.ReadCloser
	Info     ReleaseInfo
}

type ReleaseInfo struct {
	// If the tag is valid, the will be non null.
	Tag *string
}

func (c *Client) DownloadFileFromRelease(
	ctx context.Context,
	owner, repo string,
	httpClient *http.Client,
	filePattern string) (*ReleaseFile, error) {
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

	fetcher, err := httputils.NewHTTPFetcher(downloadURL, httpClient)
	if err != nil {
		slog.ErrorContext(ctx, "unable to create fetcher", "error", err)

		return nil, err
	}

	body, err := fetcher.Fetch(ctx)
	if err != nil {
		if errors.Is(err, fetchtypes.ErrFailedToFetch) || errors.Is(err, fetchtypes.ErrUnexpectedResult) {
			return nil, errors.Join(ErrUnableToDownloadAsset, err)
		}

		return nil, errors.Join(ErrFatalError, err)
	}

	// Returns a tag or empty string if not found.
	tagName := release.GetTagName()
	// In the event the tag is missing the prefix "v", add it.
	// According https://pkg.go.dev/golang.org/x/mod/semver, the version must start with v
	if len(tagName) > 0 && !strings.HasPrefix(tagName, "v") {
		tagName = "v" + tagName
	}
	var tagNamePtr *string
	if semver.IsValid(tagName) {
		tagNamePtr = &tagName
	} else {
		slog.WarnContext(ctx, "invalid tag. it will not be used", "tag", tagName)
	}

	return &ReleaseFile{
		Contents: body,
		Info: ReleaseInfo{
			Tag: tagNamePtr,
		},
	}, nil
}
