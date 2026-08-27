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
	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
)

// VCSSyncPublisherAdapter publishes code scan task events to Pub/Sub on behalf of the VCS sync workflow.
type VCSSyncPublisherAdapter struct {
	client  EventPublisher
	topicID string
}

// NewVCSSyncPublisherAdapter constructs a new VCSSyncPublisherAdapter instance.
func NewVCSSyncPublisherAdapter(client EventPublisher, topicID string) *VCSSyncPublisherAdapter {
	return &VCSSyncPublisherAdapter{
		client:  client,
		topicID: topicID,
	}
}

// PublishCodeScanTask enqueues a code scan task event to the designated Pub/Sub topic.
func (p *VCSSyncPublisherAdapter) PublishCodeScanTask(
	ctx context.Context,
	task codescantaskv1.CodeScanTaskEvent,
) error {
	msg, err := event.New(task)
	if err != nil {
		return fmt.Errorf("failed to create code scan task event: %w", err)
	}

	id, err := p.client.Publish(ctx, p.topicID, msg)
	if err != nil {
		return fmt.Errorf("failed to publish code scan task: %w", err)
	}

	slog.InfoContext(ctx, "published code scan task from vcs sync",
		"msgID", id,
		"repo", task.RepositoryFullName,
		"commit", task.CommitSHA)

	return nil
}
