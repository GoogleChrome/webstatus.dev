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
	err := c.UpsertWebFeatureChromiumHistogramEnumValue(ctx, WebFeatureChromiumHistogramEnumValue{
		WebFeatureID:                 featureIDMap["feature1"],
		ChromiumHistogramEnumValueID: enumIDMap["CompressionStreams"],
	})
	if err != nil {
		t.Fatalf("failed to insert WebFeatureChromiumHistogramEnum. err: %s", err)
	}
	err = c.UpsertWebFeatureChromiumHistogramEnumValue(ctx, WebFeatureChromiumHistogramEnumValue{
		WebFeatureID:                 featureIDMap["feature2"],
		ChromiumHistogramEnumValueID: enumIDMap["ViewTransitions"],
	})
	if err != nil {
		t.Fatalf("failed to insert WebFeatureChromiumHistogramEnum. err: %s", err)
	}
}

func insertGivenWebFeatureChromiumHistogramEnumValues(
	ctx context.Context,
	client *Client,
	t *testing.T,
	values []WebFeatureChromiumHistogramEnumValue,
) {
	for _, webFeatureChromiumHistogramEnumValue := range values {
		err := client.UpsertWebFeatureChromiumHistogramEnumValue(
			ctx,
			webFeatureChromiumHistogramEnumValue,
		)
		if err != nil {
			t.Errorf("unexpected error during insert of Chromium enums. %s", err.Error())
		}
	}
}

func (c *Client) readAllWebFeatureChromiumHistogramEnums(
	ctx context.Context, _ *testing.T) ([]WebFeatureChromiumHistogramEnumValue, error) {
	stmt := spanner.NewStatement(
		`SELECT
			WebFeatureID, ChromiumHistogramEnumValueID
		FROM WebFeatureChromiumHistogramEnumValues`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []WebFeatureChromiumHistogramEnumValue
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
		ret = append(ret, chromiumHistogramEnum.WebFeatureChromiumHistogramEnumValue)
	}

	return ret, nil
}

func sortWebFeatureChromiumHistogramEnums(left, right WebFeatureChromiumHistogramEnumValue) int {
	return cmp.Compare(left.WebFeatureID, right.WebFeatureID)
}

func webFeatureChromiumHistogramEnumEquality(left, right WebFeatureChromiumHistogramEnumValue) bool {
	return left.WebFeatureID == right.WebFeatureID &&
		left.ChromiumHistogramEnumValueID == right.ChromiumHistogramEnumValueID
}

func TestUpsertWebFeatureChromiumHistogramEnumValue(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	idMap := setupRequiredTablesForWebFeatureChromiumHistogramEnum(ctx, t)
	enumIDMap := insertSampleChromiumHistogramEnums(ctx, t, spannerClient)
	enumValueLabelToIDMap := insertSampleChromiumHistogramEnumValues(ctx, t, spannerClient, enumIDMap)
	spannerClient.createSampleWebFeatureChromiumHistogramEnums(ctx, t, idMap, enumValueLabelToIDMap)

	expected := []WebFeatureChromiumHistogramEnumValue{
		{
			WebFeatureID:                 idMap["feature1"],
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["CompressionStreams"],
		},
		{
			WebFeatureID:                 idMap["feature2"],
			ChromiumHistogramEnumValueID: enumValueLabelToIDMap["ViewTransitions"],
		},
	}
	slices.SortFunc(expected, sortWebFeatureChromiumHistogramEnums)

	chromiumHistogramEnums, err := spannerClient.readAllWebFeatureChromiumHistogramEnums(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all ChromiumHistogramEnumValues err: %s", err)
	}
	slices.SortFunc(chromiumHistogramEnums, sortWebFeatureChromiumHistogramEnums)

	if !slices.EqualFunc(expected, chromiumHistogramEnums, webFeatureChromiumHistogramEnumEquality) {
		t.Errorf("unequal ChromiumHistogramEnumValues.\nexpected %+v\nreceived %+v", expected, chromiumHistogramEnums)
	}

	// Upsert WebFeatureChromiumHistogramEnum
	err = spannerClient.UpsertWebFeatureChromiumHistogramEnumValue(ctx, WebFeatureChromiumHistogramEnumValue{
		WebFeatureID:                 idMap["feature2"],
		ChromiumHistogramEnumValueID: enumValueLabelToIDMap["ViewTransitions"],
	})
	if err != nil {
		t.Fatalf("unable to update ChromiumHistogramEnum err: %s", err)
	}

	// Should be the same.
	slices.SortFunc(expected, sortWebFeatureChromiumHistogramEnums)

	chromiumHistogramEnums, err = spannerClient.readAllWebFeatureChromiumHistogramEnums(ctx, t)
	if err != nil {
		t.Fatalf("unable to get all ChromiumHistogramEnumValues err: %s", err)
	}
	slices.SortFunc(chromiumHistogramEnums, sortWebFeatureChromiumHistogramEnums)

	if !slices.EqualFunc(expected, chromiumHistogramEnums, webFeatureChromiumHistogramEnumEquality) {
		t.Errorf("unequal ChromiumHistogramEnumValues.\nexpected %+v\nreceived %+v", expected, chromiumHistogramEnums)
	}
}
