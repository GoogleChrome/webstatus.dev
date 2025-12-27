// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	refreshv1 "github.com/GoogleChrome/webstatus.dev/lib/event/refreshsearchcommand/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type BatchFanOutPublisherAdapter struct {
	client  EventPublisher
	topicID string
}

func NewBatchFanOutPublisherAdapter(client EventPublisher, topicID string) *BatchFanOutPublisherAdapter {
	return &BatchFanOutPublisherAdapter{client: client, topicID: topicID}
}

func (a *BatchFanOutPublisherAdapter) PublishRefreshCommand(ctx context.Context,
	cmd workertypes.RefreshSearchCommand) error {
	evt := refreshv1.RefreshSearchCommand{
		SearchID:  cmd.SearchID,
		Query:     cmd.Query,
		Frequency: refreshv1.JobFrequency(cmd.Frequency),
		Timestamp: cmd.Timestamp,
	}

	msg, err := event.New(evt)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	if _, err := a.client.Publish(ctx, a.topicID, msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}
