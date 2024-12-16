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
	"slices"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
	"google.golang.org/api/iterator"
)

type dailyChromiumHistogramMetricToInsert struct {
	DailyChromiumHistogramMetric
	histogramName metricdatatypes.HistogramName
	bucketID      int64
}

func getSampleDailyChromiumHistogramMetricsToInsert() []dailyChromiumHistogramMetricToInsert {
	return []dailyChromiumHistogramMetricToInsert{
		// CompressionStreams
		{
			histogramName: metricdatatypes.WebDXFeatureEnum,
			bucketID:      1,
			DailyChromiumHistogramMetric: DailyChromiumHistogramMetric{
				Day: civil.Date{
					Year:  2000,
					Month: time.January,
					Day:   1,
				},
				Rate: *big.NewRat(7, 100),
			},
		},
		{
			histogramName: metricdatatypes.WebDXFeatureEnum,
			bucketID:      1,
			DailyChromiumHistogramMetric: DailyChromiumHistogramMetric{
				Day: civil.Date{
					Year:  2000,
					Month: time.January,
					Day:   2,
				},
				Rate: *big.NewRat(8, 100),
			},
		},
		// ViewTransitions
		{
			histogramName: metricdatatypes.WebDXFeatureEnum,
			bucketID:      2,
			DailyChromiumHistogramMetric: DailyChromiumHistogramMetric{
				Day: civil.Date{
					Year:  2000,
					Month: time.January,
					Day:   1,
				},
				Rate: *big.NewRat(91, 100),
			},
		},
	}
}

type testSpannerDailyChromiumHistogramMetric struct {
	ChromiumHistogramEnumValueID string     `spanner:"ChromiumHistogramEnumValueID"`
	Day                          civil.Date `spanner:"Day"`
	Rate                         big.Rat    `spanner:"Rate"`
}

func getSampleDailyChromiumHistogramMetricsToCheckBeforeUpdate(
	enumValueLabelToIDMap map[string]string) []testSpannerDailyChromiumHistogramMetric {
	return []testSpannerDailyChromiumHistogramMetric{
		// AnotherLabel
		{
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["CompressionStreams"],
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   1,
			},
			Rate: *big.NewRat(7, 100),
		},
		// CompressionStreams
		{
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["CompressionStreams"],
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   2,
			},
			Rate: *big.NewRat(8, 100),
		},
		// ViewTransitions
		{
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["ViewTransitions"],
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   1,
			},
			Rate: *big.NewRat(91, 100),
		},
	}
}

func getSampleDailyChromiumHistogramMetricsToCheckAfterUpdate(
	enumValueLabelToIDMap map[string]string) []testSpannerDailyChromiumHistogramMetric {
	return []testSpannerDailyChromiumHistogramMetric{
		// AnotherLabel
		{
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["CompressionStreams"],
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   1,
			},
			Rate: *big.NewRat(7, 100),
		},
		// CompressionStreams
		{
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["CompressionStreams"],
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   2,
			},
			Rate: *big.NewRat(8, 100),
		},
		// ViewTransitions
		{
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["ViewTransitions"],
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   1,
			},
			Rate: *big.NewRat(90, 100),
		},
		{
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["ViewTransitions"],
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   20,
			},
			Rate: *big.NewRat(93, 100),
		},
	}
}

func getSampleLatestDailyChromiumHistogramMetricsToCheckBeforeUpdate(
	enumValueLabelToIDMap map[string]string) []testSpannerDailyChromiumHistogramMetric {
	return []testSpannerDailyChromiumHistogramMetric{
		// CompressionStreams
		{
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["CompressionStreams"],
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   2,
			},
			Rate: *big.NewRat(8, 100),
		},
		// ViewTransitions
		{
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["ViewTransitions"],
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   1,
			},
			Rate: *big.NewRat(91, 100),
		},
	}
}

func getSampleLatestDailyChromiumHistogramMetricsToCheckAfterUpdate(
	enumValueLabelToIDMap map[string]string) []testSpannerDailyChromiumHistogramMetric {
	return []testSpannerDailyChromiumHistogramMetric{
		// CompressionStreams
		{
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["CompressionStreams"],
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   2,
			},
			Rate: *big.NewRat(8, 100),
		},
		// ViewTransitions
		{
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["ViewTransitions"],
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   20,
			},
			Rate: *big.NewRat(93, 100),
		},
	}
}

func insertTestDailyChromiumHistogramMetrics(
	ctx context.Context, c *Client, t *testing.T, values []dailyChromiumHistogramMetricToInsert) {
	for _, metricToInsert := range values {
		err := c.UpsertDailyChromiumHistogramMetric(
			ctx,
			metricToInsert.histogramName,
			metricToInsert.bucketID,
			metricToInsert.DailyChromiumHistogramMetric,
		)
		if err != nil {
			t.Errorf("unexpected error during insert of Chromium metrics. %s", err.Error())
		}
	}
}

// Helper method to get all the metrics in a stable order.
func (c *Client) readAllDailyChromiumHistogramMetrics(
	ctx context.Context) ([]testSpannerDailyChromiumHistogramMetric, error) {
	stmt := spanner.NewStatement(
		`SELECT
			ChromiumHistogramEnumValueID, Day, Rate
		FROM DailyChromiumHistogramMetrics
		ORDER BY Rate ASC`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []testSpannerDailyChromiumHistogramMetric
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var metric testSpannerDailyChromiumHistogramMetric
		if err := row.ToStruct(&metric); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		ret = append(ret, metric)
	}

	return ret, nil
}

func (c *Client) readAllLatestDailyChromiumHistogramMetrics(
	ctx context.Context) ([]testSpannerDailyChromiumHistogramMetric, error) {
	stmt := spanner.NewStatement(
		`SELECT
			ldchm.ChromiumHistogramEnumValueID,
			dchm.Day,
			dchm.Rate
		FROM LatestDailyChromiumHistogramMetrics ldchm
		JOIN DailyChromiumHistogramMetrics dchm
		ON ldchm.ChromiumHistogramEnumValueID = dchm.ChromiumHistogramEnumValueID
		AND ldchm.Day = dchm.Day
		ORDER BY Rate ASC`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []testSpannerDailyChromiumHistogramMetric
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var metric testSpannerDailyChromiumHistogramMetric
		if err := row.ToStruct(&metric); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		ret = append(ret, metric)
	}

	return ret, nil
}

func TestUpsertDailyChromiumHistogramMetric(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	idMap := setupRequiredTablesForWebFeatureChromiumHistogramEnum(ctx, t)
	sampleEnums := getSampleChromiumHistogramEnums()
	enumIDMap := insertTestChromiumHistogramEnums(ctx, spannerClient, t, sampleEnums)
	sampleEnumValues := getSampleChromiumHistogramEnumValues(enumIDMap)
	enumValueLabelToIDMap := insertTestChromiumHistogramEnumValues(ctx, spannerClient, t, sampleEnumValues)
	spannerClient.createSampleWebFeatureChromiumHistogramEnums(ctx, t, idMap, enumValueLabelToIDMap)
	sampleMetrics := getSampleDailyChromiumHistogramMetricsToInsert()
	insertTestDailyChromiumHistogramMetrics(ctx, spannerClient, t, sampleMetrics)

	metricValues, err := spannerClient.readAllDailyChromiumHistogramMetrics(ctx)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	samples := getSampleDailyChromiumHistogramMetricsToCheckBeforeUpdate(enumValueLabelToIDMap)
	if !slices.EqualFunc(samples, metricValues, dailyMetricEquality) {
		t.Errorf("unequal metrics.\nexpected %+v\nreceived %+v", samples, metricValues)
	}

	latestMetricValues, err := spannerClient.readAllLatestDailyChromiumHistogramMetrics(ctx)
	if err != nil {
		t.Errorf("unexpected error during read all (latest metrics) %s", err.Error())
	}
	sampleLatestMetrics := getSampleLatestDailyChromiumHistogramMetricsToCheckBeforeUpdate(enumValueLabelToIDMap)
	if !slices.EqualFunc(sampleLatestMetrics, latestMetricValues, latestDailyMetricEquality) {
		t.Errorf("unequal metrics.\nexpected %+v\nreceived %+v", sampleLatestMetrics, latestMetricValues)
	}

	// Update the rate of one of the items.
	err = spannerClient.UpsertDailyChromiumHistogramMetric(ctx,
		metricdatatypes.WebDXFeatureEnum, 2, DailyChromiumHistogramMetric{
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   1,
			},
			// Change it to 90
			Rate: *big.NewRat(90, 100),
		})
	if err != nil {
		t.Errorf("unexpected error during update. %s", err.Error())
	}

	// Insert a newer value.
	err = spannerClient.UpsertDailyChromiumHistogramMetric(ctx,
		metricdatatypes.WebDXFeatureEnum, 2, DailyChromiumHistogramMetric{
			Day: civil.Date{
				Year:  2000,
				Month: time.January,
				Day:   20,
			},
			// New value of 93 for new day.
			Rate: *big.NewRat(93, 100),
		})
	if err != nil {
		t.Errorf("unexpected error during update. %s", err.Error())
	}
	metricValues, err = spannerClient.readAllDailyChromiumHistogramMetrics(ctx)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	samples = getSampleDailyChromiumHistogramMetricsToCheckAfterUpdate(enumValueLabelToIDMap)
	if !slices.EqualFunc(samples, metricValues, dailyMetricEquality) {
		t.Errorf("unequal metrics.\nexpected %+v\nreceived %+v", samples, metricValues)
	}

	latestMetricValues, err = spannerClient.readAllLatestDailyChromiumHistogramMetrics(ctx)
	if err != nil {
		t.Errorf("unexpected error during read all (latest metrics) %s", err.Error())
	}
	sampleLatestMetrics = getSampleLatestDailyChromiumHistogramMetricsToCheckAfterUpdate(enumValueLabelToIDMap)
	if !slices.EqualFunc(sampleLatestMetrics, latestMetricValues, latestDailyMetricEquality) {
		t.Errorf("unequal metrics.\nexpected %+v\nreceived %+v", sampleLatestMetrics, latestMetricValues)
	}
	t.Run("errors", func(t *testing.T) {
		unsuedMetric := DailyChromiumHistogramMetric{Day: civil.Date{
			Year:  2000,
			Month: time.January,
			Day:   1,
		},
			Rate: *big.NewRat(0, 100),
		}
		testCases := []struct {
			name          string
			histogram     metricdatatypes.HistogramName
			bucketID      int64
			expectedError error
		}{
			{
				name:          "bad histogram name",
				histogram:     "",
				bucketID:      0,
				expectedError: ErrUsageMetricUpsertNoHistogramFound,
			},
			{
				name:          "bad histogram bucket id",
				histogram:     metricdatatypes.WebDXFeatureEnum,
				bucketID:      0,
				expectedError: ErrUsageMetricUpsertNoHistogramEnumFound,
			},
			{
				name:          "bucket id exists but no matching web features entry",
				histogram:     metricdatatypes.WebDXFeatureEnum,
				bucketID:      3,
				expectedError: ErrUsageMetricUpsertNoFeatureIDFound,
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := spannerClient.UpsertDailyChromiumHistogramMetric(ctx, tc.histogram, tc.bucketID, unsuedMetric)
				if !errors.Is(err, tc.expectedError) {
					t.Errorf("expected %v, received %v", tc.expectedError, err)
				}
			})
		}
	})
}

func dailyMetricEquality(left, right testSpannerDailyChromiumHistogramMetric) bool {
	return reflect.DeepEqual(left, right)
}

func latestDailyMetricEquality(left, right testSpannerDailyChromiumHistogramMetric) bool {
	return reflect.DeepEqual(left, right)
}
