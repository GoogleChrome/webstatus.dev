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

package datastoreadapters

import (
	"context"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
)

// WebFeatureDatastoreClient expects a subset of the functionality from lib/gds that only apply to WebFeatures.
type WebFeatureDatastoreClient interface {
	UpsertFeatureMetadata(
		ctx context.Context,
		data gds.FeatureMetadata,
	) error
}

// NewWebFeaturesConsumer constructs an adapter for the web features consumer service.
func NewWebFeaturesConsumer(client WebFeatureDatastoreClient) *WebFeaturesConsumer {
	return &WebFeaturesConsumer{client: client}
}

// WebFeaturesConsumer handles the conversion of web feature data between the workflow/API input
// format and the format used by the GCP Datastore client.
type WebFeaturesConsumer struct {
	client WebFeatureDatastoreClient
}

func (c *WebFeaturesConsumer) InsertWebFeaturesMetadata(
	ctx context.Context,
	featureKeyToID map[string]string,
	data map[string]web_platform_dx__web_features.FeatureData) error {
	for featureKey, featureData := range data {
		featureID, found := featureKeyToID[featureKey]
		if !found {
			// Should never happen but let's log it out.
			slog.Warn("unable to find internal ID for featue key", "feature key", featureKey)

			continue
		}
		var canIUseIDs []string
		if featureData.Caniuse != nil && featureData.Caniuse.String != nil {
			canIUseIDs = []string{*featureData.Caniuse.String}
		} else if featureData.Caniuse != nil && len(featureData.Caniuse.StringArray) > 0 {
			canIUseIDs = featureData.Caniuse.StringArray
		}
		err := c.client.UpsertFeatureMetadata(ctx,
			gds.FeatureMetadata{
				WebFeatureID: featureID,
				Description:  featureData.Description,
				CanIUseIDs:   canIUseIDs,
			},
		)
		if err != nil {
			slog.Error("unable to upsert web feature metadata",
				"feature key", featureKey, "feature id", featureID, "error", err)

			return err
		}
	}

	return nil
}
