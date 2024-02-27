package gcpspanner

import (
	"context"
	"errors"

	"google.golang.org/api/iterator"
)

type SpannerFeatureResult struct {
	ID string `spanner:"ID"`
	FeatureResult
}

// type FeatureResult struct {
// 	FeatureID           string                `spanner:"FeatureID"`
// 	Name                string                `spanner:"Name"`
// 	Status              BaselineStatus        `spanner:"Status"`
// 	StableMetrics       []WPTRunFeatureMetric `spanner:"StableMetrics"`
// 	ExperimentalMetrics []WPTRunFeatureMetric `spanner:"ExperimentalMetrics"`
// }

type Metric struct {
	BrowserName string `json:"BrowserName"`
	TotalTests  int64  `json:"TotalTests"`
	TestPass    int64  `json:"TestPass"`
}

type FeatureResult struct {
	FeatureID           string    `json:"FeatureID"`
	Name                string    `json:"Name"`
	Status              string    `json:"Status"`
	StableMetrics       []*Metric `json:"StableMetrics"`
	ExperimentalMetrics []*Metric `json:"ExperimentalMetrics"`
}

func (c *Client) FeaturesSearch(ctx context.Context, pageToken *string, pageSize int, filterables ...Filterable) ([]FeatureResult, *string, error) {
	b := FeatureSearchQueryBuilder{
		cursorID: pageToken,
		pageSize: pageSize,
	}
	stmt := b.Build(filterables...)
	txn := c.Single()
	defer txn.Close()
	it := txn.Query(ctx, stmt)
	defer it.Stop()

	var results []FeatureResult
	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, errors.Join(ErrInternalQueryFailure, err)
		}
		var result SpannerFeatureResult
		if err := row.ToStruct(&result); err != nil {
			return nil, nil, err
		}
		results = append(results, result.FeatureResult)
	}

	if len(results) == pageSize {
		lastResult := results[len(results)-1]
		newCursor := encodeFeatureResultCursor(lastResult.FeatureID)

		return results, &newCursor, nil
	}

	return results, nil, nil
}
