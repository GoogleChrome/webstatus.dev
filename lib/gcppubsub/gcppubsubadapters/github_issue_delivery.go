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
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	githubissuedeliveryv1 "github.com/GoogleChrome/webstatus.dev/lib/event/githubissuedelivery/v1"
)

// GitHubIssueJobDeliverer defines the interface for delivering GitHub issues.
type GitHubIssueJobDeliverer interface {
	ProcessJob(ctx context.Context, job githubissuedeliveryv1.GitHubIssueDeliveryEvent) error
}

type GitHubIssueDeliverySubscriberAdapter struct {
	deliverer       GitHubIssueJobDeliverer
	eventSubscriber EventSubscriber
	subscriptionID  string
	router          *event.Router
}

func NewGitHubIssueDeliverySubscriberAdapter(
	deliverer GitHubIssueJobDeliverer,
	eventSubscriber EventSubscriber,
	subscriptionID string,
) *GitHubIssueDeliverySubscriberAdapter {
	router := event.NewRouter()

	adapter := &GitHubIssueDeliverySubscriberAdapter{
		deliverer:       deliverer,
		eventSubscriber: eventSubscriber,
		subscriptionID:  subscriptionID,
		router:          router,
	}

	event.Register(router, adapter.handleGitHubIssueDeliveryEvent)

	return adapter
}

func (a *GitHubIssueDeliverySubscriberAdapter) Subscribe(ctx context.Context) error {
	return a.eventSubscriber.Subscribe(ctx, a.subscriptionID, func(ctx context.Context,
		msgID string, data []byte) error {
		return a.router.HandleMessage(ctx, msgID, data)
	})
}

func (a *GitHubIssueDeliverySubscriberAdapter) handleGitHubIssueDeliveryEvent(
	ctx context.Context,
	eventID string,
	event githubissuedeliveryv1.GitHubIssueDeliveryEvent,
) error {
	slog.InfoContext(ctx, "received github issue delivery event", "eventID", eventID, "deliveryID", event.DeliveryID)

	return a.deliverer.ProcessJob(ctx, event)
}
