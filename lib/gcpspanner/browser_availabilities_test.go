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
	"errors"
	"maps"
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func getSampleBrowserAvailabilities() map[string][]BrowserFeatureAvailability {
	return map[string][]BrowserFeatureAvailability{
		"feature1": {
			{
				BrowserName:    "fooBrowser",
				BrowserVersion: "0.0.0",
			},
			{
				BrowserName:    "barBrowser",
				BrowserVersion: "0.0.0",
			},
		},
		"feature2": {
			{
				BrowserName:    "barBrowser",
				BrowserVersion: "2.0.0",
			},
			{
				BrowserName:    "fooBrowser",
				BrowserVersion: "1.0.0",
			},
		},
	}
}

func setupRequiredTablesForBrowserFeatureAvailability(
	ctx context.Context,
	client *Client, t *testing.T) {
	sampleBrowserReleases := getSampleBrowserReleases()
	for _, release := range sampleBrowserReleases {
		err := client.InsertBrowserRelease(ctx, release)
		if err != nil {
			t.Errorf("unexpected error during insert of releases. %s", err.Error())
		}
	}
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		_, err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert of features. %s", err.Error())
		}
	}
}

// Helper method to get all the Availabilities in a stable order.
func (c *Client) ReadAllAvailabilities(ctx context.Context, _ *testing.T) ([]BrowserFeatureAvailability, error) {
	stmt := spanner.NewStatement(
		"SELECT BrowserName, BrowserVersion FROM BrowserFeatureAvailabilities ORDER BY BrowserVersion ASC, BrowserName ASC")
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []BrowserFeatureAvailability
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var availability BrowserFeatureAvailability
		if err := row.ToStruct(&availability); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		ret = append(ret, availability)
	}

	return ret, nil
}

func TestSyncBrowserFeatureAvailabilities(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	setupRequiredTablesForBrowserFeatureAvailability(ctx, spannerClient, t)
	sampleAvailabilities := getSampleBrowserAvailabilities()
	err := spannerClient.SyncBrowserFeatureAvailabilities(
		ctx, sampleAvailabilities)
	if err != nil {
		t.Errorf("unexpected error during insert. %s", err.Error())
	}

	expectedPage := []BrowserFeatureAvailability{
		// We will update this availability info for barBrowser later.
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "0.0.0",
		},
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "0.0.0",
		},
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "1.0.0",
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "2.0.0",
		},
	}

	availabilities, err := spannerClient.ReadAllAvailabilities(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	if !slices.Equal(expectedPage, availabilities) {
		t.Errorf("unequal availabilities.\nexpected %+v\nreceived %+v", expectedPage, availabilities)
	}

	// Update the availability info for feature1 on barBrowser to a later version
	updatedAvailabilities := maps.Clone(sampleAvailabilities)
	updatedAvailabilities["feature1"] = []BrowserFeatureAvailability{
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "0.0.0",
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "1.0.0",
		},
	}

	err = spannerClient.SyncBrowserFeatureAvailabilities(ctx, updatedAvailabilities)
	if err != nil {
		t.Errorf("unexpected error during update. %s", err.Error())
	}

	expectedPage = []BrowserFeatureAvailability{
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "0.0.0",
		},
		// This is the updated availability info for feature1 on barBrowser

		{
			BrowserName:    "barBrowser",
			BrowserVersion: "1.0.0",
		},
		{
			BrowserName:    "fooBrowser",
			BrowserVersion: "1.0.0",
		},
		{
			BrowserName:    "barBrowser",
			BrowserVersion: "2.0.0",
		},
	}
	availabilities, err = spannerClient.ReadAllAvailabilities(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	if !slices.Equal(expectedPage, availabilities) {
		t.Errorf("unequal availabilities.\nexpected %+v\nreceived %+v", expectedPage, availabilities)
	}
}
