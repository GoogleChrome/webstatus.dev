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
	"fmt"
	"log/slog"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

const webFeaturesTable = "WebFeatures"
const systemAuthorID = "system"

func systemSavedSearchName(featureKey string) string {
	return fmt.Sprintf("Feature %s", featureKey)
}

func systemSavedSearchQuery(featureKey string) string {
	return fmt.Sprintf("id:\"%s\"", featureKey)
}

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

func (m webFeatureSpannerMapper) createSystemManagedSavedSearchMutations(
	id string,
	entity WebFeature,
) ([]*spanner.Mutation, error) {
	description := fmt.Sprintf("A system-managed saved search for the feature %s", entity.Name)
	savedSearchID := uuid.NewString()
	savedSearch := SavedSearch{
		ID:          savedSearchID,
		Name:        systemSavedSearchName(entity.FeatureKey),
		Query:       systemSavedSearchQuery(entity.FeatureKey),
		Description: &description,
		AuthorID:    systemAuthorID,
		Scope:       SystemManagedScope,
		CreatedAt:   spanner.CommitTimestamp,
		UpdatedAt:   spanner.CommitTimestamp,
	}
	savedSearchMutation, err := spanner.InsertStruct(savedSearchesTable, &savedSearch)
	if err != nil {
		return nil, err
	}

	// Create the system-managed saved search.
	systemManagedSearch := SystemManagedSavedSearch{
		FeatureID:     id,
		SavedSearchID: savedSearchID,
		CreatedAt:     spanner.CommitTimestamp,
		UpdatedAt:     spanner.CommitTimestamp,
	}
	systemManagedSearchMutation, err := spanner.InsertStruct(systemManagedSavedSearchesTable, &systemManagedSearch)
	if err != nil {
		return nil, err
	}

	return []*spanner.Mutation{savedSearchMutation, systemManagedSearchMutation}, nil
}

func (m webFeatureSpannerMapper) PostWriteHook(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	client *Client,
	id string,
	entity WebFeature,
) ([]*spanner.Mutation, error) {
	// A new feature was added, create a system-managed saved search for it.
	_, err := client.getSystemManagedSavedSearchByFeatureIDAndTransaction(ctx, txn, id)
	if err == nil {
		// The system-managed saved search already exists.
		return nil, nil
	}

	if !errors.Is(err, ErrQueryReturnedNoResults) {
		return nil, err
	}

	return m.createSystemManagedSavedSearchMutations(id, entity)
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

func (m webFeatureSpannerMapper) NewEntity(id string, req WebFeature) (SpannerWebFeature, error) {
	return SpannerWebFeature{
		ID:         id,
		WebFeature: req,
	}, nil
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
			if errors.Is(err, ErrQueryReturnedNoResults) {
				slog.WarnContext(ctx,
					"source feature key not found during redirect, skipping",
					"sourceKey", sourceKey,
				)

				continue
			}
			slog.ErrorContext(ctx, "unable to get ID from feature key for source key",
				"sourceKey", sourceKey, "err", err)

			return nil, err
		}
		featureKeyToIDMap[sourceKey] = *sourceID

		targetID, err := c.GetIDFromFeatureKey(ctx, &FeatureIDFilter{featureKey: targetKey})
		if err != nil {
			slog.ErrorContext(ctx, "unable to get ID from feature key for target key",
				"targetKey", targetKey, "err", err)

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

func (m webFeatureSpannerMapper) moveLatestFeatureDeveloperSignals(
	ctx context.Context,
	c *Client,
	sourceID string,
	targetID string,
) ([]*spanner.Mutation, error) {
	developerSignals, err := c.getAllLatestFeatureDeveloperSignalsByWebFeatureID(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	mutations := make([]*spanner.Mutation, 0, len(developerSignals))
	for _, signal := range developerSignals {
		signal.WebFeatureID = targetID
		m, err := spanner.InsertOrUpdateStruct(latestFeatureDeveloperSignalsTableName, signal)
		if err != nil {
			slog.ErrorContext(ctx, "unable to create mutation for LatestFeatureDeveloperSignals",
				"error", err, "signal", signal)

			return nil, err
		}
		mutations = append(mutations, m)
	}

	return mutations, nil
}

func (m webFeatureSpannerMapper) moveSystemManagedSavedSearch(
	ctx context.Context,
	c *Client,
	sourceID, targetKey string,
	savedSearchMutations *[]*spanner.Mutation,
	systemManagedSearchMutations *[]*spanner.Mutation,
) error {
	systemManagedSearch, err := c.GetSystemManagedSavedSearchByFeatureID(ctx, sourceID)
	if err != nil {
		if errors.Is(err, ErrQueryReturnedNoResults) {
			slog.WarnContext(ctx, "system managed saved search not found during redirect, skipping", "sourceID", sourceID)

			return nil // Not an error, just nothing to do.
		}

		return fmt.Errorf("unable to get system managed saved search: %w", err)
	}

	savedSearch, err := c.GetSavedSearch(ctx, systemManagedSearch.SavedSearchID)
	if err != nil {
		return fmt.Errorf("unable to get saved search: %w", err)
	}

	savedSearch.Name = systemSavedSearchName(targetKey)
	savedSearch.Query = systemSavedSearchQuery(targetKey)
	savedSearch.UpdatedAt = spanner.CommitTimestamp

	updateMutation, err := spanner.UpdateStruct(savedSearchesTable, savedSearch)
	if err != nil {
		return fmt.Errorf("unable to create update mutation for saved search: %w", err)
	}
	*savedSearchMutations = append(*savedSearchMutations, updateMutation)

	// Now update the system-managed saved search association
	deleteSubMutation := spanner.Delete(systemManagedSavedSearchesTable, spanner.Key{sourceID})
	newSystemManagedSearch := SystemManagedSavedSearch{
		FeatureID:     targetKey, // The target ID (feature key) is the new FeatureID
		SavedSearchID: systemManagedSearch.SavedSearchID,
		CreatedAt:     spanner.CommitTimestamp,
		UpdatedAt:     spanner.CommitTimestamp,
	}
	insertSubMutation, err := spanner.InsertStruct(systemManagedSavedSearchesTable, &newSystemManagedSearch)
	if err != nil {
		return fmt.Errorf("unable to create insert mutation for system managed search: %w", err)
	}
	*systemManagedSearchMutations = append(*systemManagedSearchMutations, deleteSubMutation, insertSubMutation)

	return nil
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
		slog.ErrorContext(ctx, "unable to build feature key to ID map", "err", err)

		return nil, err
	}

	var wptRunFeatureMetricMutations []*spanner.Mutation
	var latestWPTRunFeatureMetricMutations []*spanner.Mutation
	var webFeatureChromiumHistogramEnumValueMutations []*spanner.Mutation
	var latestDailyChromiumHistogramMetricMutations []*spanner.Mutation
	var latestFeatureDeveloperSignalMutations []*spanner.Mutation
	var savedSearchMutations []*spanner.Mutation
	var systemManagedSearchMutations []*spanner.Mutation

	// The following sections are where the WebFeatureID is the primary key (or part of the primary key).
	// This requires us to copy the rows (with updated IDs) because Spanner does not allow the modifications of keys.
	// https://cloud.google.com/spanner/docs/schema-and-data-model#change_table_keys
	for sourceKey, targetKey := range m.RedirectTargets {
		sourceID := featureKeyToIDMap[sourceKey]
		targetID := featureKeyToIDMap[targetKey]

		mutations, err := m.moveWPTRunFeatureMetrics(ctx, c, sourceID, targetID)
		if err != nil {
			slog.ErrorContext(ctx, "unable to move wpt run feature metrics",
				"sourceID", sourceID, "targetID", targetID, "err", err)

			return nil, err
		}
		wptRunFeatureMetricMutations = append(wptRunFeatureMetricMutations, mutations...)

		mutations, err = m.moveLatestWPTRunFeatureMetrics(ctx, c, sourceID, targetID)
		if err != nil {
			slog.ErrorContext(ctx, "unable to move latest wpt run feature metrics",
				"sourceID", sourceID, "targetID", targetID, "err", err)

			return nil, err
		}
		latestWPTRunFeatureMetricMutations = append(latestWPTRunFeatureMetricMutations, mutations...)

		mutations, err = m.moveWebFeatureChromiumHistogramEnumValues(ctx, c, sourceID, targetID)
		if err != nil {
			slog.ErrorContext(ctx, "unable to move chromium histogram enum value",
				"sourceID", sourceID, "targetID", targetID, "err", err)

			return nil, err
		}
		webFeatureChromiumHistogramEnumValueMutations = append(
			webFeatureChromiumHistogramEnumValueMutations, mutations...)

		mutations, err = m.moveLatestDailyChromiumHistogramMetrics(ctx, c, sourceID, targetID)
		if err != nil {
			slog.ErrorContext(ctx, "unable to move latest chromium histogram metric",
				"sourceID", sourceID, "targetID", targetID, "err", err)

			return nil, err
		}
		latestDailyChromiumHistogramMetricMutations = append(latestDailyChromiumHistogramMetricMutations, mutations...)

		mutations, err = m.moveLatestFeatureDeveloperSignals(ctx, c, sourceID, targetID)
		if err != nil {
			slog.ErrorContext(ctx, "unable to move latest feature developer signals",
				"sourceID", sourceID, "targetID", targetID, "err", err)

			return nil, err
		}
		latestFeatureDeveloperSignalMutations = append(latestFeatureDeveloperSignalMutations, mutations...)

		// Now move the SystemManagedSavedSearch
		if err := m.moveSystemManagedSavedSearch(
			ctx, c, sourceID, targetKey, &savedSearchMutations, &systemManagedSearchMutations); err != nil {
			slog.ErrorContext(ctx, "failed to move system managed saved search", "error", err, "sourceID", sourceID)

			return nil, err
		}
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

	if len(latestFeatureDeveloperSignalMutations) > 0 {
		groups = append(groups, ExtraMutationsGroup{
			tableName: latestFeatureDeveloperSignalsTableName,
			mutations: latestFeatureDeveloperSignalMutations,
		})
	}

	if len(savedSearchMutations) > 0 {
		groups = append(groups, ExtraMutationsGroup{
			tableName: savedSearchesTable,
			mutations: savedSearchMutations,
		})
	}

	if len(systemManagedSearchMutations) > 0 {
		groups = append(groups, ExtraMutationsGroup{
			tableName: systemManagedSavedSearchesTable,
			mutations: systemManagedSearchMutations,
		})
	}

	return groups, nil
}

func (m webFeatureSpannerMapper) GetChildDeleteKeyMutations(
	ctx context.Context, client *Client, parentsToDelete []SpannerWebFeature) ([]ExtraMutationsGroup, error) {
	if len(parentsToDelete) == 0 {
		return nil, nil
	}
	metricMutations := make([]*spanner.Mutation, 0)
	browserSupportEventMutations := make([]*spanner.Mutation, 0)
	savedSearchMutations := make([]*spanner.Mutation, 0)

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

	// SystemManagedSavedSearches and SavedSearches
	parentIDs := make([]string, len(parentsToDelete))
	for i, parent := range parentsToDelete {
		parentIDs[i] = parent.ID
	}

	systemManagedSearches, err := client.ListSystemManagedSavedSearchesByFeatureIDs(ctx, parentIDs)
	if err != nil {
		return nil, err
	}

	for _, sms := range systemManagedSearches {
		// Delete the system-managed saved search
		savedSearchMutations = append(savedSearchMutations,
			spanner.Delete(systemManagedSavedSearchesTable, spanner.Key{sms.FeatureID}))

		// Delete the saved search
		savedSearchMutations = append(savedSearchMutations,
			spanner.Delete(savedSearchesTable, spanner.Key{sms.SavedSearchID}))
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
		{
			tableName: savedSearchesTable,
			mutations: savedSearchMutations,
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

func (m webFeatureSpannerMapper) GetIDFromInternal(s SpannerWebFeature) string {
	return s.ID
}

func (m webFeatureSpannerMapper) NewEntityWithID(req WebFeature) (SpannerWebFeature, string, error) {
	id := uuid.NewString()

	return SpannerWebFeature{
		ID:         id,
		WebFeature: req,
	}, id, nil
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

func (c *Client) GetIDFromFeatureKey(ctx context.Context, filter *FeatureIDFilter) (*string, error) {
	return newEntityWriterWithIDRetrievalAndHooks[
		webFeatureSpannerMapper, string, WebFeature, SpannerWebFeature, string](c).
		getIDByKey(ctx, filter.featureKey)
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
