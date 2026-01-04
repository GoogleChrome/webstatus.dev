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

package gcppubsubadapters

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	searchconfigv1 "github.com/GoogleChrome/webstatus.dev/lib/event/searchconfigurationchanged/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

type BackendAdapter struct {
	client  EventPublisher
	topicID string
}

func NewBackendAdapter(client EventPublisher, topicID string) *BackendAdapter {
	return &BackendAdapter{client: client, topicID: topicID}
}

func (p *BackendAdapter) PublishSearchConfigurationChanged(
	ctx context.Context,
	resp *backend.SavedSearchResponse,
	userID string,
	isCreation bool) error {

	evt := searchconfigv1.SearchConfigurationChangedEvent{
		SearchID:   resp.Id,
		Query:      resp.Query,
		UserID:     userID,
		Timestamp:  resp.UpdatedAt,
		IsCreation: isCreation,
		Frequency:  searchconfigv1.FrequencyImmediate,
	}

	msg, err := event.New(evt)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	id, err := p.client.Publish(ctx, p.topicID, msg)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	slog.InfoContext(ctx, "published search configuration changed event",
		"msgID", id,
		"searchID", evt.SearchID,
		"isCreation", evt.IsCreation)

	return nil
}
