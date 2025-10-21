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

package gcpspanner

import (
	"context"
	"fmt"
	"log/slog"

	"cloud.google.com/go/spanner"
)

const webFeaturesMappingDataTable = "WebFeaturesMappingData"

// WebFeaturesMappingData is the struct for ingesting WebFeaturesMappingData.
// The WebFeatureID field is expected to be the feature key.
type WebFeaturesMappingData struct {
	WebFeatureID    string
	VendorPositions spanner.NullJSON
}

// spannerWebFeaturesMappingData is the struct for the WebFeaturesMappingData table.
type spannerWebFeaturesMappingData struct {
	WebFeatureID    string `spanner:"WebFeatureID"`
	VendorPositions spanner.NullJSON
}

// webFeaturesMappingDataMapper is the mapper for the WebFeaturesMappingData table.
type webFeaturesMappingDataMapper struct{}

func (m webFeaturesMappingDataMapper) Table() string {
	return webFeaturesMappingDataTable
}

func (m webFeaturesMappingDataMapper) GetKeyFromExternal(in spannerWebFeaturesMappingData) string {
	return in.WebFeatureID
}

func (m webFeaturesMappingDataMapper) GetKeyFromInternal(in spannerWebFeaturesMappingData) string {
	return in.WebFeatureID
}

func (m webFeaturesMappingDataMapper) SelectAll() spanner.Statement {
	return spanner.NewStatement(fmt.Sprintf(`SELECT * FROM %s`, m.Table()))
}

func (m webFeaturesMappingDataMapper) MergeAndCheckChanged(
	in spannerWebFeaturesMappingData, existing spannerWebFeaturesMappingData) (spannerWebFeaturesMappingData, bool) {
	if !in.VendorPositions.Valid && !existing.VendorPositions.Valid {
		return existing, false
	}
	if (in.VendorPositions.Valid && !existing.VendorPositions.Valid) ||
		(!in.VendorPositions.Valid && existing.VendorPositions.Valid) ||
		(in.VendorPositions.Value != existing.VendorPositions.Value) {
		existing.VendorPositions = in.VendorPositions

		return existing, true
	}

	return existing, false
}

func (m webFeaturesMappingDataMapper) DeleteMutation(in spannerWebFeaturesMappingData) *spanner.Mutation {
	return spanner.Delete(webFeaturesMappingDataTable, spanner.Key{in.WebFeatureID})
}

func (m webFeaturesMappingDataMapper) GetChildDeleteKeyMutations(
	_ context.Context, _ *Client, _ []spannerWebFeaturesMappingData) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

func (m webFeaturesMappingDataMapper) PreDeleteHook(
	_ context.Context, _ *Client, _ []spannerWebFeaturesMappingData) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

// SyncWebFeaturesMappingData syncs the web features mapping data.
func (c *Client) SyncWebFeaturesMappingData(
	ctx context.Context,
	data []WebFeaturesMappingData,
) error {
	allFeatures, err := c.FetchAllWebFeatureIDsAndKeys(ctx)
	if err != nil {
		return err
	}

	featureKeyToID := make(map[string]string, len(allFeatures))
	for _, feature := range allFeatures {
		featureKeyToID[feature.FeatureKey] = feature.ID
	}

	spannerData := make([]spannerWebFeaturesMappingData, 0, len(data))
	for _, d := range data {
		internalID, ok := featureKeyToID[d.WebFeatureID]
		if !ok {
			slog.WarnContext(ctx, "feature key not found, skipping mapping data", "feature_key", d.WebFeatureID)

			continue
		}
		spannerData = append(spannerData, spannerWebFeaturesMappingData{
			WebFeatureID:    internalID,
			VendorPositions: d.VendorPositions,
		})
	}

	s := newEntitySynchronizer[webFeaturesMappingDataMapper](c)

	return s.Sync(ctx, spannerData)
}
