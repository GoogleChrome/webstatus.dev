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

package spanneradapters

import (
	"context"

	"github.com/GoogleChrome/webstatus.dev/lib/developersignaltypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
)

// DeveloperSignalsConsumer handles the conversion of the developer signals between the downloaded
// format in the workflow and the format used by the GCP Spanner client.
type DeveloperSignalsConsumer struct {
	client DeveloperSignalsClient
}

// NewDeveloperSignalsConsumer constructs an adapter for the developer signals service.
func NewDeveloperSignalsConsumer(client DeveloperSignalsClient) *DeveloperSignalsConsumer {
	return &DeveloperSignalsConsumer{client: client}
}

// DeveloperSignalsClient expects a subset of the functionality from lib/gcpspanner that only apply to
// Developer Signals.
type DeveloperSignalsClient interface {
	SyncLatestFeatureDeveloperSignals(ctx context.Context, data []gcpspanner.FeatureDeveloperSignal) error
}

// SyncLatestFeatureDeveloperSignals handles the conversion of developer signals between the workflow/API input
// format and the format used by the GCP Spanner client.
func (c *DeveloperSignalsConsumer) SyncLatestFeatureDeveloperSignals(
	ctx context.Context, data *developersignaltypes.FeatureDeveloperSignals) error {
	if data == nil || len(*data) == 0 {
		return nil
	}

	signals := make([]gcpspanner.FeatureDeveloperSignal, 0, len(*data))
	for featureID, signal := range *data {
		signals = append(signals, gcpspanner.FeatureDeveloperSignal{
			WebFeatureKey: featureID,
			Upvotes:       signal.Upvotes,
			Link:          signal.Link,
		})
	}

	return c.client.SyncLatestFeatureDeveloperSignals(ctx, signals)

}
