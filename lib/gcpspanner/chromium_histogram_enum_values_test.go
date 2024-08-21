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
	}
}

func insertSampleChromiumHistogramEnumValues(
	ctx context.Context, t *testing.T, c *Client, enumIDMap map[string]string) {
	enumValues := getSampleChromiumHistogramEnumValues(enumIDMap)
	for _, enumValue := range enumValues {
		err := c.UpsertChromiumHistogramEnumValue(ctx, enumValue)
		if err != nil {
			t.Fatalf("unable to insert sample enum value. error %s", err)
		}
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

func TestUpsertChromiumHistogramEnumValue(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	enumIDMap := insertSampleChromiumHistogramEnums(ctx, t, spannerClient)
	insertSampleChromiumHistogramEnumValues(ctx, t, spannerClient, enumIDMap)
	enumValues, err := spannerClient.ReadAllChromiumHistogramEnumValues(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	sampleHistogramsEnumValues := getSampleChromiumHistogramEnumValues(enumIDMap)
	if !slices.Equal[[]ChromiumHistogramEnumValue](sampleHistogramsEnumValues, enumValues) {
		t.Errorf("unequal enums. expected %+v actual %+v", sampleHistogramsEnumValues, enumValues)
	}

	err = spannerClient.UpsertChromiumHistogramEnumValue(ctx, ChromiumHistogramEnumValue{
		ChromiumHistogramEnumID: enumIDMap["WebDXFeatureObserver"],
		BucketID:                1,
		// Should not update
		Label: "CompressionStreamssssssss",
	})
	if err != nil {
		t.Errorf("unexpected error during update. %s", err.Error())
	}

	enumValues, err = spannerClient.ReadAllChromiumHistogramEnumValues(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}

	// Should be the same. No updates should happen.
	if !slices.Equal[[]ChromiumHistogramEnumValue](sampleHistogramsEnumValues, enumValues) {
		t.Errorf("unequal enum values after update. expected %+v actual %+v", sampleHistogramsEnumValues, enumValues)
	}
}
