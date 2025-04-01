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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"sync"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/status"
)

// ErrQueryReturnedNoResults indicates no results were returned.
var ErrQueryReturnedNoResults = errors.New("query returned no results")

// ErrInternalQueryFailure is a catch-all error for now.
var ErrInternalQueryFailure = errors.New("internal spanner query failure")

// ErrBadClientConfig indicates the the config to setup a Client is invalid.
var ErrBadClientConfig = errors.New("projectID, instanceID and name must not be empty")

// ErrFailedToEstablishClient indicates the spanner client failed to create.
var ErrFailedToEstablishClient = errors.New("failed to establish spanner client")

// ErrInvalidCursorFormat indicates the cursor is not the correct format.
var ErrInvalidCursorFormat = errors.New("invalid cursor format")

// DailyChromiumHistogramMetrics specific errors.
var (
	// ErrUsageMetricUpsertNoFeatureIDFound indicates that no web feature ID was found
	// when attempting to upsert a usage metric using the chromiumHistogramEnumValueID.
	// This typically occurs when there is no corresponding web feature associated with
	// the given chromiumHistogramEnumValueID.
	ErrUsageMetricUpsertNoFeatureIDFound = errors.New("no web feature id found when upserting usage metric")

	// ErrUsageMetricUpsertNoHistogramFound indicates that the chromium histogram metric
	// was not found when attempting to upsert a usage metric.
	ErrUsageMetricUpsertNoHistogramFound = errors.New("histogram not found when upserting usage metric")

	// ErrUsageMetricUpsertNoHistogramEnumFound indicates that the chromium histogram enum
	// was not found when attempting to upsert a usage metric. This typically occurs when
	// the histogram name associated with the metric is not found, possibly due to
	// a draft or obsolete feature for which the corresponding enum ID has not been created.
	ErrUsageMetricUpsertNoHistogramEnumFound = errors.New("histogram enum not found when upserting usage metric")
)

// Client is the client for interacting with GCP Spanner.
type Client struct {
	*spanner.Client
	featureSearchQuery  FeatureSearchBaseQuery
	missingOneImplQuery MissingOneImplementationQuery
	searchCfg           searchConfig
	batchWriter
	batchSize    int
	batchWriters int
}

type batchWriter interface {
	BatchWriteMutations(context.Context, *spanner.Client, []*spanner.Mutation) error
}

// gcpBatchWriter is a batch writer for GCP environments using a real spanner database.
type gcpBatchWriter struct{}

func (w gcpBatchWriter) BatchWriteMutations(
	ctx context.Context, client *spanner.Client, mutations []*spanner.Mutation) error {
	it := client.BatchWrite(ctx, []*spanner.MutationGroup{
		{
			Mutations: mutations,
		},
	})

	return it.Do(func(r *spannerpb.BatchWriteResponse) error {
		if status := status.ErrorProto(r.GetStatus()); status != nil {
			slog.ErrorContext(ctx, "invalid status while batch writing", "status", status)

			return ErrInternalQueryFailure
		}

		return nil
	})
}

// LocalBatchWriter is a batch writer for local environments using the emulator.
// BatchWrite is not implemented in the emulator.
// https://github.com/GoogleCloudPlatform/cloud-spanner-emulator/issues/154
// Instead, do Apply which does multiple statements atomically.
// Remove this once the emulator supports BatchWrite.
// This is only exported for the load_fake_data utility.
type LocalBatchWriter struct{}

func (w LocalBatchWriter) BatchWriteMutations(
	ctx context.Context, client *spanner.Client, mutations []*spanner.Mutation) error {
	_, err := client.Apply(ctx, mutations)

	return err
}

// searchConfig holds the application configuation for the saved search feature.
type searchConfig struct {
	maxOwnedSearchesPerUser uint32
}

const defaultMaxOwnedSearchesPerUser = 25
const defaultBatchSize = 10000
const defaultBatchWriters = 8

func combineAndDeduplicate(excluded []string, discouraged []string) []string {
	if excluded == nil && discouraged == nil {
		return nil
	}

	if excluded == nil {
		return discouraged
	}

	if discouraged == nil {
		return excluded
	}

	totalLen := len(excluded) + len(discouraged)
	combined := make([]string, 0, totalLen)

	combined = append(combined, excluded...)
	combined = append(combined, discouraged...)

	slices.Sort(combined)
	combined = slices.Compact(combined)

	return combined
}

func (c *Client) getIgnoredFeatureIDsForStats(ctx context.Context, txn *spanner.ReadOnlyTransaction) ([]string, error) {
	excludedFeatureIDs, err := c.getFeatureIDsForEachExcludedFeatureKey(ctx, txn)
	if err != nil {
		return nil, err
	}

	discouragedFeatureIDs, err := c.getAllDiscouragedFeatureIDs(ctx, txn)
	if err != nil {
		return nil, err
	}

	return combineAndDeduplicate(excludedFeatureIDs, discouragedFeatureIDs), nil
}

// NewSpannerClient returns a Client for the Google Spanner service.
func NewSpannerClient(projectID string, instanceID string, name string) (*Client, error) {
	if projectID == "" || instanceID == "" || name == "" {
		return nil, ErrBadClientConfig
	}

	client, err := spanner.NewClient(
		context.TODO(),
		fmt.Sprintf(
			"projects/%s/instances/%s/databases/%s",
			projectID, instanceID, name))
	if err != nil {
		return nil, errors.Join(ErrFailedToEstablishClient, err)
	}

	var bw batchWriter
	bw = gcpBatchWriter{}
	if _, found := os.LookupEnv("SPANNER_EMULATOR_HOST"); found {
		slog.Info("using local batch writer")
		bw = LocalBatchWriter{}
	}

	return &Client{
		client,
		GCPFeatureSearchBaseQuery{},
		GCPMissingOneImplementationQuery{},
		searchConfig{maxOwnedSearchesPerUser: defaultMaxOwnedSearchesPerUser},
		bw,
		defaultBatchSize,
		defaultBatchWriters,
	}, nil
}

func (c *Client) SetFeatureSearchBaseQuery(query FeatureSearchBaseQuery) {
	c.featureSearchQuery = query
}

func (c *Client) SetMisingOneImplementationQuery(query MissingOneImplementationQuery) {
	c.missingOneImplQuery = query
}

// WPTRunCursor: Represents a point for resuming queries based on the last
// TimeStart and ExternalRunID. Useful for pagination.
type WPTRunCursor struct {
	LastTimeStart time.Time `json:"last_time_start"`
	LastRunID     int64     `json:"last_run_id"`
}

type ChromeDailyUsageCursor struct {
	LastDate civil.Date `json:"last_date"`
}

// FeatureResultOffsetCursor: A numerical offset from the start of the result set. Enables the construction of
// human-friendly URLs specifying an exact page offset.
// Disclaimer: External users should be aware that the format of this token is subject to change and should not be
// treated as a stable interface. Instead, external users should rely on the returned pagination token long term.
type FeatureResultOffsetCursor struct {
	Offset int `json:"offset"`
}

// decodeWPTRunCursor provides a wrapper around the generic decodeCursor.
func decodeWPTRunCursor(cursor string) (*WPTRunCursor, error) {
	return decodeCursor[WPTRunCursor](cursor)
}

// decodeInputFeatureResultCursor provides a wrapper around the generic decodeCursor.
func decodeInputFeatureResultCursor(
	cursor string) (*FeatureResultOffsetCursor, error) {
	// Try for the offset based cursor
	offsetCursor, err := decodeCursor[FeatureResultOffsetCursor](cursor)
	if err != nil {
		return nil, err
	}

	if offsetCursor == nil || offsetCursor.Offset < 0 {
		return nil, ErrInvalidCursorFormat
	}

	return offsetCursor, nil
}

// decodeChromeDailyUsageCursor provides a wrapper around the generic decodeCursor.
func decodeChromeDailyUsageCursor(
	cursor string) (*ChromeDailyUsageCursor, error) {
	return decodeCursor[ChromeDailyUsageCursor](cursor)
}

// decodeCursor: Decodes a base64-encoded cursor string into a Cursor struct.
func decodeCursor[T any](cursor string) (*T, error) {
	data, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, errors.Join(ErrInvalidCursorFormat, err)
	}
	var decoded T
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		return nil, errors.Join(ErrInvalidCursorFormat, err)
	}

	return &decoded, nil
}

// BrowserFeatureCountCursor: Represents a point for resuming feature count queries. Designed for efficient pagination
// by storing the following:
//   - LastReleaseDate: The release date of the last result from the previous page, used to continue fetching from the
//     correct point.
//   - LastCumulativeCount: The cumulative count of features up to (and including) the 'LastReleaseDate'.
//     This eliminates the need to recalculate the count for prior pages.
type BrowserFeatureCountCursor struct {
	LastReleaseDate     time.Time `json:"last_release_date"`
	LastCumulativeCount int64     `json:"last_cumulative_count"`
}

// decodeBrowserFeatureCountCursor provides a wrapper around the generic decodeCursor.
func decodeBrowserFeatureCountCursor(cursor string) (*BrowserFeatureCountCursor, error) {
	return decodeCursor[BrowserFeatureCountCursor](cursor)
}

// encodeBrowserFeatureCountCursor provides a wrapper around the generic encodeCursor.
func encodeBrowserFeatureCountCursor(releaseDate time.Time, lastCount int64) string {
	return encodeCursor[BrowserFeatureCountCursor](BrowserFeatureCountCursor{
		LastReleaseDate:     releaseDate,
		LastCumulativeCount: lastCount,
	})
}

// encodeWPTRunCursor provides a wrapper around the generic encodeCursor.
func encodeWPTRunCursor(timeStart time.Time, id int64) string {
	return encodeCursor[WPTRunCursor](WPTRunCursor{LastTimeStart: timeStart, LastRunID: id})
}

func encodeChromeDailyUsageCursor(date civil.Date) string {
	return encodeCursor[ChromeDailyUsageCursor](ChromeDailyUsageCursor{LastDate: date})
}

// encodeCursor: Encodes a Cursor into a base64-encoded string.
// Returns an empty string if is unable to create a token.
// TODO: Pass in context to be used by slog.ErrorContext.
func encodeCursor[T any](in T) string {
	data, err := json.Marshal(in)
	if err != nil {
		slog.Error("unable to encode cursor", "error", err)

		return ""
	}

	return base64.RawURLEncoding.EncodeToString(data)
}

// encodeFeatureResultOffsetCursor provides a wrapper around the generic encodeCursor.
func encodeFeatureResultOffsetCursor(offset int) string {
	return encodeCursor(FeatureResultOffsetCursor{
		Offset: offset,
	})
}

// entityMapper defines the core mapping operations between an external entity
// struct and its corresponding internal representation stored in Spanner. It
// provides methods to get the external key, generate a select statement, and
// retrieve the table name associated with the entity.
type entityMapper[ExternalStruct any, SpannerStruct any, ExternalKey any] interface {
	SelectOne(ExternalKey) spanner.Statement
}

// readableEntityMapper extends EntityMapper with the ability to merge an
// external entity representation into its corresponding Spanner representation.
type readableEntityMapper[ExternalStruct any, SpannerStruct any, ExternalKey any] interface {
	entityMapper[ExternalStruct, SpannerStruct, ExternalKey]
}

// writeableEntityMapper extends EntityMapper with the ability to merge an
// external entity representation into its corresponding Spanner representation.
type writeableEntityMapper[ExternalStruct any, SpannerStruct any, ExternalKey any] interface {
	readableEntityMapper[ExternalStruct, SpannerStruct, ExternalKey]
	Merge(ExternalStruct, SpannerStruct) SpannerStruct
	GetKey(ExternalStruct) ExternalKey
	Table() string
}

// writeableEntityMapperWithIDRetrieval further extends WriteableEntityMapper
// with the capability to retrieve the ID of an entity based on its external key.
type writeableEntityMapperWithIDRetrieval[ExternalStruct any, SpannerStruct any, ExternalKey any] interface {
	writeableEntityMapper[ExternalStruct, SpannerStruct, ExternalKey]
	GetID(ExternalKey) spanner.Statement
}

// entityWriterWithIDRetrieval handles Spanner resources that use Spanner-generated
// UUIDs as their primary key, but allows users to work with a different unique
// value (e.g. ExternalKey) to find and retrieve the entity's ID.
type entityWriterWithIDRetrieval[
	M writeableEntityMapperWithIDRetrieval[ExternalStruct, SpannerStruct, ExternalKey],
	ExternalStruct any,
	SpannerStruct any,
	ExternalKey any,
	ID any] struct {
	*entityWriter[M, ExternalStruct, SpannerStruct, ExternalKey]
}

// upsertAndGetID performs an upsert operation on the entity and retrieves its ID.
// It first attempts to upsert the entity using the `upsert` method from the
// embedded `entityWriter`. If successful, it then uses the `getIDByKey` method to
// fetch the entity's ID based on its external key.
func (c *entityWriterWithIDRetrieval[M, ExternalStruct, SpannerStruct, ExternalKey, ID]) upsertAndGetID(
	ctx context.Context,
	input ExternalStruct) (*ID, error) {
	err := c.upsert(ctx, input)
	if err != nil {
		return nil, err
	}

	var mapper M
	id, err := c.getIDByKey(ctx, mapper.GetKey(input))
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	return id, nil
}

// transaction implements the transaction interface that either
// ReadWriteTransaction or ReadOnlyTransaction implement.
type transaction interface {
	Query(ctx context.Context, statement spanner.Statement) *spanner.RowIterator
}

func (c *entityReader[M, ExternalStruct, SpannerStruct, ExternalKey]) readRowByKey(
	ctx context.Context,
	key ExternalKey,
) (*SpannerStruct, error) {
	txn := c.Single()
	defer txn.Close()

	return c.readRowByKeyWithTransaction(ctx, key, txn)
}

// readRowByKey retrieves the row of an entity based on its external key with transaction.
func (c *entityReader[M, ExternalStruct, SpannerStruct, ExternalKey]) readRowByKeyWithTransaction(
	ctx context.Context,
	key ExternalKey,
	txn transaction,
) (*SpannerStruct, error) {
	var mapper M
	stmt := mapper.SelectOne(key)
	// Attempt to query for the row.
	it := txn.Query(ctx, stmt)
	defer it.Stop()
	row, err := it.Next()
	if err != nil {
		// No row found
		if errors.Is(err, iterator.Done) {
			return nil, errors.Join(ErrQueryReturnedNoResults, err)
		}

		// Catch-all for other errors.
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}
	existing := new(SpannerStruct)
	err = row.ToStruct(existing)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	return existing, nil
}

// getIDByKey retrieves the ID of an entity based on its external key.
// It uses the `GetID` method from the `WriteableEntityMapperWithIDRetrieval`
// interface to generate a Spanner query to fetch the ID.
func (c *entityWriterWithIDRetrieval[M, ExternalStruct, SpannerStruct, ExternalKey, ID]) getIDByKey(
	ctx context.Context,
	key ExternalKey,
) (*ID, error) {
	var mapper M
	stmt := mapper.GetID(key)
	// Attempt to query for the row.
	txn := c.Single()
	defer txn.Close()
	it := txn.Query(ctx, stmt)
	defer it.Stop()
	row, err := it.Next()
	if err != nil {
		// No row found
		if errors.Is(err, iterator.Done) {
			return nil, errors.Join(ErrQueryReturnedNoResults, err)
		}

		// Catch-all for other errors.
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}
	var id ID
	err = row.Column(0, &id)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	return &id, nil
}

// entityReader is a basic client for reading any row from the database.
type entityReader[
	M readableEntityMapper[ExternalStruct, SpannerStruct, ExternalKey],
	ExternalStruct any,
	SpannerStruct any,
	ExternalKey any] struct {
	*Client
}

// entityWriter is a basic client for writing any row to the database.
type entityWriter[
	M writeableEntityMapper[ExternalStruct, SpannerStruct, ExternalKey],
	ExternalStruct any,
	SpannerStruct any,
	ExternalKey any] struct {
	*Client
}

// createInsertMutation simply creates a spanner mutation from the struct to the table.
func (c *entityWriter[M, ExternalStruct, S, ExternalKey]) createInsertMutation(
	mapper M, input ExternalStruct) (*spanner.Mutation, error) {
	m, err := spanner.InsertStruct(mapper.Table(), input)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	return m, nil
}

// createUpdateMutation reads an existing entity from a Spanner row, merges it with the input
// entity using the mapper's Merge method, and creates a Spanner mutation for
// updating the row.
func (c *entityWriter[M, ExternalStruct, SpannerStruct, ExternalKey]) createUpdateMutation(
	row *spanner.Row, mapper M, input ExternalStruct) (*spanner.Mutation, error) {
	existing := new(SpannerStruct)
	// Read the existing entity and merge the values.
	err := row.ToStruct(existing)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}
	// Override values
	merged := mapper.Merge(input, *existing)
	m, err := spanner.InsertOrUpdateStruct(mapper.Table(), merged)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	return m, nil
}

// upsert performs an upsert (insert or update) operation on an entity.
// It first attempts to select the entity based on its external key.
// If the entity exists, it updates it; otherwise, it inserts a new entity.
func (c *entityWriter[M, ExternalStruct, SpannerStruct, ExternalKey]) upsert(
	ctx context.Context,
	input ExternalStruct) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return c.upsertWithTransaction(ctx, txn, input)
	})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}

// upsertWithTransaction performs an upsert operation on an entity using the existing transaction.
func (c *entityWriter[M, ExternalStruct, SpannerStruct, ExternalKey]) upsertWithTransaction(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	input ExternalStruct) error {
	var mapper M
	stmt := mapper.SelectOne(mapper.GetKey(input))
	// Attempt to query for the row.
	it := txn.Query(ctx, stmt)
	defer it.Stop()
	var m *spanner.Mutation

	row, err := it.Next()
	if err != nil {
		// Check if an unexpected error occurred.
		if !errors.Is(err, iterator.Done) {
			return errors.Join(ErrInternalQueryFailure, err)
		}

		// No rows returned. Act as if this is an insertion.
		m, err = c.createInsertMutation(mapper, input)
		if err != nil {
			return err
		}
	} else {
		m, err = c.createUpdateMutation(row, mapper, input)
		if err != nil {
			return err
		}
	}
	// Buffer the mutation to be committed.
	err = txn.BufferWrite([]*spanner.Mutation{m})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}

// update performs an update operation on an entity.
// It first attempts to select the entity based on its external key.
// If the entity exists, it updates it; otherwise, it returns an error.
// nolint:unused // TODO: Remove nolint directive once the method is used.
func (c *entityWriter[M, ExternalStruct, SpannerStruct, ExternalKey]) update(
	ctx context.Context,
	input ExternalStruct) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return c.updateWithTransaction(ctx, txn, input)
	})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}

// updateWithTransaction performs an update operation on an entity using the existing transaction.
// nolint:unused // TODO: Remove nolint directive once the method is used.
func (c *entityWriter[M, ExternalStruct, SpannerStruct, ExternalKey]) updateWithTransaction(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	input ExternalStruct) error {
	var mapper M
	stmt := mapper.SelectOne(mapper.GetKey(input))
	// Attempt to query for the row.
	it := txn.Query(ctx, stmt)
	defer it.Stop()
	var m *spanner.Mutation

	row, err := it.Next()
	if err != nil {
		// No row found
		if errors.Is(err, iterator.Done) {
			return errors.Join(ErrQueryReturnedNoResults, err)
		}

		// Catch-all for other errors.
		return errors.Join(ErrInternalQueryFailure, err)
	}

	m, err = c.createUpdateMutation(row, mapper, input)
	if err != nil {
		return err
	}

	// Buffer the mutation to be committed.
	err = txn.BufferWrite([]*spanner.Mutation{m})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}

// uniquieWriteableEntityMapper extends writeableEntityMapper with the ability to remove an entity.
type uniquieWriteableEntityMapper[ExternalStruct any, SpannerStruct any, ExternalKey any] interface {
	readableEntityMapper[ExternalStruct, SpannerStruct, ExternalKey]
	GetKey(ExternalStruct) ExternalKey
	Table() string
	DeleteKey(ExternalKey) spanner.Key
}

// entityUniqueWriter is a basic client for writing a row to the database where this a unique constraint for a key.
type entityUniqueWriter[
	M uniquieWriteableEntityMapper[ExternalStruct, SpannerStruct, ExternalKey],
	ExternalStruct any,
	SpannerStruct any,
	ExternalKey any] struct {
	*Client
}

// upsertUniqueKey performs an upsert (insert or update) operation on an entity with a unique key.
// This means that the given key can only exist once. If the entity exists, it
// must be removed before inserting. This is essentially a compare and swap due to the key.
func (c *entityUniqueWriter[M, ExternalStruct, SpannerStruct, ExternalKey]) upsertUniqueKey(ctx context.Context,
	input ExternalStruct) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return c.upsertUniqueKeyWithTransaction(ctx, txn, input)
	})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}

// createInsertMutation simply creates a spanner mutation from the struct to the table.
func (c *entityUniqueWriter[M, ExternalStruct, S, ExternalKey]) createInsertMutation(
	mapper M, input ExternalStruct) (*spanner.Mutation, error) {
	m, err := spanner.InsertStruct(mapper.Table(), input)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	return m, nil
}

// upsertUniqueKeyWithTransaction performs an upsertUniqueKey operation on an entity using the existing transaction.
func (c *entityUniqueWriter[M, ExternalStruct, SpannerStruct, ExternalKey]) upsertUniqueKeyWithTransaction(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	input ExternalStruct) error {
	var mapper M
	key := mapper.GetKey(input)
	stmt := mapper.SelectOne(key)
	// Attempt to query for the row.
	it := txn.Query(ctx, stmt)
	defer it.Stop()
	var ms []*spanner.Mutation

	_, err := it.Next()
	if err != nil {
		// Check if an unexpected error occurred.
		if !errors.Is(err, iterator.Done) {
			return errors.Join(ErrInternalQueryFailure, err)
		}

		// No rows returned. Act as if this is an insertion.
		m, err := c.createInsertMutation(mapper, input)
		if err != nil {
			return err
		}
		ms = append(ms, m)
	} else {
		m1 := spanner.Delete(mapper.Table(), mapper.DeleteKey(key))
		ms = append(ms, m1)
		m2, err := c.createInsertMutation(mapper, input)
		if err != nil {
			return err
		}
		ms = append(ms, m2)
	}
	// Buffer the mutation to be committed.
	err = txn.BufferWrite(ms)
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}

// removableEntityMapper extends writeableEntityMapper with the ability to remove an entity.
type removableEntityMapper[ExternalStruct any, SpannerStruct any, ExternalKey any] interface {
	readableEntityMapper[ExternalStruct, SpannerStruct, ExternalKey]
	GetKey(ExternalStruct) ExternalKey
	DeleteKey(ExternalKey) spanner.Key
	Table() string
}

// entityRemover is a basic client for removing any row from the database.
type entityRemover[
	M removableEntityMapper[ExternalStruct, SpannerStruct, ExternalKey],
	ExternalStruct any,
	SpannerStruct any,
	ExternalKey any] struct {
	*Client
}

// remove performs an delete operation on an entity.
func (c *entityRemover[M, ExternalStruct, SpannerStruct, ExternalKey]) remove(ctx context.Context,
	input ExternalStruct) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return c.removeWithTransaction(ctx, txn, input)
	})

	return err
}

// removeWithTransaction performs an delete operation on an entity using the existing transaction.
// nolint:unused // TODO: Remove nolint directive once the method is used.
func (c *entityRemover[M, ExternalStruct, SpannerStruct, ExternalKey]) removeWithTransaction(ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	input ExternalStruct) error {
	var mapper M
	key := mapper.GetKey(input)
	stmt := mapper.SelectOne(key)
	// Attempt to query for the row.
	it := txn.Query(ctx, stmt)
	defer it.Stop()
	var m *spanner.Mutation

	_, err := it.Next()
	if err != nil {
		// No row found
		if errors.Is(err, iterator.Done) {
			return errors.Join(ErrQueryReturnedNoResults, err)
		}

		// Catch-all for other errors.
		return errors.Join(ErrInternalQueryFailure, err)
	}

	m = spanner.Delete(mapper.Table(), mapper.DeleteKey(key))

	// Buffer the mutation to be committed.
	err = txn.BufferWrite([]*spanner.Mutation{m})
	if err != nil {
		return errors.Join(ErrInternalQueryFailure, err)
	}

	return nil
}

func newEntityWriterWithIDRetrieval[
	M writeableEntityMapperWithIDRetrieval[ExternalStruct, SpannerStruct, ExternalKey],
	ID any,
	ExternalStruct any,
	SpannerStruct any,
	ExternalKey any](c *Client) *entityWriterWithIDRetrieval[M, ExternalStruct, SpannerStruct, ExternalKey, ID] {
	return &entityWriterWithIDRetrieval[M, ExternalStruct, SpannerStruct, ExternalKey, ID]{
		entityWriter: &entityWriter[M, ExternalStruct, SpannerStruct, ExternalKey]{c}}
}

func newEntityWriter[
	M writeableEntityMapper[ExternalStruct, SpannerStruct, ExternalKey],
	ExternalStruct any,
	SpannerStruct any,
	ExternalKey any](c *Client) *entityWriter[M, ExternalStruct, SpannerStruct, ExternalKey] {
	return &entityWriter[M, ExternalStruct, SpannerStruct, ExternalKey]{c}
}

func newUniqueEntityWriter[
	M uniquieWriteableEntityMapper[ExternalStruct, SpannerStruct, ExternalKey],
	ExternalStruct any,
	SpannerStruct any,
	ExternalKey any](c *Client) *entityUniqueWriter[M, ExternalStruct, SpannerStruct, ExternalKey] {
	return &entityUniqueWriter[M, ExternalStruct, SpannerStruct, ExternalKey]{c}
}

func newEntityReader[
	M readableEntityMapper[ExternalStruct, SpannerStruct, ExternalKey],
	SpannerStruct any,
	ExternalStruct any,
	ExternalKey any](c *Client) *entityReader[M, ExternalStruct, SpannerStruct, ExternalKey] {
	return &entityReader[M, ExternalStruct, SpannerStruct, ExternalKey]{c}
}

func newEntityRemover[
	M removableEntityMapper[ExternalStruct, SpannerStruct, ExternalKey],
	SpannerStruct any,
	ExternalStruct any,
	ExternalKey any](c *Client) *entityRemover[M, ExternalStruct, SpannerStruct, ExternalKey] {
	return &entityRemover[M, ExternalStruct, SpannerStruct, ExternalKey]{c}
}

func concurrentBatchWriteEntity[SpannerStruct any](
	ctx context.Context, c *Client, wg *sync.WaitGroup, batchSize int,
	entityChan <-chan SpannerStruct, table string, errChan chan error, workerID int) {
	var totalBatches, entityCount uint
	success := true
	defer func() {
		wg.Done()
		slog.InfoContext(ctx, "batch writer worker finishing", "id", workerID,
			"totalBatches", totalBatches, "entityCount", entityCount,
			"success", success, "table", table)
	}()
	slog.InfoContext(ctx, "batch writer worker starting", "id", workerID, "table", table)
	for {
		batch := make([]*spanner.Mutation, 0, batchSize)
		for i := 0; i < batchSize; i++ {
			select {
			case entity, isChannelStillOpen := <-entityChan:
				// If the channel is closed, go ahead and apply what we have and return.
				if !isChannelStillOpen {
					if len(batch) > 0 {
						slog.InfoContext(ctx, "sending final batch", "size", len(batch), "id", workerID, "table", table)
						totalBatches++
						entityCount += uint(len(batch))
						err := c.BatchWriteMutations(ctx, c.Client, batch)
						if err != nil {
							success = false
							errChan <- err
						}
					}

					return
				}
				// Else, the channel is still open and it has received a value.
				// Create a mutation and append it to the upcoming batch
				m, err := spanner.InsertOrUpdateStruct(table, entity)
				if err != nil {
					success = false
					errChan <- err

					return
				}
				batch = append(batch, m)
			case <-ctx.Done():
				// If the system tells us that we are done, we can abort too.
				return
			}
		}
		// The current batch is full. Send the mutations to the database.
		totalBatches++
		entityCount += uint(len(batch))
		err := c.BatchWriteMutations(ctx, c.Client, batch)
		if err != nil {
			success = false
			errChan <- err

			return
		}
	}
}

func runConcurrentBatch[SpannerStruct any](ctx context.Context, c *Client,
	producerFn func(entityChan chan<- SpannerStruct), table string) error {
	entityChan := make(chan SpannerStruct, c.batchSize)
	errChan := make(chan error)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	workers := c.batchWriters
	wg.Add(workers)
	doneChan := make(chan struct{})
	go func() {
		slog.InfoContext(ctx, "waiting for batch write wait group to finish", "table", table)
		wg.Wait()
		slog.InfoContext(ctx, "batch write wait group to finished", "table", table)
		close(doneChan)
	}()
	for i := 0; i < workers; i++ {
		go concurrentBatchWriteEntity(ctx, c, &wg, c.batchSize, entityChan, table, errChan, i)
	}
	producerFn(entityChan)
	close(entityChan)

	// Check for errors from the goroutine
	select {
	case err := <-errChan:
		cancel()

		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-doneChan:
		return nil
	}
}

// OptionallySet allows distinguishing between setting a value
// and leaving it unchanged. Useful for PATCH operations where
// only specific fields are updated.
type OptionallySet[T any] struct {
	Value T
	IsSet bool
}
