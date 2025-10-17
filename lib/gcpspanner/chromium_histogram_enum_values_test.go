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
	"cmp"
	"context"
	"errors"
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func getSampleChromiumHistogramEnumValues(histogramIDMap map[string]string) []ChromiumHistogramEnumValue {
	return []ChromiumHistogramEnumValue{
		{
			ChromiumHistogramEnumID: histogramIDMap["AnotherHistogram"],
			BucketID:                1,
			Label:                   "AnotherLabel",
		},
		{
			ChromiumHistogramEnumID: histogramIDMap["WebDXFeatureObserver"],
			BucketID:                1,
			Label:                   "CompressionStreams",
		},
		{
			ChromiumHistogramEnumID: histogramIDMap["WebDXFeatureObserver"],
			BucketID:                2,
			Label:                   "ViewTransitions",
		},
		// Create an enum that does not have a match in the web features table.
		{
			ChromiumHistogramEnumID: histogramIDMap["WebDXFeatureObserver"],
			BucketID:                3,
			Label:                   "WritingSuggestions",
		},
	}
}

func insertTestChromiumHistogramEnumValues(
	ctx context.Context,
	client *Client,
	t *testing.T,
	values []ChromiumHistogramEnumValue,
) {
	err := client.SyncChromiumHistogramEnumValues(ctx, values)
	if err != nil {
		t.Fatalf("unable to sync sample enum value. error %s", err)
	}
}

// Helper method to get all the enum values in a stable order.
func (c *Client) ReadAllChromiumHistogramEnumValues(
	ctx context.Context, t *testing.T) ([]ChromiumHistogramEnumValue, error) {
	stmt := spanner.NewStatement(
		`SELECT
			ChromiumHistogramEnumID, BucketID, Label
		FROM ChromiumHistogramEnumValues
		ORDER BY BucketID ASC`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []ChromiumHistogramEnumValue
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var enum spannerChromiumHistogramEnumValue
		if err := row.ToStruct(&enum); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		if enum.ChromiumHistogramEnumID == "" {
			t.Error("retrieved enum ID is empty")
		}
		ret = append(ret, enum.ChromiumHistogramEnumValue)
	}

	return ret, nil
}

func TestSyncChromiumHistogramEnumValues(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	sampleEnums := getSampleChromiumHistogramEnums()
	enumIDMap := insertTestChromiumHistogramEnums(ctx, spannerClient, t, sampleEnums)
	sampleEnumValues := getSampleChromiumHistogramEnumValues(enumIDMap)
	insertTestChromiumHistogramEnumValues(ctx, spannerClient, t, sampleEnumValues)
	enumValues, err := spannerClient.ReadAllChromiumHistogramEnumValues(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	sampleHistogramsEnumValues := getSampleChromiumHistogramEnumValues(enumIDMap)
	slices.SortFunc(enumValues, sortChromiumHistogramEnumValues)
	if !slices.Equal(sampleHistogramsEnumValues, enumValues) {
		t.Errorf("unequal enums.\nexpected %+v\nreceived %+v", sampleHistogramsEnumValues, enumValues)
	}

	// In this sync, we are:
	// 1. Keeping `AnotherLabel`
	// 2. Removing `CompressionStreams`
	// 3. Updating `ViewTransitions` (it should update)
	// 4. Keeping `WritingSuggestions` but it should be a new bucket ID. (like popover did in real life)
	err = spannerClient.SyncChromiumHistogramEnumValues(ctx, []ChromiumHistogramEnumValue{
		{
			ChromiumHistogramEnumID: enumIDMap["AnotherHistogram"],
			BucketID:                1,
			Label:                   "AnotherLabel",
		},
		{
			ChromiumHistogramEnumID: enumIDMap["WebDXFeatureObserver"],
			BucketID:                2,
			Label:                   "ViewTransitions-changed",
		},
		{
			ChromiumHistogramEnumID: enumIDMap["WebDXFeatureObserver"],
			BucketID:                33,
			Label:                   "WritingSuggestions",
		},
	})
	if err != nil {
		t.Errorf("unexpected error during update. %s", err.Error())
	}

	enumValues, err = spannerClient.ReadAllChromiumHistogramEnumValues(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	slices.SortFunc(enumValues, sortChromiumHistogramEnumValues)

	expected := []ChromiumHistogramEnumValue{
		{
			ChromiumHistogramEnumID: enumIDMap["AnotherHistogram"],
			BucketID:                1,
			Label:                   "AnotherLabel",
		},
		{
			ChromiumHistogramEnumID: enumIDMap["WebDXFeatureObserver"],
			BucketID:                2,
			Label:                   "ViewTransitions-changed",
		},
		{
			ChromiumHistogramEnumID: enumIDMap["WebDXFeatureObserver"],
			BucketID:                33,
			Label:                   "WritingSuggestions",
		},
	}
	slices.SortFunc(expected, sortChromiumHistogramEnumValues)

	// One item should be deleted, one item should be updated.
	if !slices.Equal(expected, enumValues) {
		t.Errorf("unequal enum values after update.\nexpected %+v\nreceived %+v", expected, enumValues)
	}
}

func sortChromiumHistogramEnumValues(left, right ChromiumHistogramEnumValue) int {
	return cmp.Compare(left.Label, right.Label)
}
