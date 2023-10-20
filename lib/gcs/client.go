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

package gcs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"

	"cloud.google.com/go/storage"
)

type Client struct {
	client       *storage.Client
	bucketName   string
	bucketHandle *storage.BucketHandle
}

func NewClient(client *storage.Client, bucket string) (*Client, error) {
	if bucket == "" {
		return nil, errors.New("provided bucket variable is empty")
	}

	return &Client{
		client:       client,
		bucketName:   bucket,
		bucketHandle: client.Bucket(bucket),
	}, nil
}

func (c *Client) Get(ctx context.Context, filename string) ([]byte, error) {
	reader, err := c.bucketHandle.Object(filename).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (c *Client) Store(ctx context.Context, data io.Reader, filename string) error {
	writer := c.bucketHandle.Object(filename).NewWriter(ctx)
	defer writer.Close()
	amount, err := io.Copy(writer, data)
	slog.Info("Copying data", "filename", filename, "bucket", c.bucketName, "amount", amount)

	return err
}

func (c *Client) GetLocation() string {
	return c.bucketName
}
