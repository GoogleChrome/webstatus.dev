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
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
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
	GetAllMovedWebFeatures(ctx context.Context) ([]gcpspanner.MovedWebFeature, error)
	SyncLatestFeatureDeveloperSignals(ctx context.Context, data []gcpspanner.FeatureDeveloperSignal) error
}

func (c *DeveloperSignalsConsumer) GetAllMovedWebFeatures(
	ctx context.Context) (map[string]webdxfeaturetypes.FeatureMovedData, error) {
	movedFeatures, err := c.client.GetAllMovedWebFeatures(ctx)
	if err != nil {
		return nil, err
	}

	return convertGCPSpannerMovedFeaturesToMap(movedFeatures), nil
}

func migrateMovedFeaturesForDeveloperSignals(
	ctx context.Context,
	data *developersignaltypes.FeatureDeveloperSignals,
	movedFeatures map[string]webdxfeaturetypes.FeatureMovedData) error {
	allFeaturesSet := make(map[string]struct{}, len(*data))
	for featureID := range *data {
		allFeaturesSet[featureID] = struct{}{}
	}

	return NewMigrator(movedFeatures, allFeaturesSet, data).Migrate(ctx,
		func(oldKey, newKey string, data *developersignaltypes.FeatureDeveloperSignals) {
			(*data)[newKey] = (*data)[oldKey]
			delete(*data, oldKey)
		})
}

// SyncLatestFeatureDeveloperSignals handles the conversion of developer signals between the workflow/API input
// format and the format used by the GCP Spanner client.
func (c *DeveloperSignalsConsumer) SyncLatestFeatureDeveloperSignals(
	ctx context.Context, data *developersignaltypes.FeatureDeveloperSignals) error {
	if data == nil || len(*data) == 0 {
		return nil
	}

	// Get all moved web features so we can migrate old data to the new feature ID.
	// We do this here because we want to avoid doing this for every metric.
	// We also want to avoid doing this in the spanner client because we want to
	// keep the spanner client as a generic library.
	movedFeatures, err := c.GetAllMovedWebFeatures(ctx)
	if err != nil {
		return err
	}

	err = migrateMovedFeaturesForDeveloperSignals(ctx, data, movedFeatures)
	if err != nil {
		return err
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
