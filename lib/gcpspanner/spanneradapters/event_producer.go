// Copyright 2025 Google LLC
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

package spanneradapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type BackendAdapterForEventProducerDiffer interface {
	GetFeature(
		ctx context.Context,
		featureID string,
		wptMetricView backend.WPTMetricView,
		browsers []backend.BrowserPathParam,
	) (*backendtypes.GetFeatureResult, error)
	FeaturesSearch(
		ctx context.Context,
		pageToken *string,
		pageSize int,
		searchNode *searchtypes.SearchNode,
		sortOrder *backend.ListFeaturesParamsSort,
		wptMetricView backend.WPTMetricView,
		browsers []backend.BrowserPathParam,
	) (*backend.FeaturePage, error)
}

type EventProducerDiffer struct {
	backendAdapter BackendAdapterForEventProducerDiffer
}

type EventProducerDifferSpannerClient interface {
	BackendSpannerClient
}

// NewEventProducerDiffer constructs an adapter for the differ in the event producer service.
func NewEventProducerDiffer(adapter BackendAdapterForEventProducerDiffer) *EventProducerDiffer {
	return &EventProducerDiffer{backendAdapter: adapter}
}

func (e *EventProducerDiffer) GetFeature(
	ctx context.Context,
	featureID string) (*backendtypes.GetFeatureResult, error) {
	return e.backendAdapter.GetFeature(ctx, featureID, backend.TestCounts,
		backendtypes.DefaultBrowsers())
}

func (e *EventProducerDiffer) FetchFeatures(ctx context.Context, query string) ([]backend.Feature, error) {
	parser := searchtypes.FeaturesSearchQueryParser{}
	node, err := parser.Parse(query)
	if err != nil {
		return nil, err
	}
	var features []backend.Feature

	defaultSort := backend.NameAsc
	var pageToken *string
	for {
		featurePage, err := e.backendAdapter.FeaturesSearch(
			ctx,
			pageToken,
			// TODO: Use helper for page size https://github.com/GoogleChrome/webstatus.dev/issues/2122
			100,
			node,
			&defaultSort,
			//TODO: Use helper for test type https://github.com/GoogleChrome/webstatus.dev/issues/2122
			backend.TestCounts,
			backendtypes.DefaultBrowsers(),
		)
		if err != nil {
			return nil, err
		}
		features = append(features, featurePage.Data...)
		if featurePage.Metadata.NextPageToken == nil {
			break
		}
		pageToken = featurePage.Metadata.NextPageToken
	}

	return features, nil
}

type EventProducerSpannerClient interface {
	TryAcquireSavedSearchStateWorkerLock(
		ctx context.Context,
		savedSearchID string,
		snapshotType gcpspanner.SavedSearchSnapshotType,
		workerID string,
		ttl time.Duration) (bool, error)
	PublishSavedSearchNotificationEvent(ctx context.Context,
		event gcpspanner.SavedSearchNotificationCreateRequest, newStatePath, workerID string,
		opts ...gcpspanner.CreateOption) (*string, error)
	GetLatestSavedSearchNotificationEvent(
		ctx context.Context,
		savedSearchID string,
		snapshotType gcpspanner.SavedSearchSnapshotType,
	) (*gcpspanner.SavedSearchNotificationEvent, error)
	ReleaseSavedSearchStateWorkerLock(
		ctx context.Context,
		savedSearchID string,
		snapshotType gcpspanner.SavedSearchSnapshotType,
		workerID string) error
}

type EventProducer struct {
	client EventProducerSpannerClient
}

func NewEventProducer(client EventProducerSpannerClient) *EventProducer {
	return &EventProducer{client: client}
}

func convertFrequencyToSnapshotType(freq workertypes.JobFrequency) gcpspanner.SavedSearchSnapshotType {
	switch freq {
	// Eventually daily and unknown will be their own types.
	case workertypes.FrequencyImmediate, workertypes.FrequencyDaily, workertypes.FrequencyUnknown:
		return gcpspanner.SavedSearchSnapshotTypeImmediate
	case workertypes.FrequencyWeekly:
		return gcpspanner.SavedSearchSnapshotTypeWeekly
	case workertypes.FrequencyMonthly:
		return gcpspanner.SavedSearchSnapshotTypeMonthly
	}

	return gcpspanner.SavedSearchSnapshotTypeImmediate
}

func convertWorktypeReasonsToSpanner(reasons []workertypes.Reason) []string {
	if reasons == nil {
		return nil
	}
	spannerReasons := make([]string, 0, len(reasons))
	for _, r := range reasons {
		spannerReasons = append(spannerReasons, string(r))
	}

	return spannerReasons
}

func (e *EventProducer) AcquireLock(ctx context.Context, searchID string, frequency workertypes.JobFrequency,
	workerID string, lockTTL time.Duration) error {
	snapshotType := convertFrequencyToSnapshotType(frequency)
	_, err := e.client.TryAcquireSavedSearchStateWorkerLock(
		ctx,
		searchID,
		snapshotType,
		workerID,
		lockTTL,
	)

	return err
}

func (e *EventProducer) GetLatestEvent(ctx context.Context, frequency workertypes.JobFrequency,
	searchID string) (*workertypes.LatestEventInfo, error) {
	snapshotType := convertFrequencyToSnapshotType(frequency)

	event, err := e.client.GetLatestSavedSearchNotificationEvent(ctx, searchID, snapshotType)
	if err != nil {
		return nil, err
	}

	return &workertypes.LatestEventInfo{
		EventID:       event.ID,
		StateBlobPath: event.BlobPath,
	}, nil
}

func (e *EventProducer) ReleaseLock(ctx context.Context, searchID string, frequency workertypes.JobFrequency,
	workerID string) error {
	snapshotType := convertFrequencyToSnapshotType(frequency)

	return e.client.ReleaseSavedSearchStateWorkerLock(ctx, searchID, snapshotType, workerID)
}

func (e *EventProducer) PublishEvent(ctx context.Context, req workertypes.PublishEventRequest) error {
	var summaryObj interface{}
	if req.Summary != nil {
		if err := json.Unmarshal(req.Summary, &summaryObj); err != nil {
			return fmt.Errorf("failed to unmarshal summary JSON: %w", err)
		}
	}
	snapshotType := convertFrequencyToSnapshotType(req.Frequency)
	_, err := e.client.PublishSavedSearchNotificationEvent(ctx, gcpspanner.SavedSearchNotificationCreateRequest{
		SavedSearchID: req.SearchID,
		SnapshotType:  snapshotType,
		Timestamp:     req.GeneratedAt,
		EventType:     "", // TODO: Set appropriate event type
		Reasons:       convertWorktypeReasonsToSpanner(req.Reasons),
		BlobPath:      req.StateBlobPath,
		DiffBlobPath:  req.DiffBlobPath,
		Summary:       spanner.NullJSON{Value: map[string]any{"summary": summaryObj}, Valid: req.Summary != nil},
	},
		req.StateBlobPath,
		req.EventID,
		gcpspanner.WithID(req.EventID),
	)
	if err != nil {
		slog.ErrorContext(ctx, "unable to publish notification event", "error", err, "eventID", req.EventID)

		return err
	}

	return nil
}

type BatchEventProducerSpannerClient interface {
	ListAllSavedSearches(
		ctx context.Context) ([]gcpspanner.SavedSearchBriefDetails, error)
}

type BatchEventProducer struct {
	client BatchEventProducerSpannerClient
}

func NewBatchEventProducer(client BatchEventProducerSpannerClient) *BatchEventProducer {
	return &BatchEventProducer{client: client}
}

func (b *BatchEventProducer) ListAllSavedSearches(ctx context.Context) ([]workertypes.SearchJob, error) {
	details, err := b.client.ListAllSavedSearches(ctx)
	if err != nil {
		return nil, err
	}

	jobs := make([]workertypes.SearchJob, 0, len(details))
	for _, detail := range details {
		jobs = append(jobs, workertypes.SearchJob{ID: detail.ID, Query: detail.Query})
	}

	return jobs, nil
}
