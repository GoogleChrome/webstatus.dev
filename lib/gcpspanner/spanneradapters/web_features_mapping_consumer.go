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

	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/webfeaturesmappingtypes"
)

// WebFeaturesMappingClient is the client for interacting with the web features mapping data in Spanner.
type WebFeaturesMappingClient interface {
	SyncWebFeaturesMappingData(ctx context.Context, data []gcpspanner.WebFeaturesMappingData) error
}

// WebFeaturesMappingConsumer is a consumer for web features mapping data.
type WebFeaturesMappingConsumer struct {
	client WebFeaturesMappingClient
}

// NewWebFeaturesMappingConsumer creates a new WebFeaturesMappingConsumer.
func NewWebFeaturesMappingConsumer(client WebFeaturesMappingClient) *WebFeaturesMappingConsumer {
	return &WebFeaturesMappingConsumer{
		client: client,
	}
}

// SyncWebFeaturesMappingData syncs the web features mapping data to Spanner.
func (a *WebFeaturesMappingConsumer) SyncWebFeaturesMappingData(
	ctx context.Context,
	mappings webfeaturesmappingtypes.WebFeaturesMappings,
) error {
	data := make([]gcpspanner.WebFeaturesMappingData, 0, len(mappings))
	for featureID, mapping := range mappings {
		vendorPositions, err := VendorPositionsToNullJSON(mapping.StandardsPositions)
		if err != nil {
			return err
		}
		spannerData := gcpspanner.WebFeaturesMappingData{
			WebFeatureID:    featureID,
			VendorPositions: vendorPositions,
		}
		data = append(data, spannerData)
	}

	return a.client.SyncWebFeaturesMappingData(ctx, data)
}

// VendorPositionsToNullJSON converts a slice of StandardsPosition into a spanner.NullJSON object.
// This is used by both the real data consumer and the fake data generator to ensure consistency.
func VendorPositionsToNullJSON(positions []webfeaturesmappingtypes.StandardsPosition) (spanner.NullJSON, error) {
	if len(positions) == 0 {
		return spanner.NullJSON{Valid: false, Value: nil}, nil
	}

	return spanner.NullJSON{Value: positions, Valid: true}, nil
}
