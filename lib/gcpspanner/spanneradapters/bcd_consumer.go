// Copyright 2024 Google LLC
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

package spanneradapters

import (
	"context"
	"errors"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/bcdconsumertypes"
)

// NewBCDWorkflowConsumer constructs an adapter for the bcd consumer service.
func NewBCDWorkflowConsumer(client BCDWorkflowSpannerClient) *BCDConsumer {
	return &BCDConsumer{client: client}
}

// BCDWorkflowSpannerClient expects a subset of the functionality from lib/gcpspanner that
// only apply to inserting BCD data.
type BCDWorkflowSpannerClient interface {
	InsertBrowserRelease(ctx context.Context, release gcpspanner.BrowserRelease) error
}

// BCDConsumer is the adapter that takes data from the BCD workflow and prepares
// it to be stored in the spanner database.
type BCDConsumer struct {
	client BCDWorkflowSpannerClient
}

func (b *BCDConsumer) InsertBrowserReleases(ctx context.Context, releases []bcdconsumertypes.BrowserRelease) error {
	for _, release := range releases {
		err := b.client.InsertBrowserRelease(ctx, gcpspanner.BrowserRelease{
			BrowserName:    string(release.BrowserName),
			BrowserVersion: release.BrowserVersion,
			ReleaseDate:    release.ReleaseDate,
		})
		if err != nil {
			return errors.Join(bcdconsumertypes.ErrUnableToStoreBrowserRelease, err)
		}
	}

	return nil
}
