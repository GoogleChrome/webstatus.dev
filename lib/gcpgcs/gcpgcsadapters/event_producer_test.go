// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcpgcsadapters

import (
	"context"
	"errors"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/blobtypes"
)

type mockBlobStorageClient struct {
	writeBlobCalled bool
	writeBlobReq    struct {
		path string
		data []byte
		opts []blobtypes.WriteOption
	}
	writeBlobErr error

	readBlobCalled bool
	readBlobReq    struct {
		path string
	}
	readBlobResp *blobtypes.Blob
	readBlobErr  error
}

func (m *mockBlobStorageClient) WriteBlob(_ context.Context, path string, data []byte,
	opts ...blobtypes.WriteOption) error {
	m.writeBlobCalled = true
	m.writeBlobReq.path = path
	m.writeBlobReq.data = data
	m.writeBlobReq.opts = opts

	return m.writeBlobErr
}

func (m *mockBlobStorageClient) ReadBlob(_ context.Context, path string,
	_ ...blobtypes.ReadOption) (*blobtypes.Blob, error) {
	m.readBlobCalled = true
	m.readBlobReq.path = path

	return m.readBlobResp, m.readBlobErr
}

func TestStore(t *testing.T) {
	bucketName := "test-bucket"
	data := []byte("test-data")

	tests := []struct {
		name         string
		dirs         []string
		key          string
		mockErr      error
		expectedPath string
		wantErr      bool
	}{
		{
			name:         "root directory",
			dirs:         []string{},
			key:          "file.json",
			mockErr:      nil,
			expectedPath: "test-bucket/file.json",
			wantErr:      false,
		},
		{
			name:         "nested directory",
			dirs:         []string{"folder1", "folder2"},
			key:          "file.json",
			mockErr:      nil,
			expectedPath: "test-bucket/folder1/folder2/file.json",
			wantErr:      false,
		},
		{
			name:         "write error",
			dirs:         []string{"folder"},
			key:          "file.json",
			mockErr:      errors.New("gcs error"),
			expectedPath: "test-bucket/folder/file.json",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := new(mockBlobStorageClient)
			mock.writeBlobErr = tc.mockErr
			adapter := NewEventProducer(mock, bucketName)

			path, err := adapter.Store(context.Background(), tc.dirs, tc.key, data)

			if (err != nil) != tc.wantErr {
				t.Errorf("Store() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !mock.writeBlobCalled {
				t.Fatal("WriteBlob not called")
			}

			if mock.writeBlobReq.path != tc.expectedPath {
				t.Errorf("path mismatch: got %q, want %q", mock.writeBlobReq.path, tc.expectedPath)
			}
			if string(mock.writeBlobReq.data) != string(data) {
				t.Errorf("data mismatch")
			}

			// Verify returned path matches the full path sent to GCS
			if err == nil && path != tc.expectedPath {
				t.Errorf("returned path mismatch: got %q, want %q", path, tc.expectedPath)
			}

			// Verify the options include the correct content type
			foundContentType := false
			for _, opt := range mock.writeBlobReq.opts {
				var config blobtypes.WriteSettings
				opt(&config)
				if config.ContentType != nil && *config.ContentType == "application/json" {
					foundContentType = true

					break
				}
			}
			if !foundContentType {
				t.Error("content type option not set to application/json")
			}
		})
	}
}

func TestGet(t *testing.T) {
	bucketName := "test-bucket"
	fullPath := "test-bucket/folder/file.json"
	data := []byte("test-data")

	tests := []struct {
		name     string
		mockResp *blobtypes.Blob
		mockErr  error
		wantData []byte
		wantErr  bool
	}{
		{
			name: "success",
			mockResp: &blobtypes.Blob{
				Data:        data,
				ContentType: "application/json",
				Metadata:    nil,
				Generation:  1,
			},
			mockErr:  nil,
			wantData: data,
			wantErr:  false,
		},
		{
			name:     "read error",
			mockResp: nil,
			mockErr:  errors.New("gcs error"),
			wantData: nil,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := new(mockBlobStorageClient)
			mock.readBlobResp = tc.mockResp
			mock.readBlobErr = tc.mockErr
			adapter := NewEventProducer(mock, bucketName)

			gotData, err := adapter.Get(context.Background(), fullPath)

			if (err != nil) != tc.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !mock.readBlobCalled {
				t.Fatal("ReadBlob not called")
			}

			if mock.readBlobReq.path != fullPath {
				t.Errorf("path mismatch: got %q, want %q", mock.readBlobReq.path, fullPath)
			}

			if err == nil && string(gotData) != string(tc.wantData) {
				t.Errorf("data mismatch: got %q, want %q", gotData, tc.wantData)
			}
		})
	}
}
