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

package differ

import (
	"errors"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/blobtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

// BlobFormat defines the serialization format and file extension for blobs.
type BlobFormat string

const (
	// BlobFormatJSON indicates the blob is serialized as standard JSON.
	BlobFormatJSON BlobFormat = "json"
)

// DiffResult encapsulates the complete output of a Run.
type DiffResult struct {
	HasChanges bool
	Format     BlobFormat
	StateBytes []byte
	StateID    string
	DiffBytes  []byte
	DiffID     string
	Summary    workertypes.EventSummary
	Reasons    []string
}

var (
	ErrTransient = errors.New("transient failure")
	ErrFatal     = errors.New("fatal error")
)

type FeatureDiffer struct {
	client     workertypes.FeatureFetcher
	migrator   *blobtypes.Migrator
	comparator workertypes.Comparator
	idGen      idGenerator
	now        func() time.Time
}

func NewFeatureDiffer(client workertypes.FeatureFetcher, comparator workertypes.Comparator) *FeatureDiffer {
	m := blobtypes.NewMigrator()

	return &FeatureDiffer{
		client:     client,
		migrator:   m,
		comparator: comparator,
		idGen:      &defaultIDGenerator{},
		now:        time.Now,
	}
}
