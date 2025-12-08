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

package blobtypes

// Blob represents a generic binary object and its metadata.
type Blob struct {
	// Data is the raw byte content of the blob.
	Data []byte

	// Generation is the version ID of the object (used for optimistic locking).
	// On GCS, this maps to the 'Generation' field.
	Generation int64

	// ContentType is the MIME type of the object (e.g., "application/json").
	ContentType string

	// Metadata contains custom key/value pairs.
	Metadata map[string]string
}

// WriteSettings holds the optional configuration for writing a blob.
type WriteSettings struct {
	ContentType        *string
	Metadata           *map[string]string
	ExpectedGeneration *int64
}

// WriteOption defines a functional option for configuring a write operation.
type WriteOption func(*WriteSettings)

// WithContentType sets the MIME type of the blob.
func WithContentType(contentType string) WriteOption {
	return func(s *WriteSettings) {
		s.ContentType = &contentType
	}
}

// WithMetadata sets the custom metadata for the blob.
func WithMetadata(metadata map[string]string) WriteOption {
	return func(s *WriteSettings) {
		s.Metadata = &metadata
	}
}

// WithExpectedGeneration sets the precondition for the write.
//
//	0  -> Create new file, fail if exists.
//	-1 -> Force overwrite (ignore generation).
//	>0 -> CAS (Compare-And-Swap): Fail if current generation != expected.
func WithExpectedGeneration(gen int64) WriteOption {
	return func(s *WriteSettings) {
		s.ExpectedGeneration = &gen
	}
}

// ReadSettings holds the optional configuration for reading a blob.
type ReadSettings struct {
	// Generation is the version ID of the object to read.
	// nil means the latest version.
	Generation *int64
}

// ReadOption defines a functional option for configuring a read operation.
type ReadOption func(*ReadSettings)

// WithGeneration sets the specific generation (version) of the blob to read.
func WithGeneration(gen int64) ReadOption {
	return func(s *ReadSettings) {
		s.Generation = &gen
	}
}
