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

// nolint:dupl // WONTFIX
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

func setupRequiredTablesForWebFeatureChromiumHistogramEnum(
	ctx context.Context,
	t *testing.T,
) map[string]string {
	ret := map[string]string{}
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		id, err := spannerClient.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())

			continue
		}
		ret[feature.FeatureKey] = *id
	}

	return ret
}

func (c *Client) createSampleWebFeatureChromiumHistogramEnums(
	ctx context.Context, t *testing.T, featureIDMap map[string]string, enumIDMap map[string]string) {
	err := c.UpsertWebFeatureChromiumHistogramEnum(ctx, WebFeatureChromiumHistogramEnum{
		WebFeatureID:            featureIDMap["feature1"],
		ChromiumHistogramEnumID: enumIDMap[testEnumKey("WebDXFeatureObserver", 1)],
	})
	if err != nil {
		t.Fatalf("failed to insert WebFeatureChromiumHistogramEnum. err: %s", err)
	}
	err = c.UpsertWebFeatureChromiumHistogramEnum(ctx, WebFeatureChromiumHistogramEnum{
		WebFeatureID:            featureIDMap["feature2"],
		ChromiumHistogramEnumID: enumIDMap[testEnumKey("WebDXFeatureObserver", 2)],
	})
	if err != nil {
		t.Fatalf("failed to insert WebFeatureChromiumHistogramEnum. err: %s", err)
	}
}

func (c *Client) ReadAllWebFeatureChromiumHistogramEnums(
	ctx context.Context, _ *testing.T) ([]WebFeatureChromiumHistogramEnum, error) {
	stmt := spanner.NewStatement(
		`SELECT
			WebFeatureID, ChromiumHistogramEnumID
		FROM WebFeatureChromiumHistogramEnums`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []WebFeatureChromiumHistogramEnum
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var chromiumHistogramEnum spannerWebFeatureChromiumHistogramEnum
		if err := row.ToStruct(&chromiumHistogramEnum); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		ret = append(ret, chromiumHistogramEnum.WebFeatureChromiumHistogramEnum)
	}

	return ret, nil
}

func sortWebFeatureChromiumHistogramEnums(left, right WebFeatureChromiumHistogramEnum) int {
	return cmp.Compare(left.WebFeatureID, right.WebFeatureID)
}

func webFeatureChromiumHistogramEnumEquality(left, right WebFeatureChromiumHistogramEnum) bool {
	return left.WebFeatureID == right.WebFeatureID &&
		left.ChromiumHistogramEnumID == right.ChromiumHistogramEnumID
}

func TestUpsertWebFeatureChromiumHistogramEnum(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	idMap := setupRequiredTablesForWebFeatureChromiumHistogramEnum(ctx, t)
	enumIDMap := insertSampleChromiumHistograms(ctx, t, spannerClient)
	spannerClient.createSampleWebFeatureChromiumHistogramEnums(ctx, t, idMap, enumIDMap)

	expected := []WebFeatureChromiumHistogramEnum{
		{
			WebFeatureID:            idMap["feature1"],
			ChromiumHistogramEnumID: enumIDMap[testEnumKey("WebDXFeatureObserver", 1)],
		},
		{
			WebFeatureID:            idMap["feature2"],
			ChromiumHistogramEnumID: enumIDMap[testEnumKey("WebDXFeatureObserver", 2)],
		},
	}
	slices.SortFunc(expected, sortWebFeatureChromiumHistogramEnums)

	chromiumHistogramEnums, err := spannerClient.ReadAllWebFeatureChromiumHistogramEnums(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all ChromiumHistogramEnums err: %s", err)
	}
	slices.SortFunc(chromiumHistogramEnums, sortWebFeatureChromiumHistogramEnums)

	if !slices.EqualFunc(expected, chromiumHistogramEnums, webFeatureChromiumHistogramEnumEquality) {
		t.Errorf("unequal ChromiumHistogramEnums.\nexpected %+v\nreceived %+v", expected, chromiumHistogramEnums)
	}

	// Upsert ChromiumHistogramEnum
	err = spannerClient.UpsertWebFeatureChromiumHistogramEnum(ctx, WebFeatureChromiumHistogramEnum{
		WebFeatureID:            idMap["feature2"],
		ChromiumHistogramEnumID: testEnumKey("WebDXFeatureObserver", 2),
	})
	if err != nil {
		t.Fatalf("unable to update ChromiumHistogramEnum err: %s", err)
	}

	// Should be the same.
	slices.SortFunc(expected, sortWebFeatureChromiumHistogramEnums)

	chromiumHistogramEnums, err = spannerClient.ReadAllWebFeatureChromiumHistogramEnums(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all ChromiumHistogramEnums err: %s", err)
	}
	slices.SortFunc(chromiumHistogramEnums, sortWebFeatureChromiumHistogramEnums)

	if !slices.EqualFunc(expected, chromiumHistogramEnums, webFeatureChromiumHistogramEnumEquality) {
		t.Errorf("unequal ChromiumHistogramEnums.\nexpected %+v\nreceived %+v", expected, chromiumHistogramEnums)
	}
}
