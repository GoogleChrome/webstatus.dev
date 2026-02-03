// Copyright 2026 Google LLC
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

package producer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type SearchLister interface {
	ListAllSavedSearches(ctx context.Context) ([]workertypes.SearchJob, error)
}

type CommandPublisher interface {
	PublishRefreshCommand(ctx context.Context, cmd workertypes.RefreshSearchCommand) error
}

type BatchUpdateHandler struct {
	lister    SearchLister
	publisher CommandPublisher
	now       func() time.Time
}

func NewBatchUpdateHandler(lister SearchLister, publisher CommandPublisher) *BatchUpdateHandler {
	return &BatchUpdateHandler{
		lister:    lister,
		publisher: publisher,
		now:       time.Now,
	}
}

func (h *BatchUpdateHandler) ProcessBatchUpdate(ctx context.Context, triggerID string,
	frequency workertypes.JobFrequency) error {
	slog.InfoContext(ctx, "starting batch update fan-out", "trigger_id", triggerID, "frequency", frequency)

	// 1. List all Saved Searches
	searches, err := h.lister.ListAllSavedSearches(ctx)
	if err != nil {
		// Transient db error should be retried
		return fmt.Errorf("%w: failed to list saved searches: %w", event.ErrTransientFailure, err)
	}

	slog.InfoContext(ctx, "found saved searches to refresh", "count", len(searches))

	// 2. Fan-out
	for _, search := range searches {
		cmd := workertypes.RefreshSearchCommand{
			SearchID:  search.ID,
			Query:     search.Query,
			Frequency: frequency,
			Timestamp: h.now(),
		}

		if err := h.publisher.PublishRefreshCommand(ctx, cmd); err != nil {
			// If we fail to publish one, we should probably fail the whole batch so it retries.
			// But we don't want to re-publish successfully published ones if possible.
			// Pub/Sub doesn't support transactional batch publishes across messages easily.
			// Ideally, we just return error and let the handler retry.
			// Idempotency in ProcessSearch handles the duplicates.
			slog.ErrorContext(ctx, "failed to publish refresh command", "search_id", search.ID, "error", err,
				"trigger_id", triggerID, "frequency", frequency)

			return fmt.Errorf("%w: failed to publish refresh command for search %s: %w",
				event.ErrTransientFailure, search.ID, err)
		}
	}

	slog.InfoContext(ctx, "batch update fan-out complete", "trigger_id", triggerID)

	return nil
}
