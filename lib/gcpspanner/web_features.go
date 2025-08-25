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
	"fmt"
	"log/slog"

	"cloud.google.com/go/spanner"
)

const webFeaturesTable = "WebFeatures"

// SpannerWebFeature is a wrapper for the feature that is actually
// stored in spanner. This is useful because the spanner id is not useful to
// return to the end user since it is only used to decouple the primary keys
// between this system and web features repo.
type SpannerWebFeature struct {
	ID string `spanner:"ID"`
	WebFeature
}

// WebFeature contains common metadata for a Web Feature.
// Columns come from the ../../infra/storage/spanner/migrations/*.sql files.
type WebFeature struct {
	FeatureKey      string `spanner:"FeatureKey"`
	Name            string `spanner:"Name"`
	Description     string `spanner:"Description"`
	DescriptionHTML string `spanner:"DescriptionHtml"`
}

// Implements the syncableEntityMapper interface for WebFeature and SpannerWebFeature.
type webFeatureSpannerMapper struct {
	// key is original feature key. value is the target feature key.
	RedirectTargets map[string]string
}

// SelectAll returns a statement to select all WebFeatures.
func (m webFeatureSpannerMapper) SelectAll() spanner.Statement {
	return spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID, FeatureKey, Name, Description, DescriptionHtml
	FROM %s`, m.Table()))
}

func (m webFeatureSpannerMapper) SelectOne(key string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID, FeatureKey, Name, Description, DescriptionHtml
	FROM %s
	WHERE FeatureKey = @featureKey
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"featureKey": key,
	}
	stmt.Params = parameters

	return stmt
}

// Merge method remains for backward compatibility.
// TODO: Remove once we remove the UpsertWebFeature method.
func (m webFeatureSpannerMapper) Merge(in WebFeature, existing SpannerWebFeature) SpannerWebFeature {
	merged, _ := m.MergeAndCheckChanged(in, existing)

	return merged
}

// MergeAndCheckChanged will merge the entity and return if the entity has changed.
func (m webFeatureSpannerMapper) MergeAndCheckChanged(
	in WebFeature, existing SpannerWebFeature) (SpannerWebFeature, bool) {
	merged := SpannerWebFeature{
		ID: existing.ID,
		WebFeature: WebFeature{
			FeatureKey:      existing.FeatureKey,
			Name:            cmp.Or(in.Name, existing.Name),
			Description:     cmp.Or(in.Description, existing.Description),
			DescriptionHTML: cmp.Or(in.DescriptionHTML, existing.DescriptionHTML),
		},
	}

	hasChanged := merged.Name != existing.Name ||
		merged.Description != existing.Description ||
		merged.DescriptionHTML != existing.DescriptionHTML

	return merged, hasChanged
}

func (m webFeatureSpannerMapper) buildFeatureKeyToIDMap(ctx context.Context, c *Client) (map[string]string, error) {
	featureKeyToIDMap := map[string]string{}
	for sourceKey, targetKey := range m.RedirectTargets {
		sourceID, err := c.GetIDFromFeatureKey(ctx, &FeatureIDFilter{featureKey: sourceKey})
		if err != nil {
			return nil, err
		}
		featureKeyToIDMap[sourceKey] = *sourceID

		targetID, err := c.GetIDFromFeatureKey(ctx, &FeatureIDFilter{featureKey: targetKey})
		if err != nil {
			return nil, err
		}
		featureKeyToIDMap[targetKey] = *targetID
	}

	return featureKeyToIDMap, nil
}

func (m webFeatureSpannerMapper) moveWPTRunFeatureMetrics(
	ctx context.Context,
	c *Client,
	sourceID string,
	targetID string,
) ([]*spanner.Mutation, error) {
	metrics, err := c.getAllWPTRunFeatureMetricIDsByWebFeatureID(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	mutations := make([]*spanner.Mutation, 0, len(metrics))
	for _, metric := range metrics {
		metric.WebFeatureID = targetID
		m, err := spanner.InsertOrUpdateStruct(WPTRunFeatureMetricTable, metric)
		if err != nil {
			slog.ErrorContext(ctx, "unable to create mutation for WPTRunFeatureMetric", "error", err, "metric", metric)

			return nil, err
		}
		mutations = append(mutations, m)
	}

	return mutations, nil
}

func (m webFeatureSpannerMapper) moveLatestWPTRunFeatureMetrics(
	ctx context.Context,
	c *Client,
	sourceID string,
	targetID string,
) ([]*spanner.Mutation, error) {
	latestMetrics, err := c.getAllSpannerLatestWPTRunFeatureMetricIDsByWebFeatureID(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	mutations := make([]*spanner.Mutation, 0, len(latestMetrics))
	for _, metric := range latestMetrics {
		metric.WebFeatureID = targetID
		m, err := spanner.InsertOrUpdateStruct(LatestWPTRunFeatureMetricsTable, metric)
		if err != nil {
			slog.ErrorContext(ctx, "unable to create mutation for LatestWPTRunFeatureMetrics",
				"error", err, "metric", metric)

			return nil, err
		}
		mutations = append(mutations, m)
	}

	return mutations, nil
}

func (m webFeatureSpannerMapper) moveWebFeatureChromiumHistogramEnumValues(
	ctx context.Context,
	c *Client,
	sourceID string,
	targetID string,
) ([]*spanner.Mutation, error) {
	featureEnumValues, err := c.getAllWebFeatureChromiumHistogramEnumValuesByFeatureID(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	mutations := make([]*spanner.Mutation, 0, len(featureEnumValues))
	for _, featureEnumValue := range featureEnumValues {
		featureEnumValue.WebFeatureID = targetID
		m, err := spanner.InsertOrUpdateStruct(webFeatureChromiumHistogramEnumValuesTable, featureEnumValue)
		if err != nil {
			slog.ErrorContext(ctx, "unable to create mutation for WebFeatureChromiumHistogramEnumValues",
				"error", err, "featureEnumValue", featureEnumValue)

			return nil, err
		}
		mutations = append(mutations, m)
	}

	return mutations, nil
}

func (m webFeatureSpannerMapper) moveLatestDailyChromiumHistogramMetrics(
	ctx context.Context,
	c *Client,
	sourceID string,
	targetID string,
) ([]*spanner.Mutation, error) {
	dailyMetrics, err := c.getAllLatestDailyChromiumHistogramMetricsByFeatureID(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	mutations := make([]*spanner.Mutation, 0, len(dailyMetrics))
	for _, metric := range dailyMetrics {
		metric.WebFeatureID = targetID
		m, err := spanner.InsertOrUpdateStruct(LatestDailyChromiumHistogramMetricsTable, metric)
		if err != nil {
			slog.ErrorContext(ctx, "unable to create mutation for LatestDailyChromiumHistogramMetrics",
				"error", err, "metric", metric)

			return nil, err
		}
		mutations = append(mutations, m)
	}

	return mutations, nil
}

func (m webFeatureSpannerMapper) PreDeleteHook(
	ctx context.Context,
	c *Client,
	_ []SpannerWebFeature,
) ([]ExtraMutationsGroup, error) {
	// Check the m.RedirectTargets and move data sources to prevent data loss.
	if len(m.RedirectTargets) == 0 {
		return nil, nil
	}

	featureKeyToIDMap, err := m.buildFeatureKeyToIDMap(ctx, c)
	if err != nil {
		return nil, err
	}

	var wptRunFeatureMetricMutations []*spanner.Mutation
	var latestWPTRunFeatureMetricMutations []*spanner.Mutation
	var webFeatureChromiumHistogramEnumValueMutations []*spanner.Mutation
	var latestDailyChromiumHistogramMetricMutations []*spanner.Mutation

	// The following sections are where the WebFeatureID is the primary key (or part of the primary key).
	// This requires us to copy the rows (with updated IDs) because Spanner does not allow the modifications of keys.
	// https://cloud.google.com/spanner/docs/schema-and-data-model#change_table_keys
	for sourceKey, targetKey := range m.RedirectTargets {
		sourceID := featureKeyToIDMap[sourceKey]
		targetID := featureKeyToIDMap[targetKey]

		mutations, err := m.moveWPTRunFeatureMetrics(ctx, c, sourceID, targetID)
		if err != nil {
			return nil, err
		}
		wptRunFeatureMetricMutations = append(wptRunFeatureMetricMutations, mutations...)

		mutations, err = m.moveLatestWPTRunFeatureMetrics(ctx, c, sourceID, targetID)
		if err != nil {
			return nil, err
		}
		latestWPTRunFeatureMetricMutations = append(latestWPTRunFeatureMetricMutations, mutations...)

		mutations, err = m.moveWebFeatureChromiumHistogramEnumValues(ctx, c, sourceID, targetID)
		if err != nil {
			return nil, err
		}
		webFeatureChromiumHistogramEnumValueMutations = append(
			webFeatureChromiumHistogramEnumValueMutations, mutations...)

		mutations, err = m.moveLatestDailyChromiumHistogramMetrics(ctx, c, sourceID, targetID)
		if err != nil {
			return nil, err
		}
		latestDailyChromiumHistogramMetricMutations = append(latestDailyChromiumHistogramMetricMutations, mutations...)
	}

	var groups []ExtraMutationsGroup
	if len(wptRunFeatureMetricMutations) > 0 {
		groups = append(groups, ExtraMutationsGroup{
			tableName: WPTRunFeatureMetricTable,
			mutations: wptRunFeatureMetricMutations,
		})
	}

	if len(latestWPTRunFeatureMetricMutations) > 0 {
		groups = append(groups, ExtraMutationsGroup{
			tableName: LatestWPTRunFeatureMetricsTable,
			mutations: latestWPTRunFeatureMetricMutations,
		})
	}

	if len(webFeatureChromiumHistogramEnumValueMutations) > 0 {
		groups = append(groups, ExtraMutationsGroup{
			tableName: webFeatureChromiumHistogramEnumValuesTable,
			mutations: webFeatureChromiumHistogramEnumValueMutations,
		})
	}

	if len(latestDailyChromiumHistogramMetricMutations) > 0 {
		groups = append(groups, ExtraMutationsGroup{
			tableName: LatestDailyChromiumHistogramMetricsTable,
			mutations: latestDailyChromiumHistogramMetricMutations,
		})
	}

	return groups, nil
}

func (m webFeatureSpannerMapper) GetChildDeleteKeyMutations(
	ctx context.Context, client *Client, parentsToDelete []SpannerWebFeature) ([]ExtraMutationsGroup, error) {
	if len(parentsToDelete) == 0 {
		return nil, nil
	}
	var metricMutations, browserSupportEventMutations []*spanner.Mutation

	// WPTRunFeatureMetrics can contain a lot of entries for a given feature
	for _, parent := range parentsToDelete {
		pairs, err := client.getAllSpannerWPTRunFeatureMetricIDsByWebFeatureID(ctx, parent.ID)
		if err != nil {
			return nil, err
		}
		for _, pair := range pairs {
			metricMutations = append(metricMutations,
				spanner.Delete(WPTRunFeatureMetricTable, spanner.Key{pair.ID, pair.WebFeatureID}))
		}
	}

	// BrowserFeatureCounts can contain a lot of entries for a given feature
	for _, parent := range parentsToDelete {
		events, err := client.getAllSpannerBrowserFeatureCountIDsByWebFeatureID(ctx, parent.ID)
		if err != nil {
			return nil, err
		}
		for _, event := range events {
			browserSupportEventMutations = append(browserSupportEventMutations,
				spanner.Delete(browserFeatureSupportEventsTable,
					spanner.Key{
						event.TargetBrowserName,
						event.EventBrowserName,
						event.EventReleaseDate,
						event.WebFeatureID,
					}))
		}
	}

	return []ExtraMutationsGroup{
		{
			tableName: WPTRunFeatureMetricTable,
			mutations: metricMutations,
		},
		{
			tableName: browserFeatureSupportEventsTable,
			mutations: browserSupportEventMutations,
		},
	}, nil
}

// DeleteMutation creates a Spanner delete mutation for a given WebFeature.
// It uses the internal Spanner ID, not the FeatureKey, for the deletion.
func (m webFeatureSpannerMapper) DeleteMutation(in SpannerWebFeature) *spanner.Mutation {
	return spanner.Delete(webFeaturesTable, spanner.Key{in.ID})
}

// Table returns the name of the Spanner table.
func (m webFeatureSpannerMapper) Table() string {
	return webFeaturesTable
}

// GetKeyFromExternal returns the business key (FeatureKey) from an external WebFeature struct.
func (m webFeatureSpannerMapper) GetKeyFromExternal(in WebFeature) string {
	return in.FeatureKey
}

// GetKeyFromInternal returns the business key (FeatureKey) from an internal SpannerWebFeature struct.
func (m webFeatureSpannerMapper) GetKeyFromInternal(in SpannerWebFeature) string {
	return in.FeatureKey
}

func (m webFeatureSpannerMapper) GetID(key string) spanner.Statement {
	stmt := spanner.NewStatement(fmt.Sprintf(`
	SELECT
		ID
	FROM %s
	WHERE FeatureKey = @featureKey
	LIMIT 1`, m.Table()))
	parameters := map[string]interface{}{
		"featureKey": key,
	}
	stmt.Params = parameters

	return stmt
}

type SyncWebFeaturesOption func(*webFeatureSpannerMapper)

func WithRedirectTargets(redirects map[string]string) SyncWebFeaturesOption {
	return func(m *webFeatureSpannerMapper) {
		m.RedirectTargets = redirects
	}
}

// SyncWebFeatures reconciles the WebFeatures table with the provided list of features.
// It will insert new features, update existing ones, and delete any features
// that are in the database but not in the provided list.
func (c *Client) SyncWebFeatures(
	ctx context.Context,
	features []WebFeature,
	opts ...SyncWebFeaturesOption,
) error {
	slog.InfoContext(ctx, "Starting web features synchronization")
	synchronizer := newEntitySynchronizer[webFeatureSpannerMapper](c)
	for _, opt := range opts {
		opt(&synchronizer.Mapper)
	}

	return synchronizer.Sync(ctx, features)
}

func (c *Client) UpsertWebFeature(ctx context.Context, feature WebFeature) (*string, error) {
	return newEntityWriterWithIDRetrieval[webFeatureSpannerMapper, string](c).upsertAndGetID(ctx, feature)
}

func (c *Client) GetIDFromFeatureKey(ctx context.Context, filter *FeatureIDFilter) (*string, error) {
	return newEntityWriterWithIDRetrieval[webFeatureSpannerMapper, string](c).getIDByKey(ctx, filter.featureKey)
}

func (c *Client) GetWebFeatureByID(ctx context.Context, id string) (*SpannerWebFeature, error) {
	return newEntityReader[webFeatureSpannerMapper, SpannerWebFeature, string](c).readRowByKey(ctx, id)
}

func (c *Client) fetchAllWebFeatureIDsWithTransaction(
	ctx context.Context, txn *spanner.ReadOnlyTransaction) ([]string, error) {
	return fetchSingleColumnValuesWithTransaction[string](ctx, txn, webFeaturesTable, "ID")
}

func (c *Client) FetchAllFeatureKeys(ctx context.Context) ([]string, error) {
	txn := c.ReadOnlyTransaction()
	defer txn.Close()

	return fetchSingleColumnValuesWithTransaction[string](ctx, txn, webFeaturesTable, "FeatureKey")
}

type SpannerFeatureIDAndKey struct {
	ID         string `spanner:"ID"`
	FeatureKey string `spanner:"FeatureKey"`
}

func (c *Client) FetchAllWebFeatureIDsAndKeys(ctx context.Context) ([]SpannerFeatureIDAndKey, error) {
	txn := c.ReadOnlyTransaction()
	defer txn.Close()

	return c.fetchAllWebFeatureIDsAndKeysWithTransaction(ctx, txn)
}

func (c *Client) fetchAllWebFeatureIDsAndKeysWithTransaction(
	ctx context.Context, txn *spanner.ReadOnlyTransaction) ([]SpannerFeatureIDAndKey, error) {
	return fetchColumnValuesWithTransaction[SpannerFeatureIDAndKey](
		ctx, txn, webFeaturesTable, []string{"ID", "FeatureKey"})
}

func fetchColumnValuesWithTransaction[T any](
	ctx context.Context, txn *spanner.ReadOnlyTransaction, table string, columnNames []string) ([]T, error) {
	var values []T
	iter := txn.Read(ctx, table, spanner.AllKeys(), columnNames)
	defer iter.Stop()
	err := iter.Do(func(row *spanner.Row) error {
		var value T
		if err := row.ToStruct(&value); err != nil {
			return err
		}
		values = append(values, value)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

// Deprecated. // TODO - use fetchColumnValuesWithTransaction.
func fetchSingleColumnValuesWithTransaction[T any](
	ctx context.Context, txn *spanner.ReadOnlyTransaction, table string, columnName string) ([]T, error) {
	var values []T
	iter := txn.Read(ctx, table, spanner.AllKeys(), []string{columnName})
	defer iter.Stop()
	err := iter.Do(func(row *spanner.Row) error {
		var value T
		if err := row.Column(0, &value); err != nil {
			return err
		}
		values = append(values, value)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}
