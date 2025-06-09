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
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-github/v72/github"
)

type Downloader struct {
	ghClient   *Client
	httpClient *http.Client
}

func NewDownloader(ghClient *Client, httpClient *http.Client) *Downloader {
	return &Downloader{
		ghClient:   ghClient,
		httpClient: httpClient,
	}
}

func (d *Downloader) Download(
	ctx context.Context,
	repoOwner string,
	repoName string,
	_ *string) (
	io.ReadCloser, string, error) {
	// Check if repo exists.
	_, _, err := d.ghClient.client.Repositories.Get(ctx, repoOwner, repoName)
	if err != nil {
		return nil, "", err
	}

	archiveURL, _, err := d.ghClient.client.Repositories.GetArchiveLink(
		ctx, repoOwner, repoName, github.Tarball, nil, 10)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL.String(), nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}

	statusCode := resp.StatusCode
	if statusCode < 200 || statusCode > 299 {
		err := fmt.Errorf("bad status code:%d, unable to download wpt-metadata", statusCode)
		resp.Body.Close()

		return nil, "", err
	}

	return resp.Body, "main", nil
}

type ArchiveFile interface {
	GetData() io.Reader
	GetName() string
}

type ArchiveIterartor interface {
	Next() (ArchiveFile, error)
	Close() error
}

type ArchiveReader interface {
	NewIterator(io.ReadCloser) (ArchiveIterartor, error)
}

type FileWriter interface {
}
