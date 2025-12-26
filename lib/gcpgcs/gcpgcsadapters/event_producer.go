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

package gcpgcsadapters

import (
	"context"
	"path"

	"github.com/GoogleChrome/webstatus.dev/lib/blobtypes"
)

type EventProducerBlobStorageClient interface {
	WriteBlob(ctx context.Context, path string, data []byte, opts ...blobtypes.WriteOption) error
	ReadBlob(ctx context.Context, path string, opts ...blobtypes.ReadOption) (*blobtypes.Blob, error)
}

type EventProducer struct {
	client     EventProducerBlobStorageClient
	bucketName string
}

func NewEventProducer(client EventProducerBlobStorageClient, bucketName string) *EventProducer {
	return &EventProducer{client: client, bucketName: bucketName}
}

func (e *EventProducer) Store(ctx context.Context, dirs []string, key string, data []byte) (string, error) {
	filepath := append([]string{e.bucketName}, dirs...)
	// Add the key as the final element.
	filepath = append(filepath, key)
	path := path.Join(filepath...)
	if err := e.client.WriteBlob(ctx, path, data, blobtypes.WithContentType("application/json")); err != nil {
		return "", err
	}

	return path, nil
}

func (e *EventProducer) Get(ctx context.Context, fullpath string) ([]byte, error) {
	blob, err := e.client.ReadBlob(ctx, fullpath)
	if err != nil {
		return nil, err
	}

	return blob.Data, nil
}
