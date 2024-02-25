package gcpspanner

import (
	"context"
	"errors"
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func getSampleFeatures() []WebFeature {
	return []WebFeature{
		{
			Name:      "Feature 1",
			FeatureID: "feature1",
		},
		{
			Name:      "Feature 2",
			FeatureID: "feature2",
		},
		{
			Name:      "Feature 3",
			FeatureID: "feature3",
		},
		{
			Name:      "Feature 4",
			FeatureID: "feature4",
		},
	}
}

// Helper method to get all the features in a stable order.
func (c *Client) ReadAllWebFeatures(ctx context.Context, t *testing.T) ([]WebFeature, error) {
	stmt := spanner.NewStatement("SELECT ID, FeatureID, Name FROM WebFeatures ORDER BY FeatureID ASC")
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []WebFeature
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var feature SpannerWebFeature
		if err := row.ToStruct(&feature); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		if feature.ID == "" {
			t.Error("retrieved feature ID is empty")
		}
		ret = append(ret, feature.WebFeature)
	}

	return ret, nil
}

func TestUpsertWebFeature(t *testing.T) {
	client := getTestDatabase(t)
	ctx := context.Background()
	sampleFeatures := getSampleFeatures()
	for _, feature := range sampleFeatures {
		err := client.UpsertWebFeature(ctx, feature)
		if err != nil {
			t.Errorf("unexpected error during insert. %s", err.Error())
		}
	}
	features, err := client.ReadAllWebFeatures(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	if !slices.Equal[[]WebFeature](sampleFeatures, features) {
		t.Errorf("unequal features. expected %+v actual %+v", sampleFeatures, features)
	}

	err = client.UpsertWebFeature(ctx, WebFeature{
		Name:      "Feature 1!!",
		FeatureID: "feature1",
	})
	if err != nil {
		t.Errorf("unexpected error during update. %s", err.Error())
	}

	features, err = client.ReadAllWebFeatures(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}

	expectedPageAfterUpdate := []WebFeature{
		{
			Name:      "Feature 1!!", // Updated field
			FeatureID: "feature1",
		},
		{
			Name:      "Feature 2",
			FeatureID: "feature2",
		},
		{
			Name:      "Feature 3",
			FeatureID: "feature3",
		},
		{
			Name:      "Feature 4",
			FeatureID: "feature4",
		},
	}
	if !slices.Equal[[]WebFeature](expectedPageAfterUpdate, features) {
		t.Errorf("unequal features after update. expected %+v actual %+v", sampleFeatures, features)
	}
}
