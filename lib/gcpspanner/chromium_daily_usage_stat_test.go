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
	"math/big"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/civil"
)

func testGetAllStats(ctx context.Context, c *Client, t *testing.T) {
	stats, token, err := c.ListChromeDailyUsageStatsForFeatureID(
		ctx,
		"feature2",
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.December, 1, 0, 0, 0, 0, time.UTC),
		5,
		nil,
	)

	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token != nil {
		t.Error("expected null token")
	}
	expectedStats := []ChromeDailyUsageStatWithDate{
		{
			Date: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   15,
			},
			Usage: big.NewRat(91, 100),
		},
		{
			Date: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   2,
			},
			Usage: big.NewRat(90, 100),
		},
		{
			Date: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   1,
			},
			Usage: big.NewRat(89, 100),
		},
	}

	if !reflect.DeepEqual(expectedStats, stats) {
		t.Errorf("unequal stats. expected (%+v) received (%+v) ", expectedStats, stats)
	}
}

func testGetSubsetStats(ctx context.Context, c *Client, t *testing.T) {
	stats, token, err := c.ListChromeDailyUsageStatsForFeatureID(
		ctx,
		"feature2",
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.January, 8, 0, 0, 0, 0, time.UTC),
		5,
		nil,
	)

	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token != nil {
		t.Error("expected null token")
	}
	expectedStats := []ChromeDailyUsageStatWithDate{
		{
			Date: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   2,
			},
			Usage: big.NewRat(90, 100),
		},
		{
			Date: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   1,
			},
			Usage: big.NewRat(89, 100),
		},
	}

	if !reflect.DeepEqual(expectedStats, stats) {
		t.Errorf("unequal stats. expected (%+v) received (%+v) ", expectedStats, stats)
	}
}

func testGetStatsPages(ctx context.Context, c *Client, t *testing.T) {
	stats, token, err := c.ListChromeDailyUsageStatsForFeatureID(
		ctx,
		"feature2",
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.December, 1, 0, 0, 0, 0, time.UTC),
		2,
		nil,
	)

	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token == nil {
		t.Error("expected token")
	}
	expectedStats := []ChromeDailyUsageStatWithDate{
		{
			Date: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   15,
			},
			Usage: big.NewRat(91, 100),
		},
		{
			Date: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   2,
			},
			Usage: big.NewRat(90, 100),
		},
	}
	if !reflect.DeepEqual(expectedStats, stats) {
		t.Errorf("unequal stats. expected (%+v) received (%+v) ", expectedStats, stats)
	}

	stats, token, err = spannerClient.ListChromeDailyUsageStatsForFeatureID(
		ctx,
		"feature2",
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.December, 1, 0, 0, 0, 0, time.UTC),
		2,
		token,
	)
	if !errors.Is(err, nil) {
		t.Errorf("expected no error during listing of metrics. received %s", err.Error())
	}
	if token != nil {
		t.Error("expected no token")
	}

	expectedStats = []ChromeDailyUsageStatWithDate{
		{
			Date: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   1,
			},
			Usage: big.NewRat(89, 100),
		},
	}
	if !reflect.DeepEqual(expectedStats, stats) {
		t.Errorf("unequal stats. expected (%+v) received (%+v) ", expectedStats, stats)
	}
}

func TestListChromeDailyUsageStatsForFeatureID(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	// Load up features.
	sampleFeatures := getSampleFeatures()
	webFeatureKeyToInternalFeatureID := map[string]string{}
	for _, feature := range sampleFeatures {
		id, err := spannerClient.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert of features. %s", err.Error())
		}
		webFeatureKeyToInternalFeatureID[feature.FeatureKey] = *id
	}
	addSampleChromiumUsageMetricsData(ctx, spannerClient, t, webFeatureKeyToInternalFeatureID)

	testGetAllStats(ctx, spannerClient, t)
	testGetSubsetStats(ctx, spannerClient, t)
	testGetStatsPages(ctx, spannerClient, t)
}
