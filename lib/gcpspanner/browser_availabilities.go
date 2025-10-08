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

package gcpspanner

import (
	"context"
	"fmt"
	"log/slog"

	"cloud.google.com/go/spanner"
)

const browserFeatureAvailabilitiesTable = "BrowserFeatureAvailabilities"

// spannerBrowserFeatureAvailability is a wrapper for the browser availability
// information for a feature stored in spanner.
type spannerBrowserFeatureAvailability struct {
	WebFeatureID   string `spanner:"WebFeatureID"`
	BrowserName    string `spanner:"BrowserName"`
	BrowserVersion string `spanner:"BrowserVersion"`
}

// BrowserFeatureAvailability contains availability information for a particular
// feature in a browser.
type BrowserFeatureAvailability struct {
	BrowserName    string
	BrowserVersion string
}

// Implements the syncableEntityMapper interface for BrowserFeatureAvailability and spannerBrowserFeatureAvailability.
type browserFeatureAvailabilitySpannerMapper struct{}

// PreDeleteHook is a no-op for browser feature availabilities.
func (m browserFeatureAvailabilitySpannerMapper) PreDeleteHook(
	_ context.Context,
	_ *Client,
	_ []spannerBrowserFeatureAvailability,
) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

// GetChildDeleteKeyMutations returns nil as there are no child delete mutations for browser feature availabilities.
func (m browserFeatureAvailabilitySpannerMapper) GetChildDeleteKeyMutations(
	_ context.Context,
	_ *Client,
	_ []spannerBrowserFeatureAvailability,
) ([]ExtraMutationsGroup, error) {
	return nil, nil
}

func (m browserFeatureAvailabilitySpannerMapper) Table() string {
	return browserFeatureAvailabilitiesTable
}

func (m browserFeatureAvailabilitySpannerMapper) SelectAll() spanner.Statement {
	return spanner.NewStatement(fmt.Sprintf(`
	SELECT
		WebFeatureID, BrowserName, BrowserVersion
	FROM %s`, m.Table()))
}

func (m browserFeatureAvailabilitySpannerMapper) GetKeyFromExternal(in spannerBrowserFeatureAvailability) string {
	return fmt.Sprintf("%s-%s", in.WebFeatureID, in.BrowserName)
}

func (m browserFeatureAvailabilitySpannerMapper) GetKeyFromInternal(in spannerBrowserFeatureAvailability) string {
	return fmt.Sprintf("%s-%s", in.WebFeatureID, in.BrowserName)
}

func (m browserFeatureAvailabilitySpannerMapper) MergeAndCheckChanged(
	in spannerBrowserFeatureAvailability, existing spannerBrowserFeatureAvailability) (
	spannerBrowserFeatureAvailability, bool) {
	merged := spannerBrowserFeatureAvailability{
		WebFeatureID:   existing.WebFeatureID,
		BrowserName:    existing.BrowserName,
		BrowserVersion: in.BrowserVersion,
	}
	hasChanged := merged.BrowserVersion != existing.BrowserVersion

	return merged, hasChanged
}

func (m browserFeatureAvailabilitySpannerMapper) DeleteMutation(
	in spannerBrowserFeatureAvailability) *spanner.Mutation {
	return spanner.Delete(browserFeatureAvailabilitiesTable, spanner.Key{in.WebFeatureID, in.BrowserName})
}

// SyncBrowserFeatureAvailabilities reconciles the BrowserFeatureAvailabilities table with the provided
// list of availabilities.
func (c *Client) SyncBrowserFeatureAvailabilities(
	ctx context.Context,
	availabilities map[string][]BrowserFeatureAvailability,
) error {
	featureIDandKeys, err := c.FetchAllWebFeatureIDsAndKeys(ctx)
	if err != nil {
		return err
	}

	featureKeyToID := make(map[string]string, len(featureIDandKeys))
	for _, item := range featureIDandKeys {
		featureKeyToID[item.FeatureKey] = item.ID
	}

	var spannerAvailabilities []spannerBrowserFeatureAvailability
	for featureKey, featureAvailabilities := range availabilities {
		featureID, ok := featureKeyToID[featureKey]
		if !ok {
			slog.WarnContext(ctx, "unable to find feature id for feature key", "featureKey", featureKey)

			continue
		}
		for _, availability := range featureAvailabilities {
			spannerAvailabilities = append(spannerAvailabilities, spannerBrowserFeatureAvailability{
				WebFeatureID:   featureID,
				BrowserName:    availability.BrowserName,
				BrowserVersion: availability.BrowserVersion,
			})
		}
	}

	synchronizer := newEntitySynchronizer[browserFeatureAvailabilitySpannerMapper](c)

	return synchronizer.Sync(ctx, spannerAvailabilities)
}

func (c *Client) fetchAllBrowserAvailabilitiesWithTransaction(
	ctx context.Context, txn *spanner.ReadOnlyTransaction) ([]spannerBrowserFeatureAvailability, error) {
	var availabilities []spannerBrowserFeatureAvailability
	iter := txn.Read(ctx, browserFeatureAvailabilitiesTable, spanner.AllKeys(), []string{
		"BrowserName",
		"BrowserVersion",
		"WebFeatureID",
	})
	defer iter.Stop()
	err := iter.Do(func(row *spanner.Row) error {
		var entry spannerBrowserFeatureAvailability
		if err := row.ToStruct(&entry); err != nil {
			return err
		}
		availabilities = append(availabilities, entry)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return availabilities, nil
}
