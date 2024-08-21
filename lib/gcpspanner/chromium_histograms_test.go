package gcpspanner

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func getSampleChromiumHistograms() []ChromiumHistogramEnum {
	return []ChromiumHistogramEnum{
		{
			HistogramName: "AnotherHistogram",
			BucketID:      1,
			Label:         "CompressionStreams",
		},
		{
			HistogramName: "WebDXFeatureObserver",
			BucketID:      1,
			Label:         "CompressionStreams",
		},
		{
			HistogramName: "WebDXFeatureObserver",
			BucketID:      2,
			Label:         "ViewTransitions",
		},
	}
}

func testEnumKey(histogramName string, bucketID int64) string {
	return fmt.Sprintf("%s-%d", histogramName, bucketID)
}

func insertSampleChromiumHistograms(ctx context.Context, t *testing.T, c *Client) map[string]string {
	enums := getSampleChromiumHistograms()
	m := make(map[string]string, len(enums))
	for _, enum := range enums {
		id, err := c.UpsertChromiumHistogramEnum(ctx, enum)
		if err != nil {
			t.Fatalf("unable to insert sample histograms. error %s", err)
		}
		m[testEnumKey(enum.HistogramName, enum.BucketID)] = *id
	}

	return m
}

// Helper method to get all the enums in a stable order.
func (c *Client) ReadAllChromiumHistogramEnums(ctx context.Context, t *testing.T) ([]ChromiumHistogramEnum, error) {
	stmt := spanner.NewStatement(
		`SELECT
			ID, HistogramName, BucketID, Label
		FROM ChromiumHistogramEnums
		ORDER BY HistogramName ASC, BucketID ASC`)
	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	var ret []ChromiumHistogramEnum
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break // End of results
		}
		if err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var enum spannerChromiumHistogramEnum
		if err := row.ToStruct(&enum); err != nil {
			return nil, errors.Join(ErrInternalQueryFailure, err)
		}
		if enum.ID == "" {
			t.Error("retrieved enum ID is empty")
		}
		ret = append(ret, enum.ChromiumHistogramEnum)
	}

	return ret, nil
}

func TestChromiumHistogramEnum(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()
	insertSampleChromiumHistograms(ctx, t, spannerClient)
	enums, err := spannerClient.ReadAllChromiumHistogramEnums(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}
	sampleHistogramsEnums := getSampleChromiumHistograms()
	if !slices.Equal[[]ChromiumHistogramEnum](getSampleChromiumHistograms(), enums) {
		t.Errorf("unequal enums. expected %+v actual %+v", sampleHistogramsEnums, enums)
	}

	_, err = spannerClient.UpsertChromiumHistogramEnum(ctx, ChromiumHistogramEnum{
		HistogramName: "WebDXFeatureObserver",
		BucketID:      1,
		// Should not update
		Label: "CompressionStreamssssssss",
	})
	if err != nil {
		t.Errorf("unexpected error during update. %s", err.Error())
	}

	enums, err = spannerClient.ReadAllChromiumHistogramEnums(ctx, t)
	if err != nil {
		t.Errorf("unexpected error during read all. %s", err.Error())
	}

	// Should be the same. No updates should happen.
	if !slices.Equal[[]ChromiumHistogramEnum](sampleHistogramsEnums, enums) {
		t.Errorf("unequal enums after update. expected %+v actual %+v", sampleHistogramsEnums, enums)
	}
}
