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
)

// entitySynchronizer specific errors.
var (
	ErrSyncReadFailed                = errors.New("sync failed during read phase")
	ErrSyncMutationCreationFailed    = errors.New("sync failed to create mutation")
	ErrSyncAtomicWriteFailed         = errors.New("sync atomic write failed")
	ErrSyncBatchWriteFailed          = errors.New("sync batch write failed")
	ErrSyncFailedToGetChildMutations = errors.New("sync failed to get child mutations")
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
	// Max number saved searches per user.
	maxOwnedSearchesPerUser uint32
	// Max number of bookmarks per user (excluding the saved searches they own)
	maxBookmarksPerUser uint32
}

const defaultMaxOwnedSearchesPerUser = 25
const defaultMaxBookmarksPerUser = 25
const defaultBatchSize = 5000
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
		slog.InfoContext(context.TODO(), "using local batch writer")
		bw = LocalBatchWriter{}
	}

	return &Client{
		client,
		GCPFeatureSearchBaseQuery{},
		GCPMissingOneImplementationQuery{},
		searchConfig{
			maxOwnedSearchesPerUser: defaultMaxOwnedSearchesPerUser,
			maxBookmarksPerUser:     defaultMaxBookmarksPerUser,
		},
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
		slog.ErrorContext(context.TODO(), "unable to encode cursor", "error", err)

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

// --- Generic Entity Mapper Interfaces ---

// baseMapper provides the table name.
type baseMapper interface {
	Table() string
}

// externalKeyMapper handles getting the business key from an external struct.
type externalKeyMapper[ExternalStruct any, Key comparable] interface {
	GetKeyFromExternal(ExternalStruct) Key
}

// internalKeyMapper handles getting the primary key from an internal Spanner struct.
type internalKeyMapper[SpannerStruct any, Key comparable] interface {
	GetKeyFromInternal(SpannerStruct) Key
}

// readOneMapper provides a method to select a single entity by its key.
type readOneMapper[Key comparable] interface {
	SelectOne(Key) spanner.Statement
}

// readAllMapper provides a method to select all entities.
type readAllMapper interface {
	SelectAll() spanner.Statement
}

// mergeMapper handles the logic for updating an existing entity.
type mergeMapper[ExternalStruct any, SpannerStruct any] interface {
	Merge(ExternalStruct, SpannerStruct) SpannerStruct
}

// mergeAndCheckChangedMapper handles merging and explicitly returns if a change occurred.
// TODO: Long term: all the mappers should move from mergeMapper to mergeAndCheckChangedMapper.
type mergeAndCheckChangedMapper[ExternalStruct any, SpannerStruct any] interface {
	MergeAndCheckChanged(ExternalStruct, SpannerStruct) (SpannerStruct, bool)
}

// idRetrievalMapper provides a method to get a Spanner ID from a business key.
type idRetrievalMapper[Key comparable] interface {
	GetID(Key) spanner.Statement
}

// deleteByKeyMapper provides a way to create a delete mutation from a key.
type deleteByKeyMapper[Key comparable] interface {
	DeleteKey(Key) spanner.Key
}

// deleteByStructMapper provides a way to create a delete mutation from a struct.
type deleteByStructMapper[SpannerStruct any] interface {
	DeleteMutation(SpannerStruct) *spanner.Mutation
}

// --- Composed Interfaces for Specific Components ---

// readableEntityMapper is composed for the entityReader.
type readableEntityMapper[ExternalStruct any, SpannerStruct any, Key comparable] interface {
	readOneMapper[Key]
}

// writeableEntityMapper is composed for the entityWriter.
type writeableEntityMapper[ExternalStruct any, SpannerStruct any, Key comparable] interface {
	baseMapper
	externalKeyMapper[ExternalStruct, Key]
	readOneMapper[Key]
	mergeMapper[ExternalStruct, SpannerStruct]
}

// writeableEntityMapperWithIDRetrieval is composed for the entityWriter that
// also needs to fetch a Spanner-generated ID.
type writeableEntityMapperWithIDRetrieval[ExternalStruct any, SpannerStruct any, Key comparable] interface {
	writeableEntityMapper[ExternalStruct, SpannerStruct, Key]
	idRetrievalMapper[Key]
}

// uniquieWriteableEntityMapper is composed for the entityUniqueWriter.
type uniquieWriteableEntityMapper[ExternalStruct any, SpannerStruct any, Key comparable] interface {
	baseMapper
	externalKeyMapper[ExternalStruct, Key]
	readOneMapper[Key]
	deleteByKeyMapper[Key]
}

// removableEntityMapper is composed for the entityRemover.
type removableEntityMapper[ExternalStruct any, SpannerStruct any, Key comparable] interface {
	baseMapper
	externalKeyMapper[ExternalStruct, Key]
	readOneMapper[Key] // To verify the entity exists before deleting.
	deleteByKeyMapper[Key]
}

// syncableEntityMapper is composed for the entitySynchronizer. It needs
// methods to read all entities, handle keys, merge, and delete by struct.
type syncableEntityMapper[ExternalStruct any, SpannerStruct any, Key comparable] interface {
	baseMapper
	externalKeyMapper[ExternalStruct, Key]
	internalKeyMapper[SpannerStruct, Key]
	readAllMapper
	// mergeMapper[ExternalStruct, SpannerStruct]
	mergeAndCheckChangedMapper[ExternalStruct, SpannerStruct]
	childDeleterMapper[SpannerStruct]
	deleteByStructMapper[SpannerStruct]
}

type ChildDeleteKeyMutations struct {
	tableName string
	mutations []*spanner.Mutation
}

// childDeleterMapper provides a method to get the keys of child entities
// that need to be deleted before their parents.
type childDeleterMapper[SpannerStruct any] interface {
	GetChildDeleteKeyMutations(
		ctx context.Context,
		client *Client,
		parentsToDelete []SpannerStruct,
	) ([]ChildDeleteKeyMutations, error)
}

// --- Generic Entity Components ---

// entityWriterWithIDRetrieval handles Spanner resources that use Spanner-generated
// UUIDs as their primary key, but allows users to work with a different unique
// value (e.g. Key) to find and retrieve the entity's ID.
type entityWriterWithIDRetrieval[
	M writeableEntityMapperWithIDRetrieval[ExternalStruct, SpannerStruct, Key],
	ExternalStruct any,
	SpannerStruct any,
	Key comparable,
	ID any] struct {
	*entityWriter[M, ExternalStruct, SpannerStruct, Key]
}

// upsertAndGetID performs an upsert operation on the entity and retrieves its ID.
// It first attempts to upsert the entity using the `upsert` method from the
// embedded `entityWriter`. If successful, it then uses the `getIDByKey` method to
// fetch the entity's ID based on its external key.
func (c *entityWriterWithIDRetrieval[M, ExternalStruct, SpannerStruct, Key, ID]) upsertAndGetID(
	ctx context.Context,
	input ExternalStruct) (*ID, error) {
	err := c.upsert(ctx, input)
	if err != nil {
		return nil, err
	}

	var mapper M
	id, err := c.getIDByKey(ctx, mapper.GetKeyFromExternal(input))
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

func (c *entityReader[M, ExternalStruct, SpannerStruct, Key]) readRowByKey(
	ctx context.Context,
	key Key,
) (*SpannerStruct, error) {
	txn := c.Single()
	defer txn.Close()

	return c.readRowByKeyWithTransaction(ctx, key, txn)
}

// readRowByKey retrieves the row of an entity based on its external key with transaction.
func (c *entityReader[M, ExternalStruct, SpannerStruct, Key]) readRowByKeyWithTransaction(
	ctx context.Context,
	key Key,
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
func (c *entityWriterWithIDRetrieval[M, ExternalStruct, SpannerStruct, Key, ID]) getIDByKey(
	ctx context.Context,
	key Key,
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
	M readableEntityMapper[ExternalStruct, SpannerStruct, Key],
	ExternalStruct any,
	SpannerStruct any,
	Key comparable] struct {
	*Client
}

// entityWriter is a basic client for writing any row to the database.
type entityWriter[
	M writeableEntityMapper[ExternalStruct, SpannerStruct, Key],
	ExternalStruct any,
	SpannerStruct any,
	Key comparable] struct {
	*Client
}

// createInsertMutation simply creates a spanner mutation from the struct to the table.
func (c *entityWriter[M, ExternalStruct, S, Key]) createInsertMutation(
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
func (c *entityWriter[M, ExternalStruct, SpannerStruct, Key]) createUpdateMutation(
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
func (c *entityWriter[M, ExternalStruct, SpannerStruct, Key]) upsert(
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
func (c *entityWriter[M, ExternalStruct, SpannerStruct, Key]) upsertWithTransaction(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	input ExternalStruct) error {
	var mapper M
	stmt := mapper.SelectOne(mapper.GetKeyFromExternal(input))
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
func (c *entityWriter[M, ExternalStruct, SpannerStruct, Key]) update(
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
func (c *entityWriter[M, ExternalStruct, SpannerStruct, Key]) updateWithTransaction(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	input ExternalStruct) error {
	var mapper M
	stmt := mapper.SelectOne(mapper.GetKeyFromExternal(input))
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

// entityUniqueWriter is a basic client for writing a row to the database where this a unique constraint for a key.
type entityUniqueWriter[
	M uniquieWriteableEntityMapper[ExternalStruct, SpannerStruct, Key],
	ExternalStruct any,
	SpannerStruct any,
	Key comparable] struct {
	*Client
}

// upsertUniqueKey performs an upsert (insert or update) operation on an entity with a unique key.
// This means that the given key can only exist once. If the entity exists, it
// must be removed before inserting. This is essentially a compare and swap due to the key.
func (c *entityUniqueWriter[M, ExternalStruct, SpannerStruct, Key]) upsertUniqueKey(ctx context.Context,
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
func (c *entityUniqueWriter[M, ExternalStruct, S, Key]) createInsertMutation(
	mapper M, input ExternalStruct) (*spanner.Mutation, error) {
	m, err := spanner.InsertStruct(mapper.Table(), input)
	if err != nil {
		return nil, errors.Join(ErrInternalQueryFailure, err)
	}

	return m, nil
}

// upsertUniqueKeyWithTransaction performs an upsertUniqueKey operation on an entity using the existing transaction.
func (c *entityUniqueWriter[M, ExternalStruct, SpannerStruct, Key]) upsertUniqueKeyWithTransaction(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	input ExternalStruct) error {
	var mapper M
	key := mapper.GetKeyFromExternal(input)
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

// entityRemover is a basic client for removing any row from the database.
type entityRemover[
	M removableEntityMapper[ExternalStruct, SpannerStruct, Key],
	ExternalStruct any,
	SpannerStruct any,
	Key comparable] struct {
	*Client
}

// remove performs an delete operation on an entity.
// nolint: unused // TODO: Remove nolint directive once the method is used.
func (c *entityRemover[M, ExternalStruct, SpannerStruct, Key]) remove(ctx context.Context,
	input ExternalStruct) error {
	_, err := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return c.removeWithTransaction(ctx, txn, input)
	})

	return err
}

// removeWithTransaction performs an delete operation on an entity using the existing transaction.
// nolint:unused // TODO: Remove nolint directive once the method is used.
func (c *entityRemover[M, ExternalStruct, SpannerStruct, Key]) removeWithTransaction(ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	input ExternalStruct) error {
	var mapper M
	key := mapper.GetKeyFromExternal(input)
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

// entitySynchronizer handles the synchronization of a Spanner table with a
// desired state provided as a slice of entities. It determines whether to
// use a single atomic transaction or a high-throughput batch write based on
// the number of changes.
type entitySynchronizer[
	M syncableEntityMapper[ExternalStruct, SpannerStruct, Key],
	ExternalStruct any,
	SpannerStruct any,
	Key comparable,
] struct {
	*Client
	// The number of mutations at which the synchronizer will switch from a
	// single atomic transaction to the non-atomic batch writer.
	batchWriteThreshold int
}

// newEntitySynchronizer creates a new synchronizer with a default threshold.
func newEntitySynchronizer[
	M syncableEntityMapper[ExternalStruct, SpannerStruct, Key],
	ExternalStruct any,
	SpannerStruct any,
	Key comparable,
](
	c *Client,
) *entitySynchronizer[M, ExternalStruct, SpannerStruct, Key] {
	return &entitySynchronizer[M, ExternalStruct, SpannerStruct, Key]{
		Client:              c,
		batchWriteThreshold: defaultBatchSize,
	}
}

// Sync reconciles the state of a Spanner table with a provided list of desired entities.
// It includes detailed logging for each operation for auditing purposes.
func (s *entitySynchronizer[M, ExternalStruct, SpannerStruct, Key]) Sync(
	ctx context.Context,
	desiredState []ExternalStruct,
) error {
	var mapper M
	tableName := mapper.Table()

	// 1. READ: Fetch all existing entities from the database.
	slog.InfoContext(ctx, "Starting sync: reading all existing entities", "table", tableName)
	stmt := mapper.SelectAll()
	iter := s.Single().Query(ctx, stmt)
	defer iter.Stop()

	existingEntities := make(map[Key]SpannerStruct)
	err := iter.Do(func(r *spanner.Row) error {
		var spannerEntity SpannerStruct
		if err := r.ToStruct(&spannerEntity); err != nil {
			return err
		}
		existingEntities[mapper.GetKeyFromInternal(spannerEntity)] = spannerEntity

		return nil
	})
	if err != nil {
		return errors.Join(ErrSyncReadFailed, err)
	}
	slog.InfoContext(ctx, "Read complete", "table", tableName, "existing_count", len(existingEntities))

	// 2. COMPUTE DIFF: Separate mutations into deletes and upserts.
	var inserts, updates, deletes int
	upsertMutations := []*spanner.Mutation{}
	deleteMutations := []*spanner.Mutation{}
	desiredKeys := make(map[Key]struct{})

	for _, externalEntity := range desiredState {
		key := mapper.GetKeyFromExternal(externalEntity)
		desiredKeys[key] = struct{}{}

		var m *spanner.Mutation
		var err error

		if existing, found := existingEntities[key]; found {
			// --- UPDATE logic ---
			slog.DebugContext(ctx, "Preparing update", "table", tableName, "key", key)
			merged, hasChanged := mapper.MergeAndCheckChanged(externalEntity, existing)
			if !hasChanged {
				continue
			}
			updates++
			m, err = spanner.UpdateStruct(tableName, merged)
			if err != nil {
				return errors.Join(ErrSyncMutationCreationFailed, err)
			}
		} else {
			// --- INSERT logic ---
			slog.DebugContext(ctx, "Preparing insert", "table", tableName, "key", key)
			inserts++
			m, err = spanner.InsertStruct(tableName, externalEntity)
			if err != nil {
				return errors.Join(ErrSyncMutationCreationFailed, err)
			}
		}
		upsertMutations = append(upsertMutations, m)
	}

	entitiesToDelete := make([]SpannerStruct, 0)
	for key, entity := range existingEntities {
		if _, found := desiredKeys[key]; !found {
			slog.DebugContext(ctx, "Preparing delete", "table", tableName, "key", key)
			deletes++
			deleteMutations = append(deleteMutations, mapper.DeleteMutation(entity))
			entitiesToDelete = append(entitiesToDelete, entity)
		}
	}

	slog.InfoContext(ctx, "Diff computed",
		"table", tableName,
		"inserts", inserts,
		"updates", updates,
		"deletes", deletes)

	// 3. APPLY DELETES: Handle child and parent deletions first.
	if err := s.applyDeletes(ctx, entitiesToDelete, deleteMutations, mapper); err != nil {
		return err
	}

	// 4. APPLY UPSERTS: Apply all inserts and updates together.
	if len(upsertMutations) < s.batchWriteThreshold {
		err = s.applyAtomic(ctx, upsertMutations, tableName)
	} else {
		err = s.applyNonAtomic(ctx, upsertMutations, tableName)
	}

	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "Sync successful", "table", tableName)

	return nil
}

func (s *entitySynchronizer[M, ExternalStruct, SpannerStruct, Key]) mutationNoop(
	m *spanner.Mutation) (*spanner.Mutation, error) {
	return m, nil
}

// applyDeletes handles the deletion of child and parent entities.
func (s *entitySynchronizer[M, ExternalStruct, SpannerStruct, Key]) applyDeletes(
	ctx context.Context, entitiesToDelete []SpannerStruct, deleteMutations []*spanner.Mutation,
	mapper M) error {
	if len(entitiesToDelete) == 0 {
		return nil
	}
	tableName := mapper.Table()

	// Handle manual child deletions first.
	// The `ON DELETE CASCADE` constraint should be the default, but it can fail
	// if a cascade exceeds Spanner's 80k mutation limit.
	//
	// If a new table's sync starts failing on the parent delete step below,
	// its mapper should be updated to implement `GetChildDeleteKeyMutations`
	// to handle the child deletes manually.
	// See: https://github.com/GoogleChrome/webstatus.dev/issues/1697
	childKeyMutationSet, err := mapper.GetChildDeleteKeyMutations(ctx, s.Client, entitiesToDelete)
	if err != nil {
		return errors.Join(ErrSyncFailedToGetChildMutations, err)
	}
	for _, childKeyMutations := range childKeyMutationSet {
		slog.InfoContext(ctx, "Applying child delete mutations via batch writer",
			"count", len(childKeyMutations.mutations), "table", childKeyMutations.tableName)
		err := s.applyNonAtomic(ctx, childKeyMutations.mutations, childKeyMutations.tableName)
		if err != nil {
			return err
		}
	}

	// Delete the parent entities.
	slog.InfoContext(ctx,
		"Applying parent delete mutations via batch writer",
		"count", len(deleteMutations), "table", tableName)

	err = s.applyNonAtomic(ctx, deleteMutations, tableName)
	if err != nil {
		// See above comment about GetChildDeleteKeyMutations for possible fix.
		slog.ErrorContext(ctx, "Failed to apply parent delete mutations", "error", err)

		return err
	}

	return nil
}

// applyAtomic applies all upsert mutations in a single, atomic transaction.
func (s *entitySynchronizer[M, ExternalStruct, SpannerStruct, Key]) applyAtomic(
	ctx context.Context,
	upsertMutations []*spanner.Mutation,
	tableName string) error {
	slog.InfoContext(ctx, "Applying upsert mutations via single atomic transaction",
		"table", tableName, "count", len(upsertMutations))
	_, err := s.Apply(ctx, upsertMutations)
	if err != nil {
		return errors.Join(ErrSyncAtomicWriteFailed, err)
	}

	return nil
}

// applyNonAtomic applies upsert mutations in batches.
func (s *entitySynchronizer[M, ExternalStruct, SpannerStruct, Key]) applyNonAtomic(
	ctx context.Context,
	mutations []*spanner.Mutation,
	tableName string) error {
	slog.WarnContext(ctx,
		"Applying upsert mutations via non-atomic batch writer due to large mutation count",
		"table", tableName, "count", len(mutations))

	producerFn := func(mutationChan chan<- *spanner.Mutation) {
		for _, m := range mutations {
			mutationChan <- m
		}
	}
	err := runConcurrentBatch(ctx, s.Client, producerFn, tableName, s.mutationNoop)
	if err != nil {
		return errors.Join(ErrSyncBatchWriteFailed, err)
	}

	return nil
}

func newEntityWriterWithIDRetrieval[
	M writeableEntityMapperWithIDRetrieval[ExternalStruct, SpannerStruct, Key],
	ID any,
	ExternalStruct any,
	SpannerStruct any,
	Key comparable](c *Client) *entityWriterWithIDRetrieval[M, ExternalStruct, SpannerStruct, Key, ID] {
	return &entityWriterWithIDRetrieval[M, ExternalStruct, SpannerStruct, Key, ID]{
		entityWriter: &entityWriter[M, ExternalStruct, SpannerStruct, Key]{c}}
}

func newEntityWriter[
	M writeableEntityMapper[ExternalStruct, SpannerStruct, Key],
	ExternalStruct any,
	SpannerStruct any,
	Key comparable](c *Client) *entityWriter[M, ExternalStruct, SpannerStruct, Key] {
	return &entityWriter[M, ExternalStruct, SpannerStruct, Key]{c}
}

func newUniqueEntityWriter[
	M uniquieWriteableEntityMapper[ExternalStruct, SpannerStruct, Key],
	ExternalStruct any,
	SpannerStruct any,
	Key comparable](c *Client) *entityUniqueWriter[M, ExternalStruct, SpannerStruct, Key] {
	return &entityUniqueWriter[M, ExternalStruct, SpannerStruct, Key]{c}
}

func newEntityReader[
	M readableEntityMapper[ExternalStruct, SpannerStruct, Key],
	SpannerStruct any,
	ExternalStruct any,
	Key comparable](c *Client) *entityReader[M, ExternalStruct, SpannerStruct, Key] {
	return &entityReader[M, ExternalStruct, SpannerStruct, Key]{c}
}

func newEntityRemover[
	M removableEntityMapper[ExternalStruct, SpannerStruct, Key],
	SpannerStruct any,
	ExternalStruct any,
	Key comparable](c *Client) *entityRemover[M, ExternalStruct, SpannerStruct, Key] {
	return &entityRemover[M, ExternalStruct, SpannerStruct, Key]{c}
}

func concurrentBatchWriteEntity[SpannerStruct any](
	ctx context.Context, c *Client, wg *sync.WaitGroup, batchSize int,
	entityChan <-chan SpannerStruct, toMutationFn func(SpannerStruct) (*spanner.Mutation, error),
	table string, errChan chan error, workerID int) {
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
				m, err := toMutationFn(entity)
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
	producerFn func(entityChan chan<- SpannerStruct), table string,
	toMutationFn func(SpannerStruct) (*spanner.Mutation, error)) error {
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
	for i := range workers {
		go concurrentBatchWriteEntity(ctx, c, &wg, c.batchSize, entityChan, toMutationFn, table, errChan, i)
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
