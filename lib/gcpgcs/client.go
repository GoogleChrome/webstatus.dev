// Copyright 2025 Google LLC
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

package gcpgcs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/GoogleChrome/webstatus.dev/lib/blobtypes"
	"google.golang.org/api/googleapi"
)

type Client struct {
	client *storage.Client
	bucket string
}

// NewClient creates a new GCP GCS client.
// It automatically respects STORAGE_EMULATOR_HOST if set in the environment.
func NewClient(ctx context.Context, bucket string) (*Client, error) {
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{client: storageClient, bucket: bucket}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) WriteBlob(ctx context.Context, path string, data []byte, opts ...blobtypes.WriteOption) error {
	config := blobtypes.WriteSettings{
		ContentType:        nil,
		Metadata:           nil,
		ExpectedGeneration: nil,
	}
	for _, opt := range opts {
		opt(&config)
	}

	obj := c.client.Bucket(c.bucket).Object(path)
	if config.ExpectedGeneration != nil {
		switch *config.ExpectedGeneration {
		case 0:
			// nolint:exhaustruct // WONTFIX: external struct
			obj = obj.If(storage.Conditions{DoesNotExist: true})
		case -1:
			// Ignore generation; always overwrite.
		default:
			// nolint:exhaustruct // WONTFIX: external struct
			obj = obj.If(storage.Conditions{GenerationMatch: *config.ExpectedGeneration})
		}
	}

	wc := obj.NewWriter(ctx)

	if config.ContentType != nil {
		wc.ContentType = *config.ContentType
	}
	if config.Metadata != nil {
		wc.Metadata = *config.Metadata
	}

	if _, err := wc.Write(data); err != nil {
		if isPreconditionFailedError(err) {
			slog.WarnContext(ctx, "gcpgcs: precondition failed", "path", path, "error", err)

			return blobtypes.ErrPreconditionFailed
		}
		slog.ErrorContext(ctx, "gcpgcs: failed to write blob", "path", path, "error", err)

		return err
	}

	if err := wc.Close(); err != nil {
		if isPreconditionFailedError(err) {
			slog.WarnContext(ctx, "gcpgcs: precondition failed while closing", "path", path, "error", err)

			return blobtypes.ErrPreconditionFailed
		}
		slog.WarnContext(ctx, "gcpgcs: failed to close after writing blob", "path", path, "error", err)

		return nil
	}

	return nil
}

func isPreconditionFailedError(err error) bool {
	var e *googleapi.Error
	if errors.As(err, &e) && e.Code == http.StatusPreconditionFailed {
		return true
	}

	return false
}

func (c *Client) ReadBlob(ctx context.Context, path string, opts ...blobtypes.ReadOption) (*blobtypes.Blob, error) {
	config := blobtypes.ReadSettings{
		Generation: nil,
	}
	for _, opt := range opts {
		opt(&config)
	}

	obj := c.client.Bucket(c.bucket).Object(path)
	if config.Generation != nil {
		obj = obj.Generation(*config.Generation)
	}

	rc, err := obj.NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			slog.InfoContext(ctx, "gcpgcs: blob not found", "path", path)

			return nil, blobtypes.ErrBlobNotFound
		}
		slog.ErrorContext(ctx, "gcpgcs: failed to create reader for blob", "path", path, "error", err)

		return nil, err
	}

	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		slog.ErrorContext(ctx, "gcpgcs: failed to read blob data", "path", path, "error", err)

		return nil, err
	}

	return &blobtypes.Blob{
		Data:        data,
		Generation:  rc.Attrs.Generation,
		ContentType: rc.Attrs.ContentType,
		Metadata:    rc.Metadata(),
	}, nil
}
