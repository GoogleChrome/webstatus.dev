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
	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
)

// VCSScannerTaskProcessor defines the interface for scanning code trees.
type VCSScannerTaskProcessor interface {
	ProcessTask(ctx context.Context, task codescantaskv1.CodeScanTaskEvent) error
}

type VCSScannerSubscriberAdapter struct {
	processor       VCSScannerTaskProcessor
	eventSubscriber EventSubscriber
	subscriptionID  string
	router          *event.Router
}

func NewVCSScannerSubscriberAdapter(
	processor VCSScannerTaskProcessor,
	eventSubscriber EventSubscriber,
	subscriptionID string,
) *VCSScannerSubscriberAdapter {
	router := event.NewRouter()

	adapter := &VCSScannerSubscriberAdapter{
		processor:       processor,
		eventSubscriber: eventSubscriber,
		subscriptionID:  subscriptionID,
		router:          router,
	}

	event.Register(router, adapter.handleCodeScanTaskEvent)

	return adapter
}

func (a *VCSScannerSubscriberAdapter) Subscribe(ctx context.Context) error {
	return a.eventSubscriber.Subscribe(ctx, a.subscriptionID, func(ctx context.Context,
		msgID string, data []byte) error {
		return a.router.HandleMessage(ctx, msgID, data)
	})
}

func (a *VCSScannerSubscriberAdapter) handleCodeScanTaskEvent(
	ctx context.Context,
	eventID string,
	event codescantaskv1.CodeScanTaskEvent,
) error {
	slog.InfoContext(ctx, "received code scan task event", "eventID", eventID, "repo", event.RepositoryFullName)

	return a.processor.ProcessTask(ctx, event)
}
